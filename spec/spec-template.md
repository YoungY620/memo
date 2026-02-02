# Feature: [Feature Name]

[One-line description]

<!--
IMPORTANT: When writing examples in this spec:
- Use generic paths like `/path/to/project`, `~/.local/bin/`, `./example`
- Use generic usernames like `user` instead of real usernames
- NEVER include real file paths, usernames, API keys, or other sensitive information
-->

## Modules

| Module | Responsibility |
|--------|----------------|
| `module_a` | Description of module A |
| `module_b` | Description of module B |

## Architecture

```
┌─────────────┐      ┌─────────────┐
│  Module A   │─────▶│  Module B   │
└─────────────┘      └─────────────┘
       │                    │
       ▼                    ▼
┌─────────────┐      ┌─────────────┐
│  Module C   │◀────▶│  Module D   │
└─────────────┘      └─────────────┘
```

## Files

| File | Change |
|------|--------|
| `path/to/file.go` | Brief description |
| `path/to/new_file.go` | New file, description |

## Patch

```diff
// file.go
+ import "new/package"

- func oldFunc() {
+ func newFunc(param Type) error {
+     // new implementation
  }
```

## TODO

- [ ] Task 1
- [ ] Task 2
- [ ] Test
