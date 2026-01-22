# MCP 挂载测试指南

本指南介绍如何将 Lightkeeper 的 MCP Server 挂载到不同的 AI 客户端进行测试。

## 📋 MCP Server 概述

Lightkeeper 的 MCP Server 提供三个核心工具：

| 工具 | 功能 | 用途 |
|------|------|------|
| `list_files` | 列出文件和目录 | 探索代码结构 |
| `read_file` | 读取文件内容 | 查看源码 |
| `grep_files` | 搜索文件内容 | 查找关键代码 |

---

## 🔧 准备工作

### 1. 构建 MCP Server

```bash
cd mcp
npm install
npm run build

# 验证构建
ls -la dist/
# 应该看到 index.js 和 index.d.ts
```

### 2. 测试 MCP Server 是否正常

```bash
# 直接运行测试（会通过 stdio 通信）
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | node mcp/dist/index.js --root-path ./testdata/django-repo

# 或者使用 npx 运行
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | npx --yes mcp-inspector node mcp/dist/index.js --root-path ./testdata/django-repo
```

---

## 🚀 方式 1：Kimi CLI 挂载

### 快速添加

```bash
# 添加 MCP Server（使用绝对路径）
kimi mcp add --transport stdio lightkeeper -- \
  node /path/to/your/project/mcp/dist/index.js \
  --root-path /path/to/your/project/testdata/django-repo

# 验证添加成功
kimi mcp list

# 测试连接
kimi mcp test lightkeeper
```

### 配置文件方式

创建或编辑 `~/.kimi/mcp.json`：

```json
{
  "mcpServers": {
    "lightkeeper": {
      "command": "node",
      "args": [
        "/path/to/your/project/mcp/dist/index.js",
        "--root-path",
        "/path/to/your/project/testdata/django-repo"
      ]
    }
  }
}
```

### 临时加载配置

```bash
# 创建项目专用配置
cat > ./testdata/mcp-config.json << 'EOF'
{
  "mcpServers": {
    "lightkeeper": {
      "command": "node",
      "args": [
        "./mcp/dist/index.js",
        "--root-path",
        "./testdata/django-repo"
      ]
    }
  }
}
EOF

# 启动 Kimi CLI 并加载配置
kimi --mcp-config-file ./testdata/mcp-config.json
```

### 在 Kimi CLI 中使用

启动后，输入 `/mcp` 查看已加载的工具：

```
/mcp

# 应该看到：
# lightkeeper:
#   - list_files: List files and directories
#   - read_file: Read text file content  
#   - grep_files: Search for pattern in files
```

测试对话：

```
> 请使用 lightkeeper 工具列出 django 目录下的文件

> 搜索包含 "FileUpload" 的文件

> 读取 django/conf/global_settings.py 文件的内容
```

---

## 🖥️ 方式 2：IDE 挂载

编辑 IDE 配置文件：

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "lightkeeper": {
      "command": "node",
      "args": [
        "/path/to/your/project/mcp/dist/index.js",
        "--root-path",
        "/path/to/your/project/testdata/django-repo"
      ]
    }
  }
}
```

重启 IDE 后，在工具列表中应该能看到 `lightkeeper` 的三个工具。

---

## ⌨️ 方式 3：Cursor 挂载

在 Cursor 设置中配置 MCP：

1. 打开 Cursor 设置 (`Cmd+,` 或 `Ctrl+,`)
2. 搜索 "MCP"
3. 添加服务器配置：

```json
{
  "mcpServers": {
    "lightkeeper": {
      "command": "node",
      "args": [
        "/path/to/your/project/mcp/dist/index.js",
        "--root-path",
        "/path/to/your/project/testdata/django-repo"
      ]
    }
  }
}
```

---

## 🧪 方式 4：直接 stdio 测试

### 使用 MCP Inspector（推荐）

```bash
# 安装 MCP Inspector
npm install -g mcp-inspector

# 启动 Inspector（交互式 UI）
mcp-inspector node mcp/dist/index.js --root-path ./testdata/django-repo
```

这会打开一个 Web UI，可以直观地测试各个工具。

### 手动 JSON-RPC 测试

```bash
# 测试 list_files
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_files","arguments":{"path":"django","depth":2}}}' | \
  node mcp/dist/index.js --root-path ./testdata/django-repo

# 测试 grep_files
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"grep_files","arguments":{"pattern":"FILE_UPLOAD","path":"django/conf"}}}' | \
  node mcp/dist/index.js --root-path ./testdata/django-repo

# 测试 read_file
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"django/conf/global_settings.py","offset":300,"limit":20}}}' | \
  node mcp/dist/index.js --root-path ./testdata/django-repo
```

---

## 🎯 SWE-bench 集成测试

### 测试场景：解决 Django Issue

使用 SWE-bench 的示例 `django__django-10914`：

**问题描述**：设置默认的 `FILE_UPLOAD_PERMISSION` 为 `0o644`

**测试流程**：

```bash
# 1. 启动 Kimi CLI 并挂载 MCP
kimi --mcp-config-file ./testdata/mcp-config.json

# 2. 在对话中提问
```

```
我需要解决一个 Django issue：

问题：FILE_UPLOAD_PERMISSIONS 的默认权限不一致，需要设置默认值为 0o644。

请帮我：
1. 使用 grep_files 搜索 "FILE_UPLOAD" 相关代码
2. 找到需要修改的文件
3. 读取相关文件内容
4. 给出修复建议
```

**预期结果**：

AI 应该能够：
1. 通过 `grep_files` 找到 `django/conf/global_settings.py`
2. 通过 `read_file` 读取相关配置
3. 建议修改 `FILE_UPLOAD_PERMISSIONS` 的默认值

---

## 📊 性能测试

### 大规模文件扫描

```bash
# 测试扫描整个 Django 代码库的性能
time echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_files","arguments":{"path":".","depth":3}}}' | \
  node mcp/dist/index.js --root-path ./testdata/django-repo
```

### 大文件读取

```bash
# 测试读取大文件
time echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"django/db/models/query.py","offset":0,"limit":1000}}}' | \
  node mcp/dist/index.js --root-path ./testdata/django-repo
```

---

## 🔍 调试技巧

### 启用详细日志

```bash
# 设置 DEBUG 环境变量
DEBUG=* node mcp/dist/index.js --root-path ./testdata/django-repo
```

### 检查 MCP 通信

```bash
# 使用 tee 记录通信内容
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | \
  tee /dev/stderr | \
  node mcp/dist/index.js --root-path ./testdata/django-repo | \
  tee /dev/stderr | \
  jq .
```

### 常见问题

| 问题 | 解决方案 |
|------|---------|
| "command not found: node" | 确保 Node.js 在 PATH 中 |
| 工具不显示 | 检查 `--root-path` 是否正确 |
| 权限错误 | 检查目录读取权限 |
| 超时 | 减小 `depth` 或 `limit` 参数 |

---

## 📁 项目结构

```
testdata/
├── README.md                 # Django 测试场景说明
├── SWE-BENCH-GUIDE.md       # SWE-bench 工具链指南
├── MCP-TESTING-GUIDE.md     # 本文档
├── mcp-config.json          # MCP 配置文件
├── django-repo/             # Django 代码库
└── swebench-samples/        # SWE-bench 示例数据
```
