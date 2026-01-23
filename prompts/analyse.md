# Analysis Task

Files in the codebase have changed. Your task is to:

1. Read the changed files to understand what was modified
2. Read the current `.baecon/*.json` files to see existing documentation
3. Analyze the changes and update all four JSON files accordingly:
   - **arch.json**: Update module definitions if modules were added, removed, or changed
   - **interface.json**: Update external/internal interfaces if APIs changed
   - **stories.json**: Update user stories or call chains if workflows changed
   - **issues.json**: Update issues, TODOs, design decisions found in the code

4. Use write_file to save your changes to each JSON file

## Guidelines

- Be thorough but concise in descriptions
- Use grep-able keywords for issue locations
- Include accurate line numbers for issues
- Remove entries for deleted code
- Update entries for modified code
- Add entries for new code

Start by reading the changed files and current .baecon files, then make your updates.
