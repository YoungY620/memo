# Core 模块设计文档

## 概述

Core 模块是 indexer 的运行时核心，负责监控工作目录的文件变化、维护最新索引，以及协调分析与持久化流程。模块启动后会先执行初始化流程，随后进入持续的文件监听；当外部发出停止指令时再优雅退出。

## 设计目标

- **索引一致性**：索引与工作目录状态保持同步，任何变更都能快速反映。
- **稳定性**：文件事件高频时不丢失、不重复处理，异常时可恢复。
- **可观测性**：流程状态、统计数据可以被上层调用方读取。
- **易拓展**：新增分析器或存储后端时，需要的侵入最小。

## 核心职责与角色

| 组件 | 主要职责 |
| --- | --- |
| `CoreRuntime` | 管理生命周期（initialize → watch → shutdown），协调其他组件。 |
| `WorkspaceResolver` | 解析和校验目标目录、生成绝对路径、处理忽略规则。 |
| `IndexStore` | 负责 `.kimi-indexer` 目录的管理、索引数据的读写，以及备份恢复。 |
| `Analyzer` | 根据文件内容或变化类型生成索引条目（可插拔）。 |
| `FileWatcher` | 订阅文件系统事件，统一转为内部的 `ChangeEvent`。 |
| `EventBuffer` | 对事件做去抖、合并、批处理，保障高压场景下的稳定。 |
| `Executor` | 将事件委派给分析器并更新 `IndexStore`，并处理错误重试。 |

## 生命周期

```
Start
 └── Initialize
       ├── Resolve workspace & config
       ├── Prepare index storage (.kimi-indexer, backup)
       ├── Build baseline index snapshot
       └── Warm-up analyzers
 └── Watch
       ├── Subscribe to file events
       ├── Normalize / buffer / dedupe events
       ├── Dispatch to Analyzer pipeline
       └── Persist updated index
 └── Shutdown (triggered by stop signal)
       ├── Flush in-flight events
       ├── Persist final metadata
       └── Release resources
End
```

### Initialize

初始化阶段始终是 Core 模块的第一步，目标是保证索引目录的正确性并生成基线索引。

1. **加载配置与环境**  
   - 解析启动参数（目标目录、忽略列表、分析器配置等）。  
   - 抽象出 `RuntimeContext`，下游组件从该上下文读取所需设置。

2. **准备索引目录**  
   - 若目标目录存在 `.kimi-indexer`：  
     1. 生成备份名（如 `.kimi-indexer.backup-<timestamp>`）。  
     2. 将原目录内容移动/复制到备份位置。  
     3. 清理旧目录，重新创建空的 `.kimi-indexer`。  
   - 若不存在，则直接创建新目录。
   - 记录备份路径，供后续异常恢复或调试使用。

3. **基线扫描**  
   - 遍历当前工作目录，按忽略规则过滤。  
   - 为每个文件创建初始 `ChangeEvent(type=Create)`，交由 Analyzer 生成索引条目。  
   - `IndexStore` 写入基础快照（元数据、索引数据、版本号）。

4. **组件预热**  
   - 启动 Analyzer 所需的外部依赖（如模型、语言服务）。  
   - 预创建任务执行池，以降低 Watch 阶段首次处理事件的延迟。
5. **构造初始化提示词并驱动 bash 写入**  
   - 使用 `docs/core/prompts/initialize.md` 模板替换 `{{workspace_root}}`、`{{ignore_rules}}` 等占位符，向模型发起全面扫描请求。  
   - 模板明确要求使用 Kimi 的 **bash tool** 直接在磁盘上创建或覆盖 `.kimi-indexer/` 目录，无需在回答中回传文件内容。  
   - 初始化完成后立即运行索引验证器，确认目录结构及 JSON/Markdown 符合 `docs/design/storage-design.md` 规范；失败时在同一会话内反馈错误并重复修复。

#### 初始化索引输出

- `.kimi-indexer/` 目录结构与文件格式严格遵循 `docs/design/storage-design.md`。  
- `_index.md` 至少包含一级标题、Mermaid 组件关系图以及子模块/参考文档表格。  
- `_tags.json`、`_notes.json`、`_activities.json` 需通过对应 JSON Schema 校验，必要时补全空集合结构。  
- 子模块（如 `core/`, `indexer/` 等）按需生成 `_index.md` 与 `_activities.json`，确保交叉引用规则 (`XREF-*`) 可通过。  
- 将模型输出的 `issues`、`nextSteps` 信息持久化或记录日志，供后续 Watch 阶段继续处理。

### Watch

初始化成功后进入监听循环，实现对文件变化的实时响应。

1. **进入等待状态并采集事件**  
   - `WatchLoop` 进入阻塞等待，直到 `EventBuffer` 推送新批次。  
   - `FileWatcher` 通过系统 API（fsnotify、FSEvents 等）订阅事件并写入缓冲，统一转换为 `{path, changeType, metadata}`。

2. **事件缓冲与规整**  
   - `EventBuffer` 负责：  
     - 去抖动：短时间内多次写入合并为一次。  
     - 去重与折叠：同一文件连续 create → modify → delete 时，仅保留最终状态。  
     - 批量触发：根据延迟/数量阈值打包事件，并携带变更文件的完整内容及哈希。
   - 记录节流指标（丢弃事件数、合并数、延迟），供监控使用。

3. **构造 Watch 提示词**  
   - 收到批次后，`PromptBuilder` 根据 `docs/core/prompts/watch.md` 模板替换 `{{changed_files}}`、`{{changed_file_blobs}}`、`{{related_index_files}}` 等占位符。  
   - `related_index_files` 包含所有受影响的 `.kimi-indexer` 文件（根索引、子模块、引用、活动等），确保模型能够阅读并更新。  
   - 模板强调模型必须使用 bash tool 实际修改 `.kimi-indexer/`，并在回复中提供命令摘要与剩余问题。

4. **LLM + 验证循环**  
   - 发送初始提示后，立即运行索引验证器（按 `docs/design/storage-design.md` 检查，遇到首个错误即返回）。  
   - 若验证失败，构造 “验证失败修复 Prompt” 将错误详情（规则 ID、文件路径、报错信息）反馈给模型，要求继续通过 bash tool 修复；在同一会话内重试，最多 100 次。  
   - 若超出重试次数仍未通过，记录致命错误并上报控制层。

5. **状态更新与日志**  
   - 验证通过视为本轮 Watch 成功，记录批次 ID、变更数、执行命令摘要（来自模型回复）等信息。  
   - `issues`、`nextSteps` 等模型输出按需写入日志或外部监控，以便后续人工介入或自动化流程处理遗留问题。

6. **异常处理与降级**  
   - 如果 LLM 会话在 100 次内未通过验证，或执行链路发生致命错误，记录错误上下文并将事件批次重新入队（带退避策略）或升级为人工介入。  
   - 当工作目录被移动、权限收回或索引目录损坏时，触发 `WorkspaceInvalid` 状态，暂停 Watch 循环并通知上层控制层。

### Shutdown

当用户请求停止监控服务时，Core 模块需要优雅退出。

1. 通知 `FileWatcher` 停止接受新事件，`EventBuffer` 停止入队。  
2. 等待队列处理完成，确保无 in-flight 任务。  
3. `IndexStore` 写入最终状态（快照版本、最后更新时间、统计数据）。  
4. 保留最近一次备份路径，供上层组件决定是否清理或回滚。

## 数据结构（概要）

```go
type ChangeEvent struct {
    Path       string
    Type       ChangeType // Create | Modify | Delete | Rename
    Timestamp  time.Time
    Extra      map[string]string
}

type IndexRecord struct {
    Path      string
    Hash      string
    Metadata  map[string]string
    Payload   []byte // Analyzer 生成的索引内容
}
```

索引目录 `.kimi-indexer` 采用以下结构（默认约定，可在 `IndexStore` 中扩展）：

```
.kimi-indexer/
  ├── index.db         # 主索引（可为 BoltDB、SQLite 或 JSON 序列化文件）
  ├── meta.json        # 快照元数据（版本、构建时间、统计信息）
  ├── logs/            # 错误与诊断日志
  └── backups/         # 可选，额外存放增量备份
```

## 配置与可观测性

- **配置来源**：命令行参数、环境变量、配置文件。核心参数包括目标目录、忽略模式、批量大小、节流时间窗口、Analyzer 列表等。
- **监控指标**（可通过 `Stats()` 暴露）：  
  - 当前索引版本、记录数  
  - 事件吞吐（每秒事件数）、缓冲延迟  
  - 错误率（失败次数、重试次数、死信条目）  
  - 备份次数、最近一次备份路径
- **日志**：核心流程（初始化、备份、扫描、重试、退出）打印结构化日志，便于调试。

## 未来扩展

- 多工作目录并行索引与隔离策略。  
- 远程存储后端（如云数据库或对象存储）支持。  
- 更丰富的分析插件体系（语义理解、增量语法树等）。  
- 快照差异比较与历史版本回滚工具。
