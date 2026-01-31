# Feature: Auto Update via Git Repository

Automatically update memo binary from any git repository using `make update`.

## Status

- [x] Solution defined
- [x] Implemented

## Solution

Use `make update` to pull latest code and rebuild:

```bash
make update
```

| Step | Command | Description |
|------|---------|-------------|
| 1 | `git pull --rebase` | Fetch and rebase latest changes |
| 2 | `make install` | Build and install binary |

## Advantages

| Benefit | Description |
|---------|-------------|
| Universal | Works with any git service (GitHub, GitLab, Gitee, self-hosted) |
| Simple | No API integration needed |
| Reliable | Uses standard git workflow |
| Transparent | User sees exactly what's happening |

## Files

| File | Change |
|------|--------|
| `Makefile` | Add `update` target |

## Patch

```diff
// Makefile
+ update:
+ 	git pull --rebase
+ 	$(MAKE) install
```

## TODO

- [x] Add `update` target to Makefile
- [ ] Test
