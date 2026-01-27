# Memo

AI-powered codebase memory for coding agents. Watches file changes and maintains `.memo` documentation automatically.

## Installation

```bash
git clone https://github.com/YoungY620/memo.git
cd memo
make install  # installs to ~/.local/bin
```

## Why Memo?

Vibe coding with AI agents across multiple sessions evolves fast—too fast for humans to keep up. Projects spiral out of control due to lack of **global consistency** : models can't read the entire codebase in one pass, and neither can humans.

Inspired by traditional code indexing, Memo maintains a semantic index specifically for coding agents, capturing architecture and key decisions. This enables:

- **"Summarize this repo"** — No need to read every file. Memo provides instant context.
- **Preserve design decisions** — Trade-offs and constraints are recorded once, not repeated every session.
- **Holistic refactoring** — After modifying a module, agents can update related code across the project, even connections that static analysis can't detect.
- **Large document navigation** — Find related content without scanning everything.

## Usage

```bash
memo                        # watch current directory
memo --path /path/to/repo   # watch specific directory
memo --config config.yaml   # custom config
memo --once                 # analyze once and exit
memo --version
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

## MCP Integration (Kimi CLI)

Memo provides two MCP tools for Kimi CLI:

- `memo_list_keys` — List keys at a JSON path in `.memo/index`
- `memo_get_value` — Get value at a JSON path in `.memo/index`

### Option 1: Separate watcher (recommended)

Add to `~/.kimi/mcp.json`:

```json
{
  "mcpServers": {
    "memo": {
      "command": "memo",
      "args": ["--mcp"]
    }
  }
}
```

Run watcher in a separate terminal to see real-time output:

```bash
memo --path /path/to/project
```

### Option 2: All-in-one

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

Runs MCP server and watcher together. Convenient but hides watcher output.

### Verify

```bash
kimi
> Summarize this repo
```

Kimi will use memo tools to read `.memo/index` and provide a summary.

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
