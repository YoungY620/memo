# Lightkeeper

> *In the age of vibe coding, when agents navigate vast codebases through fog and storm, the Lightkeeper stands watch — faithful, tireless, ensuring no ship loses its way.*

**Lightkeeper** is a semantic indexing service for the vibe coding era. It watches your files, maintains a faithful map of your codebase, and keeps AI agents oriented — no matter how large or fast-moving your project becomes.

## Why Lightkeeper?

Traditional IDEs have indexers for syntax analysis — they parse your code, build symbol tables, and enable features like "go to definition" and autocomplete.

**Lightkeeper is the indexer for vibe coding.**

When you're building with AI agents, the challenge isn't syntax — it's *coherence*. As projects grow, agents lose sight of the whole. They make changes that conflict with distant parts of the codebase. They forget architectural decisions made yesterday. They drift.

**Loss of global coherence is the #1 reason vibe coding projects spiral out of control.**

Lightkeeper solves this by maintaining an **absolutely reliable source of truth** — a semantic index that evolves with your code, always consistent, always available. It's not just about saving context tokens (though it does that too). It's about giving agents a faithful map they can trust.

## How It Works

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Your Files    │────▶│   Lightkeeper   │────▶│  Semantic Index │
│  (code, docs)   │     │   (watcher +    │     │   (.kimi-index) │
│                 │     │    AI engine)   │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │   AI Agents     │
                                               │ (via MCP tools) │
                                               └─────────────────┘
```

1. **Watch**: Lightkeeper monitors your files for changes
2. **Accumulate**: Changes are buffered and debounced intelligently  
3. **Analyze**: When ready, AI analyzes the changes and updates the index
4. **Serve**: Agents access the index through MCP tools — progressive disclosure, search, navigation

The index is designed for **progressive disclosure**: agents can start with a high-level overview and drill down only where needed, preserving precious context for actual work.

## Modules

| Module | Responsibility |
|--------|----------------|
| **Core** | File watching → Change buffering → AI analysis → Index updates |
| **MCP** | Read-only interface for agents to explore the index |

## Quick Start

```bash
# Build
go build -o lightkeeper ./core/cmd/indexer

# Start watching (monitors current directory)
./lightkeeper start

# Or just serve the MCP interface for existing index
./lightkeeper mcp --index-path .kimi-index
```

## MCP Configuration

### IDE / Cursor

```json
{
  "mcpServers": {
    "lightkeeper": {
      "command": "lightkeeper",
      "args": ["mcp", "--index-path", "/path/to/project/.kimi-index"]
    }
  }
}
```

### Kimi CLI

```yaml
mcp:
  servers:
    lightkeeper:
      command: lightkeeper
      args: ["mcp", "--index-path", "/path/to/project/.kimi-index"]
```

## Documentation

- [DESIGN.md](./DESIGN.md) — Architecture and implementation details
- [INDEX.md](./INDEX.md) — Index storage structure
- [MCP.md](./MCP.md) — MCP module design
- [VALIDATOR.md](./VALIDATOR.md) — Validation rules

## The Name

A **lightkeeper** is the keeper of a lighthouse — the person who tends the flame, ensures the light never goes out, and guides ships safely through darkness and storm.

In vibe coding, your codebase is the sea. AI agents are the ships. And Lightkeeper stands watch, maintaining the one true map, so that no matter how wild the waters get, every agent can find its way.

## License

MIT
