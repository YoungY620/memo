# Agent 输出行缓冲设计文档

## 问题描述

当前 SDK 的流式输出是逐 token 返回的，导致日志非常碎片化：

```
[DEBUG] Agent output: I'll start
[DEBUG] Agent output:  by reading
[DEBUG] Agent output:  the changed files and the current `.memo
[DEBUG] Agent output: /index/*.
```

期望效果：

```
[DEBUG] Agent output: I'll start by reading the changed files and the current `.memo/index/*.json` files to understand what needs to be updated.
```

## 设计目标

1. **至少一行完整再输出** - 缓冲文本直到遇到换行符 `\n`
2. **不永久阻塞** - 即使没有换行符，超时后也要输出
3. **Step 结束时刷新** - 当 step 结束时，输出剩余缓冲内容

## SDK 可用信息

根据 `wire/message.go` 分析，SDK 提供以下事件可以辅助判断：

| 事件类型 | 说明 | 用途 |
|---------|------|------|
| `StepBegin` | 新 step 开始 | 重置缓冲区 |
| `ContentPart` | 文本片段 | 累积到缓冲区 |
| `StatusUpdate` | 状态更新（含 token 统计） | 可作为"一轮输出结束"的提示 |
| `StepInterrupted` | step 被中断 | 刷新缓冲区 |

**注意**：SDK **没有** 明确的 "一句话结束" 或 "一段结束" 标记，需要自行实现行缓冲逻辑。

## 实现方案

### 核心数据结构

```go
type LineBuffer struct {
    buffer    strings.Builder
    lastFlush time.Time
    timeout   time.Duration  // 建议 500ms~1s
    mu        sync.Mutex
}
```

### 刷新触发条件

1. **遇到换行符** - 立即输出缓冲区内容
2. **超时** - 距离上次输出超过 `timeout` 且缓冲区非空
3. **Step 结束** - `step.Messages` channel 关闭时
4. **特殊标记** - 遇到 `StatusUpdate` 事件时（通常表示一轮生成结束）

### 代码改动

修改 `analyser.go` 的 `runPrompt` 函数：

```go
func (a *Analyser) runPrompt(ctx context.Context, session *agent.Session, prompt string) error {
    turn, err := session.Prompt(ctx, wire.NewStringContent(prompt))
    if err != nil {
        return fmt.Errorf("prompt failed: %w", err)
    }

    lb := NewLineBuffer(500 * time.Millisecond)  // 500ms 超时

    for step := range turn.Steps {
        for msg := range step.Messages {
            switch m := msg.(type) {
            case wire.ApprovalRequest:
                logDebug("Auto-approving request")
                m.Respond(wire.ApprovalRequestResponseApprove)
            case wire.ContentPart:
                if m.Type == wire.ContentPartTypeText && m.Text.Valid {
                    lb.Write(m.Text.Value)
                    // 检查是否需要刷新（换行或超时）
                    if lines := lb.Flush(false); lines != "" {
                        logDebug("Agent output: %s", lines)
                    }
                }
            case wire.StatusUpdate:
                // StatusUpdate 通常意味着一轮生成告一段落
                if lines := lb.Flush(true); lines != "" {
                    logDebug("Agent output: %s", lines)
                }
            }
        }
        // Step 结束，强制刷新剩余内容
        if lines := lb.Flush(true); lines != "" {
            logDebug("Agent output: %s", lines)
        }
    }

    if err := turn.Err(); err != nil {
        return fmt.Errorf("turn error: %w", err)
    }
    return nil
}
```

### LineBuffer 实现

```go
type LineBuffer struct {
    buffer    strings.Builder
    lastFlush time.Time
    timeout   time.Duration
}

func NewLineBuffer(timeout time.Duration) *LineBuffer {
    return &LineBuffer{
        timeout:   timeout,
        lastFlush: time.Now(),
    }
}

func (lb *LineBuffer) Write(s string) {
    lb.buffer.WriteString(s)
}

// Flush 返回可以输出的内容
// force=true: 强制输出所有缓冲内容
// force=false: 只输出完整的行，或超时后输出
func (lb *LineBuffer) Flush(force bool) string {
    content := lb.buffer.String()
    if content == "" {
        return ""
    }

    // 强制刷新
    if force {
        lb.buffer.Reset()
        lb.lastFlush = time.Now()
        return strings.TrimRight(content, "\n")
    }

    // 检查是否有完整行
    if idx := strings.LastIndex(content, "\n"); idx != -1 {
        lines := content[:idx]
        lb.buffer.Reset()
        lb.buffer.WriteString(content[idx+1:])
        lb.lastFlush = time.Now()
        return lines
    }

    // 检查超时
    if time.Since(lb.lastFlush) >= lb.timeout {
        lb.buffer.Reset()
        lb.lastFlush = time.Now()
        return content
    }

    return ""
}
```

## 配置项建议

在 `config.yaml` 中添加：

```yaml
output:
  line_buffer_timeout_ms: 500  # 行缓冲超时，默认 500ms
```

## 流程图

```
收到 ContentPart
       │
       ▼
  写入缓冲区
       │
       ▼
   有换行符？ ──是──▶ 输出完整行，保留剩余
       │
       否
       │
       ▼
    已超时？ ──是──▶ 输出缓冲区全部内容
       │
       否
       │
       ▼
     等待下一个 ContentPart
       │
       │
收到 StatusUpdate / Step 结束
       │
       ▼
  强制刷新缓冲区
```

## 边界情况处理

| 场景 | 处理方式 |
|------|---------|
| 模型长时间不输出换行 | 超时后强制输出 |
| 模型一次输出多行 | 全部输出（按 `\n` 分割） |
| Step 结束时缓冲区有内容 | 强制刷新 |
| 空内容 | 跳过，不输出 |

## 预期效果

**Before:**
```
[DEBUG] Agent output: I'll start
[DEBUG] Agent output:  by reading
[DEBUG] Agent output:  the changed files
```

**After:**
```
[DEBUG] Agent output: I'll start by reading the changed files and the current `.memo/index/*.json` files to understand what needs to be updated.
```

## 文件组织

所有改动集中在 **`log.go`** 和 **`analyser.go`** 两个文件：

| 文件 | 改动内容 |
|------|----------|
| **`log.go`** | 新增 `LineBuffer` 结构体及方法 |
| **`analyser.go`** | `runPrompt` 函数使用 LineBuffer |

## 实现优先级

1. ✅ 基础行缓冲（检测 `\n`）
2. ✅ 超时机制（防止永久阻塞）
3. ✅ Step 结束刷新
4. ⬜ 可配置的超时参数
