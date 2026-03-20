# Task: Analyze Checkpoint Transcripts

You are given one or more Entire.io checkpoint data sets containing AI session transcripts and metadata. Your task is to analyze each checkpoint and **fuse the insights into the existing 4 index files** (arch.json, interface.json, stories.json, issues.json).

A checkpoint transcript is just another data source — treat it the same as reading source code, but the information comes from a recorded AI coding session.

## Steps

1. **Read current index files** — load all four `.memo/index/*.json` files to understand existing entries.
2. **For each checkpoint**, analyze the provided data:
   - Read `metadata.json` for session metadata (agent, timestamp, attribution).
   - Read `full.jsonl` (transcript) to understand what happened in the session.
   - Read `context.md` and `prompt.txt` if available for additional context.
3. **Map insights to the 4 dimensions**:

### arch.json
- If the session reveals structural or module-level changes (new modules created, modules refactored, dependency changes), update `arch.json` accordingly.
- Update module descriptions and relationships if the session changed them.

### interface.json
- If new external or internal interfaces were created or modified during the session, add or update entries in `interface.json`.

### stories.json
- **Each checkpoint session IS a story.** Add a story entry with:
  - `title`: A descriptive title summarizing what the session accomplished
  - `tags`: Must include `"ai-session"` plus relevant topic tags (e.g., `"refactoring"`, `"bugfix"`, `"feature"`)
  - `content`: Narrative summary of the session — what the developer intended, what the AI did, what was accomplished, and any notable decisions or outcomes
- Use the checkpoint ID in the title or content to make it traceable.

### issues.json
- If the session uncovered bugs, raised concerns, made design decisions, or left TODOs, add entries to `issues.json` with appropriate tags.
- Design decisions from the session should use tags like `["design-decision", "ai-session"]`.
- TODOs or concerns should use tags like `["todo", "ai-session"]` or `["bug", "ai-session"]`.

4. **Validate**: Read all `.memo/index/*.json` files after writing to ensure valid JSON conforming to their schemas.

## Rules

- Read every piece of provided checkpoint data thoroughly.
- **Preserve existing entries** — do not remove entries that are unrelated to these checkpoints.
- **Avoid duplicates** — if a story or issue for a checkpoint already exists (match by checkpoint ID in title/content), update it instead of creating a duplicate.
- Be concise but thorough in summaries.
- Use specific file paths, not vague descriptions.
- Capture the "why" behind decisions, not just the "what".
- Reference module names from `arch.json` where applicable.

Start now: Read the current index files, then process each checkpoint.
