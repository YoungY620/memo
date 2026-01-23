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

agent:
  api_key: "your-api-key"
  model: "your-model"

watch:
  ignore_patterns:
    - ".git"
    - "node_modules"
    - ".memo"
    - "*.log"
  debounce_ms: 5000    # 5s quiet period
  max_wait_ms: 300000  # 5min max wait
```

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
