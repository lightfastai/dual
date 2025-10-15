# @lightfastai/dual

## 0.1.0

### Features

- Implement dual doctor health check command (#59)
- Shell completions for bash/zsh/fish (#44, #57)
- Add npm wrapper package for seamless package.json integration (#48, #56)
- Wave 1 & 2 - Core features implementation (#41, #42, #37, #36, #43, #32, #40, #52)
- Implement environment layering foundation (#31, #33, #39, #51)

### Bug Fixes

- Use PAT for Homebrew tap updates (5fcac64)
- Remove invalid fieldalignment setting from golangci.yml (#46)

### Documentation

- Add bash wrapper pattern for package.json integration (#49, #55)
- Fix documentation inaccuracies across all files
- Update CLAUDE.md with implementation details

---

**Note:** This CHANGELOG is now managed by [Changesets](https://github.com/changesets/changesets).
Going forward, all changes will be automatically documented through changeset files.

To contribute:
1. Make your changes
2. Run `npm run changeset` to create a changeset
3. Describe your changes in the changeset
4. Commit the changeset file with your PR

When the PR is merged, a "Version Packages" PR will be automatically created with updated versions and CHANGELOG.
