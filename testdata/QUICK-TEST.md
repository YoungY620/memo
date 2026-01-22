# 快速测试指南

## 🎯 TL;DR - 三步测试流程

```bash
# 1. 修改代码
vim testdata/django-repo/django/conf/global_settings.py

# 2. 生成 patch
cd testdata/django-repo && git diff > /tmp/my_fix.patch

# 3. 运行评估
python3 -m swebench.harness.run_evaluation \
    --predictions_path /tmp/predictions.jsonl \
    --instance_ids django__django-10914 \
    --run_id test
```

---

## 📋 完整示例流程

### 示例问题：`django__django-10914`

**问题**：`FILE_UPLOAD_PERMISSIONS` 默认值应该是 `0o644` 而不是 `None`

**修复位置**：`django/conf/global_settings.py`

### Step 1: 模拟 Bug 状态

```bash
cd testdata/django-repo

# 引入 bug（将 0o644 改为 None）
sed -i '' 's/FILE_UPLOAD_PERMISSIONS = 0o644/FILE_UPLOAD_PERMISSIONS = None/' \
    django/conf/global_settings.py

# 确认 bug 状态
grep "FILE_UPLOAD_PERMISSIONS" django/conf/global_settings.py
# 输出: FILE_UPLOAD_PERMISSIONS = None
```

### Step 2: 应用你的修复

```bash
# 修复 bug（将 None 改回 0o644）
sed -i '' 's/FILE_UPLOAD_PERMISSIONS = None/FILE_UPLOAD_PERMISSIONS = 0o644/' \
    django/conf/global_settings.py

# 确认修复
grep "FILE_UPLOAD_PERMISSIONS" django/conf/global_settings.py
# 输出: FILE_UPLOAD_PERMISSIONS = 0o644
```

### Step 3: 生成 Patch 文件

```bash
# 查看修改
git diff

# 保存 patch
git diff > /tmp/my_fix.patch
cat /tmp/my_fix.patch
```

### Step 4: 创建评估输入文件

```bash
python3 << 'EOF'
import json

# 读取你的 patch
with open('/tmp/my_fix.patch', 'r') as f:
    patch = f.read()

# 创建预测文件
prediction = {
    "instance_id": "django__django-10914",
    "model_name_or_path": "manual-fix",
    "model_patch": patch
}

with open('/tmp/predictions.jsonl', 'w') as f:
    f.write(json.dumps(prediction) + '\n')

print("✅ Created /tmp/predictions.jsonl")
print(f"📝 Patch length: {len(patch)} chars")
EOF
```

### Step 5: 运行 SWE-bench 评估

```bash
# 安装 swebench（如果还没安装）
pip install swebench

# 运行评估
python3 -m swebench.harness.run_evaluation \
    --dataset_name princeton-nlp/SWE-bench_Lite \
    --predictions_path /tmp/predictions.jsonl \
    --max_workers 1 \
    --instance_ids django__django-10914 \
    --run_id manual-test

# 查看结果
cat evaluation_results/manual-test/results.json
```

### Step 6: 解读结果

```json
// 成功的结果
{
  "resolved": ["django__django-10914"],
  "unresolved": [],
  "error": [],
  "resolve_rate": 1.0
}

// 失败的结果
{
  "resolved": [],
  "unresolved": ["django__django-10914"],
  "error": [],
  "resolve_rate": 0.0
}
```

---

## 🔄 重置测试环境

```bash
cd testdata/django-repo

# 丢弃所有本地修改
git checkout .

# 或者重置到特定状态
git reset --hard HEAD
```

---

## 🧪 不使用 SWE-bench 的简单验证

如果不想安装 swebench，可以直接对比你的 patch 和 ground truth：

```bash
# 你的 patch
cat /tmp/my_fix.patch

# Ground Truth
python3 -c "
import json
with open('testdata/swebench-samples/example_instance.json') as f:
    print(json.load(f)['patch'])
"

# 自动对比
diff <(cat /tmp/my_fix.patch) \
     <(python3 -c "import json; print(json.load(open('testdata/swebench-samples/example_instance.json'))['patch'])")
```

---

## 📊 批量测试多个实例

```bash
# 处理多个实例
for instance in testdata/swebench-samples/django_instances.jsonl; do
    INSTANCE_ID=$(echo "$instance" | jq -r '.instance_id')
    echo "Testing: $INSTANCE_ID"
    # ... 你的测试逻辑
done
```

---

## ⚠️ 常见问题

| 问题 | 解决方案 |
|------|---------|
| `base_commit not found` | 需要更深的 git clone：`git fetch --unshallow` |
| `swebench not found` | 安装：`pip install swebench` |
| Docker 错误 | 确保 Docker 已启动并有足够空间 |
| 测试超时 | 增加 timeout 或减少 max_workers |
