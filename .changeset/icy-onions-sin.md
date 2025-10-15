---
"@lightfastai/dual": minor
---

Enable npm package distribution with automated multi-channel releases

The dual CLI is now available via npm (`npm install -g @lightfastai/dual`) in addition to GitHub Releases and Homebrew. The npm wrapper automatically downloads the correct native binary for your platform during installation.

This release also introduces fully automated release management using Changesets:
- Automatic version bumping based on semantic versioning
- Auto-generated CHANGELOGs from changeset descriptions
- "Version Packages" PRs that preview all changes before release
- One-click releases across GitHub, Homebrew, and npm
- Post-release verification of all distribution channels

Contributors can now document changes by running `npm run changeset` when making PRs. See CONTRIBUTING.md for the complete workflow.
