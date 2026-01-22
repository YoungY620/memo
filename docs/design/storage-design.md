# 索引存储格式规范与验证器设计

本文档定义 `.kimi-index/` 目录的存储格式规范，以及可通过静态分析验证的约束条件。

---

## 1. 目录结构规范

### 1.1 根目录结构

```
.kimi-index/
├── _index.md              # [必需] 根索引文件
├── _tags.json             # [必需] 全局标签定义（仅根目录）
├── _notes.json            # [必需] 闪记笔记（仅根目录）
├── _activities.json       # [必需] 活动追踪
├── _reference/            # [可选] 详细参考文档目录
│   └── *.md
└── <submodule>/           # [可选] 子模块目录（递归结构）
    ├── _index.md          # [必需] 子模块索引
    ├── _activities.json   # [必需] 子模块活动
    └── _reference/        # [可选] 子模块参考文档
```

### 1.2 命名规则

| 规则 | 描述 | 模式 |
|------|------|------|
| 系统文件 | 以 `_` 开头 | `^_[a-z]+\.(md\|json)$` |
| 子模块目录 | 不以 `_` 开头，小写 | `^[a-z][a-z0-9-]*$` |
| 参考文档 | 小写字母、数字、连字符 | `^[a-z][a-z0-9-]*\.md$` |

---

## 2. JSON Schema 验证

所有 JSON 文件格式通过 JSON Schema 验证，Schema 文件位于 `schemas/` 目录。

### 2.1 Schema 文件列表

| 文件 | Schema | 描述 |
|------|--------|------|
| `_tags.json` | `schemas/tags.schema.json` | 全局标签定义 |
| `_notes.json` | `schemas/notes.schema.json` | 闪记笔记 |
| `_activities.json` | `schemas/activities.schema.json` | 活动追踪 |

### 2.2 Schema 覆盖的验证规则

以下规则由 JSON Schema 自动验证：

| 规则 ID | 文件 | 描述 | Schema 约束 |
|---------|------|------|-------------|
| `JSON-001` | 所有 JSON | 必须是有效 JSON | JSON 解析 |
| `JSON-002` | `_tags.json` | 必须是字符串数组 | `type: array, items.type: string` |
| `JSON-003` | `_tags.json` | 标签格式：小写字母开头 | `pattern: ^[a-z][a-z0-9-]*$` |
| `JSON-004` | `_tags.json` | 无重复标签 | `uniqueItems: true` |
| `JSON-005` | `_tags.json` | 最多 100 个标签 | `maxItems: 100` |
| `JSON-006` | `_notes.json` | 必须是对象数组 | `type: array, items.type: object` |
| `JSON-007` | `_notes.json` | `content` 字段必需 | `required: ["content"]` |
| `JSON-008` | `_notes.json` | `id` 为 UUID 格式 | `format: uuid` |
| `JSON-009` | `_notes.json` | 时间戳为 ISO 8601 | `format: date-time` |
| `JSON-010` | `_notes.json` | 最多 50 条笔记 | `maxItems: 50` |
| `JSON-011` | `_activities.json` | 顶层为对象 | `type: object` |
| `JSON-012` | `_activities.json` | 每个 type 包含 `items` 和 `children` | `required: ["items", "children"]` |
| `JSON-013` | `_activities.json` | `items` 是对象数组 | `type: array` |
| `JSON-014` | `_activities.json` | `children` 是字符串数组 | `type: array, items.type: string` |
| `JSON-015` | `_activities.json` | activity type 名称格式 | `propertyNames.pattern` |
| `JSON-016` | `_activities.json` | 最多 100 个 type | `maxProperties: 100` |
| `JSON-017` | 所有 JSON | 不允许额外字段 | `additionalProperties: false` |

### 2.3 使用方式

```bash
# 使用 ajv-cli 验证
ajv validate -s schemas/tags.schema.json -d .kimi-index/_tags.json
ajv validate -s schemas/notes.schema.json -d .kimi-index/_notes.json
ajv validate -s schemas/activities.schema.json -d .kimi-index/_activities.json

# 或在代码中使用
import "github.com/xeipuuv/gojsonschema"
```

---

## 3. 额外验证规则（Schema 无法覆盖）

以下规则需要自定义验证逻辑：

### 3.1 结构规则（STRUCT）

| 规则 ID | 级别 | 描述 | 验证方式 |
|---------|------|------|----------|
| `STRUCT-001` | error | 根目录必须存在 `_index.md` | `os.Stat()` |
| `STRUCT-002` | error | 根目录必须存在 `_tags.json` | `os.Stat()` |
| `STRUCT-003` | error | 根目录必须存在 `_notes.json` | `os.Stat()` |
| `STRUCT-004` | error | 根目录必须存在 `_activities.json` | `os.Stat()` |
| `STRUCT-005` | error | 子模块必须存在 `_index.md` | 递归检查 |
| `STRUCT-006` | error | 子模块必须存在 `_activities.json` | 递归检查 |
| `STRUCT-007` | error | 子模块不得存在 `_tags.json` | 递归检查 |
| `STRUCT-008` | error | 子模块不得存在 `_notes.json` | 递归检查 |
| `STRUCT-009` | error | 目录名符合命名规则 | 正则匹配 |

### 3.2 Markdown 规则（MD）

| 规则 ID | 级别 | 描述 | 验证方式 |
|---------|------|------|----------|
| `MD-001` | error | `_index.md` 必须包含一级标题 | `^# .+` 正则 |
| `MD-002` | warning | `_index.md` 应包含 Mermaid 图表 | ` ```mermaid` 检测 |
| `MD-003` | warning | Mermaid 语法有效 | Mermaid 解析器 |
| `MD-004` | error | 文件必须是有效 UTF-8 | 编码检测 |
| `MD-005` | warning | 行尾无多余空格 | 正则检查 |

### 3.3 交叉引用规则（XREF）

| 规则 ID | 级别 | 描述 | 验证方式 |
|---------|------|------|----------|
| `XREF-001` | error | `_index.md` 中引用的子模块目录必须存在 | 链接解析 + 路径检查 |
| `XREF-002` | error | `_index.md` 中引用的 `_reference/*.md` 必须存在 | 链接解析 + 路径检查 |
| `XREF-003` | warning | `_reference/*.md` 应被 `_index.md` 引用 | 反向引用检查 |
| `XREF-004` | error | `_activities.json` 中 `children` 目录必须存在 | 路径检查 |
| `XREF-005` | error | 所有 `tags` 必须在根 `_tags.json` 中定义 | 集合包含检查 |
| `XREF-006` | error | `_activities.json` 中 `file` 路径有效（若存在） | 路径检查 |

---

## 4. 验证器设计

### 4.1 验证流程

```
┌─────────────────────────────────────────────────────────────┐
│                      Validator                               │
├─────────────────────────────────────────────────────────────┤
│  1. 结构检查 (STRUCT-*)                                      │
│     └─ 检查必需文件/目录存在性                                │
│                                                              │
│  2. JSON Schema 验证 (JSON-*)                                │
│     └─ 使用 schemas/*.schema.json 验证所有 JSON 文件         │
│                                                              │
│  3. Markdown 检查 (MD-*)                                     │
│     └─ 检查标题、Mermaid、编码等                              │
│                                                              │
│  4. 交叉引用检查 (XREF-*)                                    │
│     └─ 检查链接有效性、标签定义、路径存在性                    │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 输入输出

```go
// 输入
type ValidateInput struct {
    IndexPath   string   // .kimi-index 目录路径
    SchemaDir   string   // schemas 目录路径
    StrictMode  bool     // 严格模式：warning 也导致失败
    RuleFilter  []string // 仅检查指定规则前缀，如 ["STRUCT", "XREF"]
}

// 输出
type ValidateResult struct {
    Valid  bool              // 是否通过验证
    Errors []ValidationError // 错误列表
}

type ValidationError struct {
    RuleID     string // 规则 ID，如 "STRUCT-001"
    Severity   string // "error" | "warning" | "info"
    File       string // 相关文件路径
    Line       int    // 行号（如适用）
    Message    string // 错误描述
    Suggestion string // 修复建议
}
```

### 4.3 验证器伪代码

```go
func Validate(input ValidateInput) ValidateResult {
    result := ValidateResult{Valid: true}
    
    // 1. 结构验证
    result.Merge(validateStructure(input.IndexPath))
    
    // 2. 加载全局标签（供后续交叉引用检查）
    tags, err := loadTags(filepath.Join(input.IndexPath, "_tags.json"))
    if err != nil {
        result.AddError("STRUCT-002", err)
    }
    
    // 3. JSON Schema 验证
    schemaFiles := map[string]string{
        "_tags.json":       "tags.schema.json",
        "_notes.json":      "notes.schema.json",
        "_activities.json": "activities.schema.json",
    }
    result.Merge(validateJSONSchemas(input.IndexPath, input.SchemaDir, schemaFiles))
    
    // 4. 递归验证每个目录
    result.Merge(validateDirectoryRecursive(input.IndexPath, tags, true))
    
    // 5. 交叉引用验证
    result.Merge(validateCrossReferences(input.IndexPath, tags))
    
    return result
}

func validateJSONSchemas(indexPath, schemaDir string, schemas map[string]string) ValidateResult {
    result := ValidateResult{Valid: true}
    
    for jsonFile, schemaFile := range schemas {
        jsonPath := filepath.Join(indexPath, jsonFile)
        schemaPath := filepath.Join(schemaDir, schemaFile)
        
        // 使用 gojsonschema 验证
        schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
        documentLoader := gojsonschema.NewReferenceLoader("file://" + jsonPath)
        
        validationResult, err := gojsonschema.Validate(schemaLoader, documentLoader)
        if err != nil {
            result.AddError("JSON-001", jsonFile, err.Error())
            continue
        }
        
        for _, desc := range validationResult.Errors() {
            result.AddError("JSON-XXX", jsonFile, desc.String())
        }
    }
    
    return result
}

func validateCrossReferences(indexPath string, globalTags []string) ValidateResult {
    result := ValidateResult{Valid: true}
    tagSet := toSet(globalTags)
    
    // 遍历所有 _activities.json 和 _notes.json 检查标签引用
    filepath.Walk(indexPath, func(path string, info os.FileInfo, err error) error {
        if info.Name() == "_activities.json" || info.Name() == "_notes.json" {
            usedTags := extractTags(path)
            for _, tag := range usedTags {
                if !tagSet[tag] {
                    result.AddError("XREF-005", path, 
                        fmt.Sprintf("Tag '%s' not defined in _tags.json", tag))
                }
            }
        }
        return nil
    })
    
    return result
}
```

---

## 5. 命令行接口

```bash
# 验证索引目录
kimi-indexer validate [--path .kimi-index] [--schema ./schemas]

# 输出格式
kimi-indexer validate --format json    # JSON 格式输出
kimi-indexer validate --format text    # 人类可读格式（默认）

# 严格模式（warning 也导致失败）
kimi-indexer validate --strict

# 仅检查特定规则
kimi-indexer validate --rules STRUCT,XREF

# 忽略特定规则
kimi-indexer validate --skip JSON-003,MD-005
```

**退出码：**

| 码 | 含义 |
|----|------|
| 0 | 验证通过 |
| 1 | 验证失败（存在 error） |
| 2 | 参数/配置错误 |

---

## 6. 规则汇总表

### 6.1 JSON Schema 自动验证 (17 条)

| ID | 文件 | 描述 |
|----|------|------|
| `JSON-001` ~ `JSON-005` | `_tags.json` | 类型、格式、唯一性、数量限制 |
| `JSON-006` ~ `JSON-010` | `_notes.json` | 类型、必需字段、格式、数量限制 |
| `JSON-011` ~ `JSON-017` | `_activities.json` | 类型、必需字段、格式、数量限制 |

### 6.2 结构规则 (9 条)

| ID | 描述 |
|----|------|
| `STRUCT-001` ~ `STRUCT-004` | 根目录必需文件检查 |
| `STRUCT-005` ~ `STRUCT-006` | 子模块必需文件检查 |
| `STRUCT-007` ~ `STRUCT-008` | 子模块禁止文件检查 |
| `STRUCT-009` | 目录命名规则 |

### 6.3 Markdown 规则 (5 条)

| ID | 描述 |
|----|------|
| `MD-001` | 必须包含一级标题 |
| `MD-002` | 应包含 Mermaid 图表 |
| `MD-003` | Mermaid 语法有效 |
| `MD-004` | 有效 UTF-8 编码 |
| `MD-005` | 行尾无多余空格 |

### 6.4 交叉引用规则 (6 条)

| ID | 描述 |
|----|------|
| `XREF-001` ~ `XREF-002` | 链接目标存在性 |
| `XREF-003` | 参考文档被引用 |
| `XREF-004` | children 目录存在 |
| `XREF-005` | 标签已定义 |
| `XREF-006` | file 路径有效 |

---

## 7. 实现检查清单

- [x] JSON Schema 定义 (`schemas/`)
- [ ] 结构验证实现
- [ ] JSON Schema 验证集成
- [ ] Markdown 格式验证
- [ ] 交叉引用验证
- [ ] 命令行接口
- [ ] 错误报告格式化
- [ ] 单元测试
