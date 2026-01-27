# Fix: dyld Zombie Process on Binary Replacement

Memo processes hang in dyld (Uninterruptible Sleep) after binary replacement, causing cascading failures.

## Problem

After recompiling and installing memo to `~/.local/bin/memo`, new memo processes hang indefinitely and become unkillable zombie processes.

### Symptoms

1. `memo --version` hangs forever
2. `memo --mcp` hangs, causing kimi startup timeout when memo is configured as MCP server
3. Processes enter UE (Uninterruptible Sleep) state, cannot be killed by `kill -9`
4. System accumulates 40+ zombie memo processes over time

```bash
$ ps aux | grep memo
user  78319  0.0  0.0 407971904  16  ??  UE   3:26PM  0:00.00 memo --version
user  79498  0.0  0.0 407971904  16  ??  UEs  3:29PM  0:00.00 memo --mcp
user  79497  0.0  0.0 407971904  16  ??  UEs  3:29PM  0:00.00 memo --mcp
# ... 40+ more UE/UEs processes
```

### Reproduction Steps

```bash
# 1. Have memo running or have zombie processes holding file references
$ lsof ~/.local/bin/memo
# Shows 13+ processes holding references

# 2. Rebuild and install (cp overwrites existing file)
$ make install

# 3. Try to run new version - hangs forever
$ ~/.local/bin/memo --version
# Hangs in dyld, becomes UE state zombie
```

### Impact on Kimi MCP

When memo is configured as MCP server in `~/.kimi/mcp.json`:

```json
{
  "mcpServers": {
    "memo": {
      "command": "memo",
      "args": ["--mcp-with-watcher"]
    }
  }
}
```

Kimi CLI startup will timeout waiting for memo MCP server to initialize.

## Root Cause

### Direct Cause

Process hangs in **dyld (dynamic linker)** before `main()` executes:

```
Call graph:
    Thread_1465544: Main Thread
      ???  (in <unknown binary>)  [0x10385ca40]  <-- Inside dyld __TEXT region
```

From vmmap:
```
__TEXT  103858000-1038e8000  [576K]  /usr/lib/dyld
        ↑ Process stuck here (0x10385ca40)
```

### Underlying Cause

When `cp` overwrites an existing binary file that is referenced by zombie processes:

```
┌─────────────────────────────────────────────────────────────────┐
│  1. Zombie processes hold references to old binary             │
│     └── lsof shows: memo (UE) -> /Users/.../memo (inode X)     │
│                                                                 │
│  2. `cp memo ~/.local/bin/memo` overwrites file                │
│     └── File content changed, inode may change                 │
│                                                                 │
│  3. New process tries to execute updated binary                │
│     └── dyld attempts to load/verify the binary                │
│                                                                 │
│  4. Conflict between old references and new content            │
│     └── Process hangs in dyld, enters UE state                 │
│                                                                 │
│  5. UE state process cannot be killed                          │
│     └── Only system reboot can clear these processes           │
└─────────────────────────────────────────────────────────────────┘
```

### Key Evidence

```bash
# Zombie processes hold file references
$ lsof ~/.local/bin/memo
COMMAND   PID   FD   TYPE  NODE NAME
memo    78319  txt   REG   8087650 ~/.local/bin/memo
memo    78829  txt   REG   8087650 ~/.local/bin/memo
# ... 13+ processes

# Different inodes for local vs global binary
$ ls -lai ./memo ~/.local/bin/memo
8139179 ./memo                        # Local build
8087650 ~/.local/bin/memo  # Global install

# Copying to fresh location works fine
$ cp ./memo /tmp/memo_test
$ /tmp/memo_test --version
memo 09d1daa  # Works!

# Deleting before copy also works
$ rm -f ~/.local/bin/memo
$ cp ./memo ~/.local/bin/memo
$ ~/.local/bin/memo --version
memo 09d1daa  # Works!
```

## Solution

### Approach

The root cause is a macOS-specific issue with dyld and file references. **There is no way to fix the UE zombie processes from userspace** - only a system reboot can clear them.

However, we can **prevent the problem from occurring** by ensuring the old binary is deleted before installing the new one:

```
┌─────────────────────────────────────────────────────────────────┐
│  Before (broken):                                               │
│    cp ./memo ~/.local/bin/memo                                  │
│    └── Overwrites file in-place, conflicts with zombie refs     │
│                                                                 │
│  After (fixed):                                                 │
│    rm -f ~/.local/bin/memo     ← Delete old file first          │
│    cp ./memo ~/.local/bin/memo ← Create new file (new inode)    │
│    └── No conflict, new inode has no zombie references          │
└─────────────────────────────────────────────────────────────────┘
```

### Why This Works

1. `rm -f` removes the file entry from the directory
2. Zombie processes still hold references to the **old inode** (file content remains on disk until all refs are closed)
3. `cp` creates a **new file with new inode**
4. New processes execute the new inode, no conflict with zombie refs

## Files

| File | Change |
|------|--------|
| `Makefile` | Add `rm -f` before `cp` in install target |

## Patch

### Makefile

```diff
  install: build
      mkdir -p $(HOME)/.local/bin
+     @# Remove old binary first to avoid dyld issues when processes hold references
+     rm -f $(HOME)/.local/bin/$(BINARY)
      cp $(BINARY) $(HOME)/.local/bin/$(BINARY)
      @echo "Installed $(BINARY) to $(HOME)/.local/bin"
```

## Troubleshooting

### Detecting Zombie Processes

```bash
# Check for UE/UEs state memo processes
ps aux | grep memo | grep -E 'UE|UEs'

# Check if processes are holding binary references
lsof /path/to/memo
```

### Clearing Zombie Processes

**UE state processes cannot be killed by any signal.** The only solution is:

1. **Reboot the system** to clear all zombie processes
2. After reboot, reinstall memo using the fixed `make install`

### Prevention

Always use `make install` to install memo. Never manually `cp` over an existing binary.

## TODO

- [x] Determine solution approach
- [x] Implement fix in Makefile
- [x] Test: Verify binary replacement works with zombie processes present
- [x] Test: Verify kimi MCP startup works after memo reinstall
- [x] Document: Add troubleshooting guide for zombie process cleanup
