# Lightkeeper

AI-powered codebase documentation keeper. Watches file changes and automatically maintains `.baecon` documentation using Kimi Agent.

## Installation

### Option 1: go install (requires Go 1.21+)

```bash
go install github.com/YoungY620/lightkeeper@latest
```

### Option 2: One-line install script

```bash
curl -sSL https://raw.githubusercontent.com/YoungY620/lightkeeper/main/install.sh | bash
```

### Option 3: Download binary

Download from [Releases](https://github.com/YoungY620/lightkeeper/releases) and add to PATH.

## Usage

```bash
# Watch current directory
lightkeeper

# Watch specific directory
lightkeeper --path /path/to/project

# Use custom config
lightkeeper --config /path/to/config.yaml

# Show version
lightkeeper --version
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
    - ".baecon"
    - "*.log"
  debounce_ms: 5000    # 5s quiet period
  max_wait_ms: 300000  # 5min max wait
```

## Output

Lightkeeper maintains `.baecon/` directory with:

- `arch.json` - Module definitions
- `interface.json` - External/internal interfaces
- `stories.json` - User stories and call chains
- `issues.json` - Design decisions, TODOs, bugs

## License

MIT
