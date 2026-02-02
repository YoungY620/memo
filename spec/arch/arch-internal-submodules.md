# Spec: arch.json 增加 internal 字段

## 需求

优化 `arch.json` 格式，为每个 module 添加 `internal` 字段，用于描述模块内部结构。

## Schema 变更

```json
{
  "modules": [
    {
      "name": "string",
      "description": "string",
      "interfaces": "string",
      "internal": {
        "submodules": [
          {
            "name": "string",
            "description": "string",
            "interfaces": "string"
          }
        ],
        "relationships": "string"
      }
    }
  ],
  "relationships": "string"
}
```

## 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `internal` | object | 可选，模块内部结构 |
| `internal.submodules` | array | 子模块列表，字段同外层 module（无 internal） |
| `internal.relationships` | string | 子模块间关系，格式同外层 relationships |

## 示例

```json
{
  "name": "api",
  "description": "REST API layer",
  "interfaces": "HTTP endpoints, calls auth and service modules",
  "internal": {
    "submodules": [
      {
        "name": "router",
        "description": "URL routing and middleware chain",
        "interfaces": "Receives requests, dispatches to handlers"
      },
      {
        "name": "handlers",
        "description": "Request handlers for each endpoint",
        "interfaces": "Called by router, returns responses"
      }
    ],
    "relationships": "router → handlers → response"
  }
}
```

## 影响范围

- [x] `prompts/context.md` - 更新 arch.json schema 和示例
- [x] `validator.go` - 更新 JSON schema 验证
- [ ] `testdata/` - 添加测试用例（可选）
