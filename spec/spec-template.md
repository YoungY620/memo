# Feature: [Feature Name]

[One-line description]

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
