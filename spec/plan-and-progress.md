# Milestone Plan

## 总览

| 状态 | Spec | 描述 |
|------|------|------|
| ✅ | feature-mcp-default | MCP + Watcher 模式 |
| ✅ | feature-mcp-query | MCP JSON 查询接口 |
| ✅ | feature-once-mode | 单次分析模式 |
| ✅ | feature-thorough-analysis | 逐模块分析 |
| ✅ | feature-watcher-single-instance | Watcher 单实例锁 + 状态感知 |
| ✅ | fix-dyld-zombie-process | macOS 僵尸进程修复 |
| ✅ | fix-session-id-pollution | Session ID 固定化 |
| ⚠️ | arch-internal-submodules | 已实现，缺测试用例 |
| ⚠️ | fix-large-codebase-context-overflow | 已实现，缺大型仓库测试 |
| ⚠️ | fix-mcp-infinite-recursion | 已实现，缺文档和测试 |
| ⚠️ | line-buffer-design | 已实现，超时参数不可配置 |
| ❌ | feature-concurrent-analysis-guard | 未实现 |

---

## 未完成 TODO

### P0 - 功能缺失

#### feature-concurrent-analysis-guard
防止并发 Flush 导致的分析冲突。

- [ ] `watcher.go`: Add `analysing sync.Mutex`
- [ ] Test concurrent flush scenario

**影响**: 当 Timer1 的分析还在进行时，Timer2 触发的 Flush 会导致并发写入 index 文件。

---

### P1 - 测试缺失

#### fix-large-codebase-context-overflow
- [ ] Test with large codebase (15000+ files)

#### fix-mcp-infinite-recursion
- [ ] Test: Verify no recursion when memo is in `~/.kimi/mcp.json`
- [ ] Test: Verify watcher analysis still works correctly

#### arch-internal-submodules
- [ ] `testdata/` - 添加测试用例（可选）

---

### P2 - 文档/配置

#### fix-mcp-infinite-recursion
- [ ] `README.md`: Document `.memo/mcp.json` for user customization

#### line-buffer-design
- [ ] 可配置的超时参数 (当前硬编码 500ms)

---

## 已完成功能清单

### 核心功能
- [x] 文件监听 + 去抖动
- [x] AI 分析 + 自动更新 index
- [x] MCP 服务 (stdio)
- [x] `--once` 单次分析模式
- [x] `--mcp-with-watcher` 组合模式

### 稳定性
- [x] Watcher 单实例锁 (`flock`)
- [x] 分析状态感知 (`status.json`)
- [x] Session ID 固定化 (防 session 污染)
- [x] 本地 MCP 配置 (防递归)
- [x] 大文件批量处理 (防 context overflow)
- [x] 行缓冲输出 (整行日志)

### 修复
- [x] dyld 僵尸进程 (rm before cp)
- [x] 相对路径 (节省 tokens)

---

## 下一步计划

1. **实现 concurrent-analysis-guard** — 最重要的稳定性问题
2. **补充测试** — 大型仓库测试、递归测试
3. **可配置超时** — line buffer timeout 可配置化
