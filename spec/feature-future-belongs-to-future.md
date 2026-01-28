# Feature: Future Belongs to Future (未来属于未来)

定期用全新 session 重建 index，避免长期运行导致的 session 状态累积和偏差。

## Problem

```
Session 启动 ──→ 处理事件 ──→ 处理事件 ──→ ... ──→ 处理事件
                  ↓              ↓                    ↓
              context 累积   偏差累积            可能产生幻觉/遗忘
```

长期运行的 session 可能：
- 累积错误的理解
- 遗忘早期的上下文
- 产生与代码不符的幻觉

## Solution

双 index 轮换机制（类似蓝绿部署）：

```
时间 →

index         [====== 主服务 ======]                    [====== 主服务 ======]
                                    ↘                  ↗
                                     (切换)
                                    ↗                  ↘
index-rebuild        [=== 构建中 ===]                         [=== 构建中 ===]
              
Session A     [===================]
Session B            [===================]
Session C                                 [===================]
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Watcher                             │
│                           │                                 │
│              ┌────────────┴────────────┐                    │
│              ▼                         ▼                    │
│    ┌─────────────────┐       ┌─────────────────┐            │
│    │   Analyser A    │       │   Analyser B    │            │
│    │  (Session A)    │       │  (Session B)    │            │
│    │                 │       │                 │            │
│    │  writes to:     │       │  writes to:     │            │
│    │  .memo/index/   │       │  .memo/index-   │            │
│    │                 │       │    rebuild/     │            │
│    └─────────────────┘       └─────────────────┘            │
│              │                         │                    │
│              ▼                         ▼                    │
│    ┌─────────────────┐       ┌─────────────────┐            │
│    │  .memo/index/   │       │  .memo/index-   │            │
│    │  ├─ arch.json   │       │    rebuild/     │            │
│    │  ├─ interface   │  ←──  │  (定期切换)      │            │
│    │  ├─ stories     │       │                 │            │
│    │  └─ issues      │       │                 │            │
│    └─────────────────┘       └─────────────────┘            │
│                                                             │
│                      ┌──────────┐                           │
│                      │ Rotator  │  ← 定时触发切换            │
│                      └──────────┘                           │
└─────────────────────────────────────────────────────────────┘
```

## Lifecycle

```
Phase 1: 启动
├─ 创建 Analyser A (Session A) → writes to index/
├─ ScanAll → Analyser A 处理全量文件
└─ 启动 rebuild 定时器 (e.g., 30 分钟)

Phase 2: 正常运行
├─ Watcher 监听文件变化
├─ 变化 → Analyser A 更新 index/
└─ MCP 读取 index/

Phase 3: 启动 Rebuild (定时器触发)
├─ 创建 Analyser B (Session B) → writes to index-rebuild/
├─ ScanAll → Analyser B 处理全量文件
└─ 后续变化 → 同时发送给 A 和 B

Phase 4: Rebuild 期间
├─ Analyser A: 更新 index/
├─ Analyser B: 更新 index-rebuild/
└─ MCP 仍读取 index/ (用户无感知)

Phase 5: 切换 (rebuild 完成 + 稳定期后)
├─ 停止 Analyser A
├─ mv index/ index-old/ (备份)
├─ mv index-rebuild/ index/
├─ rm -rf index-old/
└─ Analyser B 现在写入 index/

Phase 6: 新一轮 Rebuild
├─ 创建 Analyser C (Session C) → writes to index-rebuild/
├─ ScanAll → Analyser C 处理全量文件
└─ 循环回到 Phase 4
```

## Configuration

```yaml
# config.yaml
rebuild:
  enabled: true
  interval_minutes: 30      # 多久启动一次 rebuild
  stabilize_minutes: 5      # rebuild 完成后等待多久再切换
```

## Design Details

### Rotator 组件

```go
type Rotator struct {
    watcher        *Watcher
    activeAnalyser *Analyser
    rebuildAnalyser *Analyser
    interval       time.Duration
    stabilize      time.Duration
}

func (r *Rotator) Start() {
    // 定期触发 rebuild
    ticker := time.NewTicker(r.interval)
    for range ticker.C {
        r.startRebuild()
    }
}

func (r *Rotator) startRebuild() {
    // 1. 创建新 Analyser，写入 index-rebuild/
    r.rebuildAnalyser = NewAnalyser(workDir, "index-rebuild")
    
    // 2. ScanAll 触发全量分析
    r.watcher.ScanAllTo(r.rebuildAnalyser)
    
    // 3. 等待稳定期后切换
    time.AfterFunc(r.stabilize, r.rotate)
}

func (r *Rotator) rotate() {
    // 1. 停止旧 Analyser
    r.activeAnalyser.Stop()
    
    // 2. 切换目录
    os.Rename(".memo/index", ".memo/index-old")
    os.Rename(".memo/index-rebuild", ".memo/index")
    os.RemoveAll(".memo/index-old")
    
    // 3. 新 Analyser 成为 active
    r.activeAnalyser = r.rebuildAnalyser
    r.rebuildAnalyser = nil
}
```

### Watcher 双发

```go
func (w *Watcher) Flush() {
    // ... collect files ...
    
    // 发送给 active analyser
    if w.onChange != nil {
        w.onChange(files)
    }
    
    // 如果有 rebuild analyser，也发送
    if w.onChangeRebuild != nil {
        w.onChangeRebuild(files)
    }
}
```

### MCP 读取

MCP 始终读取 `index/`，切换对用户透明：

```go
func (s *Server) handleQuery(path string) {
    // 始终从 .memo/index/ 读取
    // 切换是原子的 (rename)，用户无感知
}
```

## Randomized File Order (随机化文件顺序)

为确保每一代 session 产生不同的理解，需要随机化文件传入顺序：

```
旧一代: [a.go, b.go, c.go] → [d.go, e.go] → [f.go, g.go]
新一代: [c.go, a.go, b.go] → [g.go, f.go] → [e.go, d.go]
                ↑                   ↑            ↑
            batch 内 shuffle    batch 内 shuffle  batch 顺序也 shuffle
```

### 为什么有效？

LLM 对输入顺序敏感：
- 先看到的文件会影响对后续文件的理解
- 不同顺序可能发现不同的关联
- 增加多样性，避免每代产生相同的偏差

### 实现

只需在 `Analyse` 入口处 shuffle：

```go
// analyser.go

import "math/rand"

func (a *Analyser) Analyse(files []string) {
    // 随机化文件顺序，确保每代不同
    rand.Shuffle(len(files), func(i, j int) {
        files[i], files[j] = files[j], files[i]
    })
    
    relFiles := toRelativePaths(files, a.workDir)
    batches := splitIntoBatches(relFiles, maxFilesPerBatch)
    
    // 随机化批次顺序
    rand.Shuffle(len(batches), func(i, j int) {
        batches[i], batches[j] = batches[j], batches[i]
    })
    
    // ... 处理 batches ...
}
```

### 随机源

- Go 1.20+ 默认 `rand` 已自动 seed，无需手动设置
- 每次 rebuild 启动时间不同 → 随机序列不同
- 如需可复现，可在配置中指定 seed

## Benefits

| 好处 | 说明 |
|------|------|
| **Session 新鲜度** | 定期用全新 session，避免累积偏差 |
| **全量重建** | 每次 rebuild 都是全量 ScanAll，纠正增量更新可能的遗漏 |
| **随机化顺序** | 不同文件顺序 → 不同理解 → 多样性 |
| **无缝切换** | 用户/MCP 无感知，始终读取 index/ |
| **容错** | 如果 rebuild 失败，不影响现有 index |

## Edge Cases

| 场景 | 处理 |
|------|------|
| Rebuild 期间 watcher 关闭 | 清理 index-rebuild，下次启动重新开始 |
| Rebuild 失败 | 保留现有 index，记录错误，下个周期重试 |
| 切换时 MCP 正在读取 | rename 是原子操作，要么读旧的要么读新的 |
| Rebuild 期间又触发 rebuild | 忽略，等当前 rebuild 完成 |

## Files

| File | Change |
|------|--------|
| `rotator.go` | 新增：Rotator 组件，管理双 Analyser 生命周期 |
| `watcher.go` | 修改：支持双 onChange 回调 |
| `analyser.go` | 修改：支持指定输出目录 |
| `config.go` | 修改：添加 rebuild 配置 |
| `main.go` | 修改：初始化 Rotator |

## TODO

- [ ] 设计评审：确认机制合理性
- [ ] `analyser.go`: 添加文件顺序随机化 (rand.Shuffle)
- [ ] `analyser.go`: 添加批次顺序随机化
- [ ] `config.go`: 添加 rebuild 配置项
- [ ] `rotator.go`: 实现 Rotator 组件
- [ ] `analyser.go`: 支持指定输出目录 (index vs index-rebuild)
- [ ] `watcher.go`: 支持双 onChange 回调
- [ ] `main.go`: 初始化和启动 Rotator
- [ ] 测试：验证切换过程的原子性
- [ ] 测试：验证 MCP 读取不受影响
