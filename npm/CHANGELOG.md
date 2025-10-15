# @lightfastai/dual

## 0.2.2

### Patch Changes

- 35ba33e: Test automated release workflow after consolidation refactoring. This patch validates that the new path-based trigger system correctly automates tag creation and multi-channel publishing without manual intervention.

## 0.2.1

### Patch Changes

- ecb9c90: Add npm package badges to README

  Display npm version and download count badges in the README to make npm installation more prominent and show package popularity.

## 0.2.0

### Minor Changes

- ec2d79a: Enable npm package distribution with automated multi-channel releases

  The dual CLI is now available via npm (`npm install -g @lightfastai/dual`) in addition to GitHub Releases and Homebrew. The npm wrapper automatically downloads the correct native binary for your platform during installation.

  This release also introduces fully automated release management using Changesets:

  - Automatic version bumping based on semantic versioning
  - Auto-generated CHANGELOGs from changeset descriptions
  - "Version Packages" PRs that preview all changes before release
  - One-click releases across GitHub, Homebrew, and npm
  - Post-release verification of all distribution channels

  Contributors can now document changes by running `npm run changeset` when making PRs. See CONTRIBUTING.md for the complete workflow.
