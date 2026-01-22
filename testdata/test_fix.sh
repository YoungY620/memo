#!/bin/bash
set -e

INSTANCE_FILE="${1:-swebench-samples/example_instance.json}"
REPO_DIR="${2:-django-repo}"

# 提取信息
INSTANCE_ID=$(python3 -c "import json; print(json.load(open('$INSTANCE_FILE'))['instance_id'])")
BASE_COMMIT=$(python3 -c "import json; print(json.load(open('$INSTANCE_FILE'))['base_commit'])")

echo "========================================"
echo "Testing: $INSTANCE_ID"
echo "Base Commit: $BASE_COMMIT"
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
git diff

echo ""
echo "🧪 Generating prediction file..."

# 生成 patch（处理特殊字符）
git diff > /tmp/my_patch.diff
PATCH=$(cat /tmp/my_patch.diff)

# 创建预测文件
python3 << EOF
import json

with open('/tmp/my_patch.diff', 'r') as f:
    patch = f.read()

prediction = {
    "instance_id": "$INSTANCE_ID",
    "model_name_or_path": "manual-fix", 
    "model_patch": patch
}

with open('/tmp/predictions.jsonl', 'w') as f:
    f.write(json.dumps(prediction))

print("✅ Created /tmp/predictions.jsonl")
EOF

echo ""
echo "📋 To run full evaluation with swebench:"
echo "   python3 -m swebench.harness.run_evaluation \\"
echo "       --dataset_name princeton-nlp/SWE-bench_Lite \\"
echo "       --predictions_path /tmp/predictions.jsonl \\"
echo "       --max_workers 1 \\"
echo "       --instance_ids $INSTANCE_ID \\"
echo "       --run_id manual-test"
echo ""
echo "Or verify against ground truth:"
echo "   cat /tmp/predictions.jsonl"
