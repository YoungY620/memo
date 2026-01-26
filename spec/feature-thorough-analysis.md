# Feature: Thorough Analysis Mode

Ensure all changed files are read and index files are updated module by module.

## Problem

Current analyze prompt may skip files or update index files in one batch, leading to:
- Missed file changes
- Incomplete analysis
- Context overflow on large changesets

## Architecture

```
Current:   Read files → Analyze all → Update all index files at once
Proposed:  Group files by module → Read module files → Update index → Next module
```

**Workflow**:
1. Create fine-grained todo list: one item per changed file
2. Group files by module/directory
3. For each module:
   - Read files one by one, mark each as done in todo
   - After all files in module are read, update relevant index files
4. Final validation pass

**Example todo**:
```
- [x] Read src/auth/login.go
- [x] Read src/auth/logout.go
- [x] Read src/auth/token.go
- [x] Update index for auth module
- [ ] Read src/api/handler.go
- [ ] Update index for api module
```

## Files

| File | Change |
|------|--------|
| `prompts/analyse.md` | Add module-by-module workflow with todo enforcement |

## TODO

- [x] `prompts/analyse.md`: Rewrite with todo-driven, module-based workflow
- [x] Test with large changeset (22 files, passed)
