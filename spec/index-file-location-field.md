# Index 文件位置引用字段设计

## 背景

当前 `.memo/index/` 下的四个文件（arch.json, interface.json, stories.json, issues.json）描述了代码库的语义信息，但缺少指向源代码位置的引用。这导致：

1. AI Agent 无法快速定位到相关源代码
2. 无法验证索引内容与代码的一致性
3. 缺少从语义到代码的导航能力

## 目标

为每个索引条目添加位置引用字段，支持多种粒度：

| 粒度 | 示例 | 用途 |
|------|------|------|
| 文件夹 | `cmd/` | 模块级别定位 |
| 单文件 | `analyzer/watcher.go` | 精确单文件 |
| 多文件 | `["cmd/watch.go", "cmd/scan.go"]` | 精确多文件 |
| 文件+行范围 | `lines: [30, 54]` | 函数/类级别 |
| 文件+符号 | `symbol: "NewWatcher"` | 符号定位 |

---

## 字段规范

### 通用字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `source` | `string \| string[]` | 主要源码位置（目录/文件/多文件） |
| `refs` | `Ref[]` | 精确引用列表（可选） |
| `locations` | `Location[]` | issues.json 专用（保持现有命名） |

### Ref 对象结构

```typescript
interface Ref {
  file?: string;                      // 文件路径（可省略，继承 source）
  symbol?: string;                    // 函数/类/变量名
  lines?: [number, number] | number;  // 行范围或单行
}
```

### Location 对象结构（issues.json 专用）

```typescript
interface Location {
  file: string;                       // 文件路径（必需）
  symbol?: string;                    // 函数/类/变量名（替代原 keyword）
  lines?: [number, number] | number;  // 行范围或单行（替代原 line）
}
```

---

## 格式修改

### arch.json

**现有格式：**
```json
{
  "modules": [
    {
      "name": "cmd",
      "description": "CLI command layer...",
      "interfaces": "Execute()...",
      "internal": {
        "submodules": [
          {
            "name": "watch",
            "description": "Watch mode...",
            "interfaces": "watchCmd..."
          }
        ],
        "relationships": "root → watch/scan..."
      }
    }
  ]
}
```

**新格式（新增 `source` 和 `refs`）：**
```json
{
  "modules": [
    {
      "name": "cmd",
      "description": "CLI command layer...",
      "interfaces": "Execute()...",
      "source": "cmd/",
      "internal": {
        "submodules": [
          {
            "name": "watch",
            "description": "Watch mode...",
            "interfaces": "watchCmd...",
            "source": "cmd/watch.go",
            "refs": [
              {"symbol": "runWatch", "lines": [37, 146]}
            ]
          }
        ],
        "relationships": "root → watch/scan..."
      }
    }
  ]
}
```

**字段位置：**
- `source`: 添加到 module 和 submodule 层级
- `refs`: 添加到 module 和 submodule 层级（可选）

---

### interface.json

**现有格式：**
```json
{
  "external": [
    {
      "type": "cli",
      "name": "memo watch",
      "params": "-p/--path...",
      "description": "Watch mode..."
    }
  ],
  "internal": [
    {
      "type": "callback",
      "name": "analyzer.NewWatcher(...)",
      "params": "root, ignore...",
      "description": "Creates filesystem watcher..."
    }
  ]
}
```

**新格式（新增 `source` 和 `refs`）：**
```json
{
  "external": [
    {
      "type": "cli",
      "name": "memo watch",
      "params": "-p/--path...",
      "description": "Watch mode...",
      "source": "cmd/watch.go",
      "refs": [
        {"symbol": "runWatch"}
      ]
    }
  ],
  "internal": [
    {
      "type": "callback",
      "name": "analyzer.NewWatcher(...)",
      "params": "root, ignore...",
      "description": "Creates filesystem watcher...",
      "source": "analyzer/watcher.go",
      "refs": [
        {"symbol": "NewWatcher", "lines": [56, 76]}
      ]
    }
  ]
}
```

**字段位置：**
- `source`: 添加到 external 和 internal 条目
- `refs`: 添加到 external 和 internal 条目（可选）

---

### stories.json

**现有格式：**
```json
{
  "stories": [
    {
      "title": "Watch Mode: Continuous File Monitoring",
      "tags": ["watch", "fsnotify", "debounce"],
      "content": "Developer runs 'memo'..."
    }
  ]
}
```

**新格式（新增 `source`）：**
```json
{
  "stories": [
    {
      "title": "Watch Mode: Continuous File Monitoring",
      "tags": ["watch", "fsnotify", "debounce"],
      "content": "Developer runs 'memo'...",
      "source": [
        "cmd/watch.go",
        "analyzer/watcher.go",
        "analyzer/analyser.go"
      ]
    }
  ]
}
```

**字段位置：**
- `source`: 添加到 story 条目，通常为数组（故事涉及多文件）

---

### issues.json

**现有格式：**
```json
{
  "issues": [
    {
      "tags": ["todo"],
      "title": "--mcp-with-watcher flag not implemented",
      "description": "...",
      "locations": [
        {"file": "cmd/watch.go", "keyword": "runWatch", "line": 37}
      ]
    }
  ]
}
```

**新格式（改进 `locations`）：**
```json
{
  "issues": [
    {
      "tags": ["todo"],
      "title": "--mcp-with-watcher flag not implemented",
      "description": "...",
      "locations": [
        {
          "file": "cmd/watch.go",
          "symbol": "runWatch",
          "lines": [37, 50]
        }
      ]
    }
  ]
}
```

**字段变更：**
- `keyword` → `symbol`（更明确的命名）
- `line` → `lines`（支持 `[start, end]` 范围或单个数字）

---

## 字段汇总表

| 文件 | 条目层级 | `source` | `refs` / `locations` |
|------|----------|----------|----------------------|
| arch.json | module | ✅ 目录或文件 | ✅ refs（可选） |
| arch.json | submodule | ✅ 文件 | ✅ refs（可选） |
| interface.json | external | ✅ 文件 | ✅ refs（可选） |
| interface.json | internal | ✅ 文件 | ✅ refs（可选） |
| stories.json | story | ✅ 文件数组 | ❌ 不需要 |
| issues.json | issue | ❌ 不需要 | ✅ locations（已有，改进格式） |

---

## 实施步骤

1. **更新 JSON Schema**：修改 `analyzer/validator.go` 中的 schema 定义
2. **更新 AI Prompt**：修改 `analyzer/prompts/*.md` 让 AI 生成位置信息
3. **迁移现有数据**：issues.json 的 `keyword` → `symbol`, `line` → `lines`
4. **更新 MCP 查询**：支持按位置查询/过滤

---

## 待确认

1. ~~字段命名~~：已确定 `source` + `refs` / `locations`
2. ~~行号格式~~：已确定 `[start, end]` 数组或单个数字
3. **向后兼容**：issues.json 旧格式 `keyword`/`line` 是否仍支持？
4. **必填 vs 可选**：`source` 是否为必填字段？
