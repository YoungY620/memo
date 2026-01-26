# Memo

AI-powered codebase memory keeper. Watches file changes and maintains `.memo` documentation for coding agents.

## Installation

### Option 1: go install (requires Go 1.21+)

```bash
go install github.com/YoungY620/memo@latest
```

### Option 2: One-line install script

```bash
curl -sSL https://raw.githubusercontent.com/YoungY620/memo/main/install.sh | bash
```

### Option 3: Download binary

Download from [Releases](https://github.com/YoungY620/memo/releases) and add to PATH.

### Option 4: Build from source

```bash
git clone https://github.com/YoungY620/memo.git
cd memo
make install  # builds and installs to /usr/local/bin
memo --version
```

## Usage

```bash
# Watch current directory
memo

# Watch specific directory
memo --path /path/to/project

# Use custom config
memo --config /path/to/config.yaml

# Show version
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

## Kimi CLI MCP Integration

Memo can be used as an MCP server with Kimi CLI. This enables two MCP tools:

- `memo_list_keys`: List available keys at a path in `.memo/index` JSON files
- `memo_get_value`: Get JSON value at a path in `.memo/index` files

### Option 1: All-in-one (recommended)

Add to `~/.kimi/mcp.json`:

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

The `--mcp-with-watcher` flag runs both the MCP server and file watcher together, automatically keeping the index updated as you edit code.

### Option 2: MCP server only

If you want to see memo's real-time output (for debugging or monitoring), run the watcher manually in a separate terminal:

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

Then in another terminal:

```bash
memo --path /path/to/project
```

This way you can monitor the watcher's analysis output while Kimi CLI uses the MCP tools.

## Output

Memo maintains `.memo/` directory with:

```
.memo/
├── index/
│   ├── arch.json       # Module definitions
│   ├── interface.json  # External/internal interfaces
│   ├── stories.json    # User stories and call chains
│   └── issues.json     # Design decisions, TODOs, bugs
└── config.yml          # Local repo configuration (reserved)
```

## License

MIT
