# Kimi SDK Agent Indexer

多文档一致性管理与动态索引维护服务。

监控文件变化，利用 AI 生成渐进式披露的索引文档，供 Agent 高效探索。

适用于代码、文档、配置、笔记等任意文件集合。

## 模块

| 模块 | 职责 |
|------|------|
| **Core** | 监听文件变化 → 积攒变更 → 调用 Kimi → 更新 index 文件夹 |
| **MCP** | 只读接口，供 Agent 探索 index 内容 |

## 快速开始

```bash
# 安装
npm install -g kimi-agent-indexer

# 启动索引服务（监控当前目录）
kimi-indexer start

# 仅启动 MCP 服务（供 Agent 使用）
kimi-indexer mcp --index-path .kimi-index
```

## MCP 配置

### IDE / Cursor

```json
{
  "mcpServers": {
    "project-index": {
      "command": "kimi-indexer",
      "args": ["mcp", "--index-path", "/path/to/project/.kimi-index"]
    }
  }
}
```

### Kimi CLI

```yaml
mcp:
  servers:
    project-index:
      command: kimi-indexer
      args: ["mcp", "--index-path", "/path/to/project/.kimi-index"]
```

## 文档

- [设计文档](./DESIGN.md) - 架构设计和实现计划

## License

MIT
