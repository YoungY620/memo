# Session Context

## User Prompts

### Prompt 1

Implement the following plan:

# Plan: Robust MCP Server Startup

## Context

Currently, the MCP config requires `memo` to be pre-installed and in PATH:
```json
{ "mcpServers": { "memo": { "command": "memo", "args": ["mcp"] } } }
```
Problems:
1. If memo isn't installed, MCP server fails silently — agent loses all memo tools
2. `memo mcp` hard-exits if `.memo/index` doesn't exist (kills the MCP server process)
3. Corrupted index files produce unhelpful error messages

Goal: `eval "$(curl ...)"...

### Prompt 2

add commit push

