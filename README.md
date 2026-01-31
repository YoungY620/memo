# Memo

AI-powered codebase memory for coding agents. Watches file changes and maintains `.memo` documentation automatically.

## Installation

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/YoungY620/memo/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/YoungY620/memo/main/install.ps1 | iex
```

### Update

Run the same installation command to update to the latest version. Memo also checks for updates automatically on startup.

### Build from Source

```bash
git clone https://github.com/YoungY620/memo.git
cd memo
make install  # installs to ~/.local/bin
```

## Why Memo?

Vibe coding with AI agents across multiple sessions evolves fast—too fast for humans to keep up. Projects spiral out of control due to lack of **global consistency**: models can't read the entire codebase in one pass, and neither can humans.

Inspired by traditional code indexing, Memo maintains a semantic index specifically for coding agents, capturing architecture and key decisions. This enables:

- **"Summarize this repo"** — No need to read every file. Memo provides instant context.
- **Preserve design decisions** — Trade-offs and constraints are recorded once, no need to repeat every session.
- **Holistic refactoring** — After modifying a module, agents can update related code across the project, even connections that static analysis can't detect.
- **Beyond coding: Large document navigation** — Find related content without scanning everything.

## Benchmark

Evaluated on a subset of [SWE-bench Lite](https://www.swebench.com/) (23 instances, limited by time):

[performance_analysis_final.png](performance_analysis_final.png)

**Key findings:**
- **4× pass rate improvement** (4.3% → 17.4%)
- **15% faster inference** (212s → 180s per instance)
- Memo index generation is one-time cost, amortized across tasks

## Usage

Memo has three commands:

### Watch Mode (default)
Continuously monitors file changes and updates `.memo/index`:
```bash
memo                          # watch current directory
memo watch                    # explicit watch command
memo watch -p /path/to/repo   # watch specific directory
memo watch --skip-scan        # skip initial full scan (when index is up-to-date)
```

### Scan Mode
Analyzes all files once, updates index, then exits. Useful for CI or initial setup:
```bash
memo scan
memo scan -p /path/to/repo
```

### MCP Mode
Starts an MCP server for AI agents to query the index. Requires an existing `.memo/index` (run watch/scan first):
```bash
memo mcp
memo mcp -p /path/to/repo
```

### Global Options
```bash
memo --version                # print version
memo --help                   # show help
memo <command> --help         # show command-specific help
```

### Command-Specific Options
```bash
# watch and scan commands
memo watch -c config.yaml     # custom config file
memo watch --log-level debug  # log level: error/notice/info/debug
```

## Configuration

Create `config.yaml`:

```yaml
log_level: info  # error, notice, info, debug

watch:
  ignore_patterns:
    - ".git"
    - "node_modules"
    - ".memo"
    - "*.log"
  debounce_ms: 5000    # 5s quiet period
  max_wait_ms: 300000  # 5min max wait
```

## MCP Integration

Memo exposes `.memo/index` to AI agents via MCP protocol:

- `memo_list_keys` — List keys at a JSON path
- `memo_get_value` — Get value at a JSON path

### Typical Workflow

1. **Start watcher** (keeps index updated as you code):
   ```bash
   cd /path/to/project && memo   # or: memo watch -p /path/to/project
   ```

2. **Configure AI agent** to use memo MCP server. Example for Kimi CLI (`~/.kimi/mcp.json`):
   ```json
   {
     "mcpServers": {
       "memo": {
         "command": "memo",
         "args": ["mcp"]
       }
     }
   }
   ```

3. **Query via agent**:
   ```bash
   kimi
   > Summarize this repo
   ```

## Output

```
.memo/
├── index/
│   ├── arch.json       # modules and structure
│   ├── interface.json  # external/internal APIs
│   ├── stories.json    # user stories and flows
│   └── issues.json     # TODOs, decisions, bugs
├── mcp.json            # local MCP config
└── .gitignore          # excludes runtime files
```

## License

MIT
