# Milestone Plan

## 总览

### 已归档 (spec/arch/)

| Spec | 描述 |
|------|------|
| arch-internal-submodules | arch.json internal 子模块 schema |
| feature-auto-update | Git 仓库自动更新 |
| feature-concurrent-analysis-guard | 并发分析保护 (sem channel) |
| feature-mcp-default | MCP + Watcher 模式 |
| feature-mcp-query | MCP JSON 查询接口 |
| feature-scan-mode | Scan mode (single analysis) |
| feature-thorough-analysis | 逐模块分析 |
| feature-watcher-single-instance | Watcher 单实例锁 + 状态感知 |
| fix-dyld-zombie-process | macOS 僵尸进程修复 |
| fix-large-codebase-context-overflow | 大型仓库上下文溢出修复 |
| fix-mcp-infinite-recursion | MCP 无限递归修复 |
| fix-session-id-pollution | Session ID 固定化 |
| line-buffer-design | 行缓冲输出设计 |

### 待完成 (spec/)

| 状态 | Spec | 描述 |
|------|------|------|
| ❌ | feature-future-belongs-to-future | 定期 session 重建机制 |
| ❌ | fix-empty-message-content | 空消息内容修复 |

---

## 未完成 TODO

### P0 - 功能缺失

#### feature-future-belongs-to-future
定期用全新 session 重建 index，避免长期运行导致的 session 状态累积和偏差。

- [ ] 设计评审
- [ ] 实现 Rotator 组件
- [ ] 测试

#### fix-empty-message-content
Session history 中存在空内容消息导致 API 错误。

- [ ] 复现并定位根因
- [ ] 确定解决方案
- [ ] 实现修复
- [ ] 测试

---

## 已完成功能清单

### 核心功能
- [x] 文件监听 + 去抖动
- [x] AI 分析 + 自动更新 index
- [x] MCP 服务 (stdio)
- [x] `memo scan` command (single analysis)
- [x] `memo mcp` command (MCP server)

### 稳定性
- [x] Watcher 单实例锁 (`flock`)
- [x] 分析状态感知 (`status.json`)
- [x] Session ID 固定化 (防 session 污染)
- [x] 本地 MCP 配置 (防递归)
- [x] 大文件批量处理 (防 context overflow)
- [x] 行缓冲输出 (整行日志)
- [x] 并发分析保护 (`sem channel`)

### 修复
- [x] dyld 僵尸进程 (rm before cp)
- [x] 相对路径 (节省 tokens)

---

## 下一步计划

1. **feature-future-belongs-to-future** — 定期 session 重建避免偏差累积
2. **fix-empty-message-content** — 空消息内容修复
