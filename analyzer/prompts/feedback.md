# Validation Failed - Please Fix

The JSON files in `.memo/index` failed schema validation. You must fix the errors.

## Instructions

1. Read the validation errors below carefully
2. Read the affected JSON files in `.memo/index/`
3. Fix the schema violations:
   - Missing required fields
   - Wrong data types
   - Invalid JSON syntax
   - Incorrect structure

4. Use write_file to save the corrected JSON files

## Common Fixes

- Ensure all required fields are present
- Ensure arrays are not null (use empty array `[]` instead)
- Ensure strings are not null (use empty string `""` instead)
- Ensure `locations` in issues.json has proper objects with `file`, `keyword`, and `line`
- Ensure `line` in locations is an integer, not a string

Fix all errors and save the corrected files.
