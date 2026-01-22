# SWE-bench 手动解题与测试指南

## 📋 完整流程

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ 1. 获取问题 │───▶│ 2. 检出代码 │───▶│ 3. 手动修复 │───▶│ 4. 运行测试 │
│             │    │   到指定版本 │    │             │    │   验证结果  │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

---

## 步骤 1：获取问题详情

```bash
cd testdata

# 查看示例问题
python3 << 'EOF'
import json

with open('swebench-samples/example_instance.json') as f:
    instance = json.load(f)

print("=" * 60)
print(f"Instance ID: {instance['instance_id']}")
print(f"Repo: {instance['repo']}")
print(f"Base Commit: {instance['base_commit']}")
print("=" * 60)
print("\n📝 Problem Statement:\n")
print(instance['problem_statement'])
print("\n" + "=" * 60)
print("\n🧪 Tests that must FAIL → PASS:\n")
print(instance['FAIL_TO_PASS'])
print("\n🧪 Tests that must PASS → PASS:\n")
print(instance['PASS_TO_PASS'][:200] + "..." if len(instance['PASS_TO_PASS']) > 200 else instance['PASS_TO_PASS'])
EOF
```

---

## 步骤 2：检出到指定版本

```bash
cd testdata/django-repo

# 获取 base_commit（从示例数据）
BASE_COMMIT=$(python3 -c "import json; print(json.load(open('../swebench-samples/example_instance.json'))['base_commit'])")
echo "Checking out to: $BASE_COMMIT"

# 检出到问题发生时的代码版本
git fetch --depth=100 origin
git checkout $BASE_COMMIT

# 确认版本
git log --oneline -1
```

---

## 步骤 3：查看 Ground Truth（参考答案）

```bash
# 查看官方修复补丁
python3 << 'EOF'
import json

with open('../swebench-samples/example_instance.json') as f:
    instance = json.load(f)

print("=" * 60)
print("📋 Ground Truth Patch:")
print("=" * 60)
print(instance['patch'])
EOF
```

---

## 步骤 4：手动修复代码

### 方式 A：直接应用 Ground Truth

```bash
# 将 patch 保存到文件
python3 -c "import json; print(json.load(open('../swebench-samples/example_instance.json'))['patch'])" > /tmp/fix.patch

# 应用补丁
git apply /tmp/fix.patch

# 查看修改
git diff
```

### 方式 B：手动编辑文件

根据问题描述，自己编写修复代码：

```bash
# 例如：编辑 django/conf/global_settings.py
vim django/conf/global_settings.py

# 查看你的修改
git diff
```

### 方式 C：生成自己的 Patch

```bash
# 修改代码后，生成 patch 文件
git diff > /tmp/my_fix.patch

# 查看 patch
cat /tmp/my_fix.patch
```

---

## 步骤 5：运行测试验证

### 方式 A：使用 SWE-bench 评估器（推荐）

```bash
# 1. 创建预测文件
INSTANCE_ID=$(python3 -c "import json; print(json.load(open('../swebench-samples/example_instance.json'))['instance_id'])")
MY_PATCH=$(git diff)

# 创建 predictions.jsonl
python3 << EOF
import json

prediction = {
    "instance_id": "$INSTANCE_ID",
    "model_name_or_path": "manual-fix",
    "model_patch": '''$MY_PATCH'''
}

with open('/tmp/predictions.jsonl', 'w') as f:
    f.write(json.dumps(prediction))
    
print("Created /tmp/predictions.jsonl")
EOF

# 2. 运行评估
python -m swebench.harness.run_evaluation \
    --dataset_name princeton-nlp/SWE-bench_Lite \
    --predictions_path /tmp/predictions.jsonl \
    --max_workers 1 \
    --instance_ids $INSTANCE_ID \
    --run_id manual-test

# 3. 查看结果
cat evaluation_results/manual-test/results.json
```

### 方式 B：直接运行项目测试

```bash
# 1. 安装 Django 开发依赖
pip install -e .
pip install -r tests/requirements/py3.txt

# 2. 获取需要测试的测试用例
TEST_CASE=$(python3 -c "import json; print(json.load(open('../swebench-samples/example_instance.json'))['FAIL_TO_PASS'])")
echo "Running test: $TEST_CASE"

# 3. 运行特定测试
# Django 使用自己的测试运行器
cd tests
python runtests.py test_utils.tests.OverrideSettingsTests.test_override_file_upload_permissions

# 或者运行整个测试模块
python runtests.py test_utils
```

### 方式 C：使用 Docker 隔离测试

```bash
# 创建 Dockerfile
cat > /tmp/Dockerfile << 'EOF'
FROM python:3.9
WORKDIR /app
COPY . .
RUN pip install -e .
RUN pip install -r tests/requirements/py3.txt || true
EOF

# 构建并运行测试
docker build -t django-test -f /tmp/Dockerfile .
docker run --rm django-test python tests/runtests.py test_utils.tests.OverrideSettingsTests
```

---

## 步骤 6：对比结果

### 检查 FAIL_TO_PASS

```bash
# 修复前：测试应该失败
git stash  # 暂存修改
python tests/runtests.py test_utils.tests.OverrideSettingsTests.test_override_file_upload_permissions
# 预期：FAILED

# 修复后：测试应该通过
git stash pop  # 恢复修改
python tests/runtests.py test_utils.tests.OverrideSettingsTests.test_override_file_upload_permissions
# 预期：PASSED
```

### 检查 PASS_TO_PASS（回归测试）

```bash
# 确保其他测试仍然通过
python tests/runtests.py test_utils
# 预期：所有原本通过的测试仍然通过
```

---

## 📊 结果解读

### 成功标准

| 测试类型 | 修复前 | 修复后 | 结果 |
|---------|-------|-------|------|
| FAIL_TO_PASS | ❌ FAIL | ✅ PASS | 修复生效 |
| PASS_TO_PASS | ✅ PASS | ✅ PASS | 无回归 |

### 评估输出示例

```json
{
  "resolved": ["django__django-10914"],
  "unresolved": [],
  "error": [],
  "total": 1,
  "resolved_count": 1,
  "resolve_rate": 1.0
}
```

---

## 🔄 快速测试脚本

创建一键测试脚本：

```bash
cat > testdata/test_fix.sh << 'SCRIPT'
#!/bin/bash
set -e

INSTANCE_FILE="${1:-swebench-samples/example_instance.json}"
REPO_DIR="${2:-django-repo}"

# 提取信息
INSTANCE_ID=$(python3 -c "import json; print(json.load(open('$INSTANCE_FILE'))['instance_id'])")
BASE_COMMIT=$(python3 -c "import json; print(json.load(open('$INSTANCE_FILE'))['base_commit'])")
FAIL_TO_PASS=$(python3 -c "import json; print(json.load(open('$INSTANCE_FILE'))['FAIL_TO_PASS'])")

echo "========================================"
echo "Testing: $INSTANCE_ID"
echo "Base Commit: $BASE_COMMIT"
echo "Test: $FAIL_TO_PASS"
echo "========================================"

cd "$REPO_DIR"

# 检查是否有修改
if git diff --quiet; then
    echo "❌ No changes detected. Please fix the code first."
    exit 1
fi

echo ""
echo "📝 Your changes:"
git diff --stat

echo ""
echo "🧪 Running tests..."

# 生成 patch
PATCH=$(git diff)

# 创建预测文件
python3 << EOF
import json
prediction = {
    "instance_id": "$INSTANCE_ID",
    "model_name_or_path": "manual-fix",
    "model_patch": '''$PATCH'''
}
with open('/tmp/predictions.jsonl', 'w') as f:
    f.write(json.dumps(prediction))
EOF

# 运行评估
cd ..
python3 -m swebench.harness.run_evaluation \
    --dataset_name princeton-nlp/SWE-bench_Lite \
    --predictions_path /tmp/predictions.jsonl \
    --max_workers 1 \
    --instance_ids "$INSTANCE_ID" \
    --run_id manual-test 2>&1

# 显示结果
echo ""
echo "========================================"
echo "📊 Results:"
cat evaluation_results/manual-test/results.json 2>/dev/null || echo "Check logs for details"
SCRIPT

chmod +x testdata/test_fix.sh
```

使用方法：

```bash
cd testdata

# 1. 检出代码并修复
cd django-repo
git checkout <base_commit>
# ... 进行修改 ...

# 2. 运行测试
cd ..
./test_fix.sh
```

---

## 🎯 示例：完整流程演示

```bash
# 1. 进入测试目录
cd testdata

# 2. 查看问题
cat swebench-samples/example_instance.json | python3 -c "
import json, sys
d = json.load(sys.stdin)
print(f'Problem: {d[\"instance_id\"]}')
print(f'Description: {d[\"problem_statement\"][:300]}...')
"

# 3. 检出代码
cd django-repo
git checkout $(python3 -c "import json; print(json.load(open('../swebench-samples/example_instance.json'))['base_commit'])")

# 4. 应用 Ground Truth 补丁
python3 -c "import json; print(json.load(open('../swebench-samples/example_instance.json'))['patch'])" | git apply

# 5. 验证修改
git diff HEAD

# 6. 运行测试
cd ..
./test_fix.sh swebench-samples/example_instance.json django-repo
```
