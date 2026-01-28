# Fix: Empty Message Content in Session History

Session history may contain messages with empty content, causing API errors.

## Status

- [ ] Solution TBD

## Problem

| Issue | Impact |
|-------|--------|
| Empty content in session history | API rejects request with 400 error |
| Error: "all messages must have non-empty content" | Analysis fails completely |

**Error log:**
```
2026/01/28 18:48:58 [ERROR] Analysis failed: turn error: {"code":-32003,"message":"Error code: 400 - {'error': {'message': 'request id: xxx messages.125: all messages must have non-empty content except for the optional final assistant message', 'type': 'invalid_request_error'}}","data":null}
```

## Root Cause Analysis

TBD - Need to investigate:

1. **When does empty content occur?**
   - Interrupted analysis leaving incomplete message?
   - Tool call message without text content?
   - Session corruption?

2. **Where is the issue?**
   - Kimi Agent SDK session persistence?
   - memo's message handling?
   - API provider validation?

## Potential Solutions

| Solution | Pros | Cons |
|----------|------|------|
| Filter empty messages before sending | Quick fix | Hides underlying issue |
| Clear session on error and retry | Self-healing | Loses context |
| Fix message generation at source | Proper fix | Need root cause analysis |
| Don't reuse sessions | Avoids issue entirely | Loses session benefits |

## Workaround

Manually delete the corrupted session directory:
```bash
rm -rf ~/.kimi/sessions/<workdir-hash>/memo-<session-id>
```

## TODO

- [ ] Reproduce and identify root cause
- [ ] Determine solution approach
- [ ] Implement fix
- [ ] Test
