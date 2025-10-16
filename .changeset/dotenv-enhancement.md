---
"@lightfastai/dual": minor
---

feat: major improvements to environment handling, error UX, and dotenv compatibility

This release includes significant enhancements from multiple PRs:

## PR #77: Full Dotenv Compatibility
Replaces the custom .env file parser with the industry-standard godotenv library, adding full compatibility with Node.js dotenv features:

- **Multiline values**: Support for certificates, keys, and formatted text using quotes
- **Variable expansion**: `${VAR}` and `$VAR` syntax for DRY configuration
- **Escape sequences**: Process `\n`, `\\`, `\"` within double-quoted strings
- **Inline comments**: Support `KEY=value # comment` syntax
- **Complex quoting**: Handle nested and mixed quotes properly

## PR #82: Unified Environment Loading
Fixes critical bugs and unifies environment loading implementation:

- **Fixed**: Environment variables now properly load from service .env files
- **Fixed**: Base environment configuration now correctly recognized
- **Unified**: All environment loading now goes through consistent `LoadLayeredEnv()` function
- **Improved**: Consistent behavior across all commands (`env show`, `run`, etc.)

## PR #73: Enhanced Error Handling & UX
Improves error messages with actionable user guidance:

- **Better error messages**: Clear, actionable hints for common issues
- **Improved diagnostics**: More detailed information when things go wrong
- **User-friendly output**: Helpful suggestions for fixing configuration problems
- **Better validation**: Early detection of configuration issues

Breaking changes:
- Variable expansion is now enabled by default (previously literal values)
- Escape sequences like `\n` are now processed in double quotes (previously literal)
- Inline comments after values are now stripped (previously included in value)

Migration guide:
- Use single quotes for literal `${VAR}` values: `'${BASE_URL}/api'`
- Use single quotes or escape backslash for literal `\n`: `'Hello\nWorld'` or `"Hello\\nWorld"`
- Remove inline comments or quote values containing `#`: `"value#notacomment"`