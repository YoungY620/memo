# Analysis Task

Files in the codebase have changed. Your task is to read all changed files and update `.memo/index/*.json` accordingly.

## Step 1: Create Todo List

First, create a fine-grained todo list with **one item per changed file**, grouped by module/directory:

```
Example:
- [ ] Read src/auth/login.go
- [ ] Read src/auth/logout.go
- [ ] Read src/auth/token.go
- [ ] Update index for auth module
- [ ] Read src/api/handler.go
- [ ] Read src/api/router.go
- [ ] Update index for api module
- [ ] Final validation
```

## Step 2: Process Module by Module

For each module group:

1. **Read each file** in the module one by one, mark as done after reading
2. **After all files in module are read**, update the relevant index files:
   - `arch.json`: Module definitions
   - `interface.json`: External/internal interfaces
   - `stories.json`: User stories, call chains
   - `issues.json`: TODOs, design decisions, bugs
3. Mark module update as done

## Step 3: Final Validation

After all modules processed:
1. Read all `.memo/index/*.json` files
2. Verify consistency and completeness
3. Fix any issues found

## Rules

- **Read every single changed file** - do not skip any
- **Mark todo items as you progress** - this ensures nothing is missed
- Be thorough but concise in descriptions
- Use grep-able keywords for issue locations
- Include accurate line numbers for issues
- Remove entries for deleted code
- Update entries for modified code
- Add entries for new code

Start now: Create your todo list from the changed files below, then process module by module.
