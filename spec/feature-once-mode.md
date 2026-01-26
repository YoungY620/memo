# Feature: Once Mode

一次性扫描模式，执行初始化和分析后直接退出，不进入监听状态。

## 架构设计

```
正常模式:  ScanAll() → pending → timer → flush() → onChange
once模式:  ScanAll() → pending → Flush() → onChange → exit
                                    ↑
                              直接调用，复用同一方法
```

**核心改动**：`flush()` → `Flush()`（公开化），两种模式复用。

## 涉及文件

| 文件 | 改动 |
|------|------|
| `watcher.go` | `flush()` 改为 `Flush()` |
| `main.go` | 添加 `--once` flag，once 模式调用 `Flush()` 后退出 |

## TODO

- [x] `watcher.go`: `flush()` → `Flush()`
- [x] `main.go`: 添加 `--once` flag 及分支逻辑
- [x] 测试验证
