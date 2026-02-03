# Spec 目录说明

## 目录结构

```
spec/
├── README.md                   # 本说明文件
├── spec-template.md            # Spec 模板
├── plan-and-progress.md        # 项目规划和进度
├── note-*.md                   # 设计笔记
│
├── feature-*.md                # 未完成/正在进行的功能 spec
├── fix-*.md                    # 未完成/正在进行的修复 spec
├── index-*.md                  # 未完成/正在进行的索引相关 spec
│
└── arch/                       # 已完成并归档的 spec
    ├── feature-*.md            # 已实现的功能
    ├── fix-*.md                # 已修复的问题
    └── *.md                    # 其他已完成的设计文档
```

## 文件分类

### 📝 模板和文档
- `spec-template.md` - 编写新 spec 的模板
- `plan-and-progress.md` - 项目整体规划和进度跟踪
- `note-*.md` - 设计笔记、思考和决策记录

### 🚧 进行中（spec/）
当前未完成或正在实现的 spec 放在根目录：

- `feature-nested-gitignore.md` - 嵌套 gitignore 支持（正在进行）
- `feature-once-mode.md` - 一次性扫描模式
- `feature-future-belongs-to-future.md` - 未来功能规划
- `fix-empty-message-content.md` - 空消息内容修复
- `index-file-location-field.md` - 索引文件位置字段

### ✅ 已完成（spec/arch/）
已实现并归档的 spec 放在 `arch/` 子目录：

**功能（Features）**
- `feature-auto-update.md` - 自动更新功能
- `feature-concurrent-analysis-guard.md` - 并发分析保护
- `feature-mcp-default.md` - MCP 默认模式
- `feature-mcp-query.md` - MCP 查询功能
- `feature-scan-mode.md` - 扫描模式
- `feature-thorough-analysis.md` - 彻底分析
- `feature-watcher-single-instance.md` - Watcher 单实例

**修复（Fixes）**
- `fix-dyld-zombie-process.md` - dyld 僵尸进程修复
- `fix-large-codebase-context-overflow.md` - 大代码库上下文溢出修复
- `fix-mcp-infinite-recursion.md` - MCP 无限递归修复
- `fix-session-id-pollution.md` - Session ID 污染修复

**架构（Architecture）**
- `arch-internal-submodules.md` - 内部子模块架构
- `line-buffer-design.md` - 行缓冲设计

## 工作流程

### 创建新 spec
1. 复制 `spec-template.md`
2. 在 `spec/` 根目录创建新文件：`feature-xxx.md` 或 `fix-xxx.md`
3. 填写内容，遵循模板结构
4. **重要**：使用通用路径，不要包含真实用户名或敏感信息

### 实现完成后归档
1. 确认功能已完全实现并测试
2. 将 spec 文件移动到 `spec/arch/`：
   ```bash
   git mv spec/feature-xxx.md spec/arch/
   ```
3. 提交归档记录

### Spec 模板规范
编写 spec 时必须遵守：
- ✅ 使用通用路径：`/path/to/project`, `~/.local/bin/`
- ✅ 使用通用用户名：`user` 而非真实用户名
- ❌ 不要包含真实文件路径
- ❌ 不要包含 API keys 或其他敏感信息

## 当前统计

- 未完成 spec: 5 个
- 已归档 spec: 13 个
- 总计: 18 个

---

_最后更新: 2024_
