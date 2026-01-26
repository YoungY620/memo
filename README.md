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

### Option 1: MCP server + manual watcher (recommended)

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

Then run the watcher manually in a separate terminal:

```bash
memo --path /path/to/project
```

This way you can monitor the watcher's real-time analysis output while Kimi CLI uses the MCP tools.

### Option 2: All-in-one (experimental)

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

The `--mcp-with-watcher` flag runs both the MCP server and file watcher together. This is convenient but hides watcher output, making it harder to diagnose issues.

### Testing

To verify the MCP integration is working:

1. Open a new Kimi CLI session in your project directory:
   ```bash
   kimi
   ```

2. Ask Kimi to summarize the repo:
   ```
   Summarize this repo for me
   ```

Kimi should use the `memo_list_keys` and `memo_get_value` tools to read the `.memo/index` files and provide a summary.

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
