# Lightkeeper 测试数据集

本目录包含用于测试 Lightkeeper 语义索引框架的真实代码库。

## 选择标准

我们选择了 **Django** 作为主要测试场景，它是 [SWE-bench](https://www.swebench.com/) 基准测试中：

- ✅ **数据量最大** 的仓库（约 507,000+ 行代码）
- ✅ **代码规模最复杂** 的项目（2,887 个 Python 文件）
- ✅ **涉及多语言** 调用（Python、JavaScript、HTML、CSS）

## Django 代码库统计

| 指标 | 数值 |
|------|------|
| Python 文件 | 2,887 |
| 总代码行数 | 507,569 |
| JavaScript 文件 | 113 |
| HTML 模板 | 372 |
| CSS 文件 | 48 |
| 测试文件夹数 | 223 |

## Django 架构复杂度

Django 是一个全功能的 Web 框架，包含以下复杂子系统：

```
django/
├── apps/          # 应用注册系统
├── conf/          # 配置管理
├── contrib/       # 内置应用（admin, auth, sessions 等）
├── core/          # 核心功能（缓存、邮件、验证等）
├── db/            # ORM 数据库层
├── dispatch/      # 信号调度系统
├── forms/         # 表单处理
├── http/          # HTTP 请求/响应
├── middleware/    # 中间件系统
├── template/      # 模板引擎
├── test/          # 测试框架
├── urls/          # URL 路由
├── utils/         # 工具函数
└── views/         # 视图层
```

## 为什么选择 Django?

1. **SWE-bench 核心项目**：Django 是 SWE-bench 中问题数量最多的仓库之一，被广泛用于评估 AI 代码理解能力。

2. **复杂调用关系**：
   - ORM 层与数据库的交互
   - 模板引擎与 Python 代码的混合
   - 中间件链式调用
   - 信号系统的发布/订阅模式

3. **多语言特性**：
   - Python 核心代码
   - JavaScript 管理后台交互
   - HTML/CSS 模板和样式

4. **丰富的测试用例**：223 个测试目录，覆盖各种边界情况。

## 测试 Lightkeeper

### 基本测试

```bash
# 启动索引器监视 Django 项目
./lightkeeper start --path ./testdata/django-repo

# 或者仅服务已有索引
./lightkeeper mcp --index-path ./testdata/django-repo/.kimi-index
```

### 测试场景建议

1. **大规模文件变更**：修改 `django/db/` 下的文件，观察索引更新
2. **跨模块引用**：测试 `django/contrib/admin/` 对其他模块的引用追踪
3. **模板解析**：测试 HTML 模板与 Python 代码的关联
4. **增量更新**：频繁修改单个文件，测试 debounce 机制

## 数据来源

- **仓库**：https://github.com/django/django
- **克隆深度**：100 commits（减少磁盘占用同时保留足够历史）
- **SWE-bench 参考**：https://github.com/princeton-nlp/SWE-bench
