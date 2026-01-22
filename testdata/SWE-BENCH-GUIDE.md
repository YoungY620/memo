# SWE-bench 工具链完整指南

## 📋 概述

SWE-bench 评估流程包含三个核心阶段：

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   1. 推理阶段   │────▶│   2. 评估阶段   │────▶│   3. 评分阶段   │
│   (Inference)   │     │  (Evaluation)   │     │   (Scoring)     │
│                 │     │                 │     │                 │
│ 给模型问题描述  │     │ 在Docker中运行  │     │ 对比测试结果    │
│ 生成修复补丁    │     │ 项目的测试用例  │     │ 计算通过率      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

---

## 🔧 环境准备

### 1. 安装 SWE-bench

```bash
# 从 PyPI 安装
pip install swebench

# 或从源码安装
git clone https://github.com/princeton-nlp/SWE-bench.git
cd SWE-bench
pip install -e .
```

### 2. 安装 Docker

SWE-bench 使用 Docker 进行隔离的可复现评估。

```bash
# macOS
brew install --cask docker

# Linux
sudo apt-get install docker.io
```

### 3. 加载数据集

```python
from datasets import load_dataset

# 加载完整数据集 (2,294 个问题)
swebench = load_dataset('princeton-nlp/SWE-bench', split='test')

# 加载精简版 (300 个问题，推荐用于快速评估)
swebench_lite = load_dataset('princeton-nlp/SWE-bench_Lite', split='test')

# 加载验证版 (500 个人工确认可解的问题)
swebench_verified = load_dataset('princeton-nlp/SWE-bench_Verified', split='test')
```

---

## 📊 数据集结构

每个 instance 包含以下字段：

| 字段 | 描述 | 示例 |
|------|------|------|
| `instance_id` | 唯一标识符 | `django__django-11848` |
| `repo` | GitHub 仓库 | `django/django` |
| `base_commit` | 基础提交 hash | `abc123...` |
| `problem_statement` | Issue 描述 | "TypeError when..." |
| `hints_text` | 可选提示 | 评论、讨论内容 |
| `patch` | **Ground Truth 补丁** | `diff --git a/...` |
| `test_patch` | 验证用的测试补丁 | `diff --git a/tests/...` |
| `FAIL_TO_PASS` | 需要从失败变通过的测试 | `["test_foo"]` |
| `PASS_TO_PASS` | 需要保持通过的测试 | `["test_bar"]` |

---

## 🚀 阶段 1：推理 (Inference)

### 方式 A：使用 SWE-agent（推荐）

SWE-agent 是 Princeton 开发的自动化解题工具：

```bash
# 安装 SWE-agent
pip install sweagent

# 单个问题推理
sweagent run \
    --agent.model.name=model-name \
    --problem_statement.id=django__django-11848 \
    --problem_statement.source=swe_bench

# 批量推理
sweagent run-batch \
    --agent.model.name=model-name \
    --problem_statements.path=princeton-nlp/SWE-bench_Lite \
    --problem_statements.split=test
```

### 方式 B：自定义推理

你的模型需要输出 **JSONL 格式** 的预测文件：

```json
{"instance_id": "django__django-11848", "model_name_or_path": "my-model", "model_patch": "diff --git a/django/db/models/fields/__init__.py b/django/db/models/fields/__init__.py\n--- a/django/db/models/fields/__init__.py\n+++ b/django/db/models/fields/__init__.py\n@@ -100,6 +100,7 @@ def __init__(self, ...):\n+        # Fix: Added null check\n         if value is None:\n             return None\n"}
{"instance_id": "django__django-11849", "model_name_or_path": "my-model", "model_patch": "..."}
```

**关键字段：**
- `instance_id`: 必须与数据集中的 ID 完全匹配
- `model_patch`: 模型生成的 diff 补丁（unified diff 格式）

---

## 🔬 阶段 2：评估 (Evaluation)

### 评估流程

评估在 Docker 容器中执行，流程如下：

```
1. 克隆目标仓库到指定 commit
2. 应用 test_patch（添加验证测试）
3. 应用 model_patch（模型的修复）
4. 运行 FAIL_TO_PASS 测试（必须从失败变通过）
5. 运行 PASS_TO_PASS 测试（必须保持通过）
6. 记录测试结果
```

### 运行评估

```bash
# 评估预测结果
python -m swebench.harness.run_evaluation \
    --dataset_name princeton-nlp/SWE-bench_Lite \
    --predictions_path ./predictions.jsonl \
    --max_workers 8 \
    --run_id my-evaluation

# 验证 Gold Patches（Ground Truth）
python -m swebench.harness.run_evaluation \
    --predictions_path gold \
    --max_workers 1 \
    --instance_ids sympy__sympy-20590 \
    --run_id validate-gold

# 在 Modal 云端运行（更快）
python -m swebench.harness.run_evaluation \
    --dataset_name princeton-nlp/SWE-bench_Lite \
    --predictions_path ./predictions.jsonl \
    --modal true \
    --run_id cloud-eval
```

### 系统要求

| 资源 | 最低要求 |
|------|---------|
| 存储空间 | 120GB+ |
| 内存 | 16GB+ |
| CPU 核心 | 8+ |
| 架构 | x86_64（推荐），arm64（实验性） |

---

## 📈 阶段 3：评分 (Scoring)

### 评估输出结构

```
evaluation_results/
├── <run_id>/
│   ├── results.json          # 汇总结果
│   ├── <instance_id>.json    # 每个实例的详细结果
│   └── ...

logs/
├── build_images/             # Docker 镜像构建日志
└── run_evaluation/           # 评估执行日志
```

### 结果文件格式

`results.json` 示例：

```json
{
  "resolved": ["django__django-11848", "django__django-11849"],
  "unresolved": ["django__django-11850"],
  "error": [],
  "total": 300,
  "resolved_count": 2,
  "unresolved_count": 1,
  "error_count": 0,
  "resolve_rate": 0.67
}
```

### 核心评分指标

| 指标 | 计算方式 | 含义 |
|------|---------|------|
| **% Resolved** | `resolved / total * 100` | 成功解决的问题比例 |
| **FAIL_TO_PASS** | 原本失败的测试是否通过 | 验证修复是否生效 |
| **PASS_TO_PASS** | 原本通过的测试是否保持通过 | 验证没有引入回归 |

### 计算最终得分

```python
import json

# 加载结果
with open('evaluation_results/my-evaluation/results.json') as f:
    results = json.load(f)

# 计算通过率
resolve_rate = results['resolved_count'] / results['total'] * 100
print(f"Resolve Rate: {resolve_rate:.2f}%")
```

---

## 🛠️ 完整工作流示例

```bash
#!/bin/bash
# complete_swebench_eval.sh

# 1. 准备环境
pip install swebench sweagent datasets

# 2. 下载数据集
python -c "from datasets import load_dataset; load_dataset('princeton-nlp/SWE-bench_Lite', split='test')"

# 3. 使用 SWE-agent 进行推理
sweagent run-batch \
    --agent.model.name=model-name \
    --problem_statements.path=princeton-nlp/SWE-bench_Lite \
    --problem_statements.split=test \
    --output_dir=./predictions

# 4. 转换为评估格式（SWE-agent 自动生成）
# predictions/all_preds.jsonl

# 5. 运行评估
python -m swebench.harness.run_evaluation \
    --dataset_name princeton-nlp/SWE-bench_Lite \
    --predictions_path ./predictions/all_preds.jsonl \
    --max_workers 8 \
    --run_id gpt4o-eval

# 6. 查看结果
cat evaluation_results/gpt4o-eval/results.json
```

---

## 📚 SWE-bench 数据集变体

| 数据集 | 实例数 | 用途 |
|--------|-------|------|
| **SWE-bench (Full)** | 2,294 | 完整评估 |
| **SWE-bench Lite** | 300 | 快速迭代、成本敏感 |
| **SWE-bench Verified** | 500 | 人工验证可解，更可靠 |
| **SWE-bench Multimodal** | 517 | 包含图片的视觉问题 |

---

## 🔗 相关资源

- **SWE-bench 仓库**: https://github.com/princeton-nlp/SWE-bench
- **SWE-agent**: https://github.com/princeton-nlp/SWE-agent
- **官方排行榜**: https://www.swebench.com/
- **论文**: https://arxiv.org/abs/2310.06770
- **数据集 (HuggingFace)**: https://huggingface.co/datasets/princeton-nlp/SWE-bench

---

## 🎯 为 Lightkeeper 测试场景

使用 SWE-bench 数据集测试 Lightkeeper 的语义索引能力：

```bash
# 1. 加载一个 Django 实例
python -c "
from datasets import load_dataset
ds = load_dataset('princeton-nlp/SWE-bench_Lite', split='test')
django_instances = [x for x in ds if x['repo'] == 'django/django']
print(f'Django instances: {len(django_instances)}')
print(f'Example: {django_instances[0][\"instance_id\"]}')
print(f'Problem: {django_instances[0][\"problem_statement\"][:200]}...')
"

# 2. 使用 Lightkeeper 索引 Django 代码库
./indexer start --path ./testdata/django-repo

# 3. 测试语义搜索是否能找到相关文件
# （基于 problem_statement 的描述）
```

这样可以验证 Lightkeeper 的索引是否能帮助 AI Agent 更快定位问题相关的代码。
