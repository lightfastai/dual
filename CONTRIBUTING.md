# Contributing to Dual

Thank you for your interest in contributing to dual! This document provides guidelines for contributing to the project.

## Development Setup

### Prerequisites

- Go 1.23 or later
- Node.js 20 or later (for changesets and npm wrapper)
- Git

### Getting Started

```bash
# Clone the repository
git clone https://github.com/lightfastai/dual.git
cd dual

# Install dependencies
npm install

# Build the binary
go build -o dual ./cmd/dual

# Run tests
go test ./...

# Run integration tests
go test ./test/integration/...

# Run linter
golangci-lint run
```

## Making Changes

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 2. Make Your Changes

- Follow Go best practices and idioms
- Add tests for new functionality
- Update documentation as needed
- Ensure all tests pass: `go test ./...`
- Ensure linter passes: `golangci-lint run`

### 3. Create a Changeset

**Important:** All PRs must include a changeset (unless it's a docs-only change).

```bash
npm run changeset
```

You'll be prompted:

```
? What kind of change is this for @lightfastai/dual?
  ❯ patch  (bug fixes, no breaking changes)
    minor  (new features, no breaking changes)
    major  (breaking changes)

? Please enter a summary for this change:
  Add automatic context switching between git branches
```

This creates a file in `.changeset/` with a random name. Commit this file with your changes.

**Changeset Guidelines:**

- **Patch**: Bug fixes, documentation, refactoring with no user-facing changes
- **Minor**: New features, enhancements, non-breaking API additions
- **Major**: Breaking changes, major rewrites, API removals

**Example changeset file** (`.changeset/funny-clouds-dance.md`):

```markdown
---
"@lightfastai/dual": minor
---

Add automatic context switching between git branches

Dual now automatically detects when you switch branches and updates
the active context accordingly. This eliminates the need to manually
run `dual context switch` after checking out a branch.
```

### 4. Commit Your Changes

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```bash
git add .
git commit -m "feat: add automatic context switching"
```

**Commit format:**

- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `refactor:` Code refactoring
- `test:` Test additions or changes
- `chore:` Build process or auxiliary tool changes

### 5. Push and Create PR

```bash
git push origin feature/your-feature-name
gh pr create --fill
```

## What Happens After Your PR is Merged?

1. Your changeset is collected by the Changesets bot
2. A "Version Packages" PR is automatically created/updated
3. The Version Packages PR will include:
   - Updated version in `npm/package.json`
   - Your changes added to `CHANGELOG.md`
   - Your changeset file removed
4. A maintainer reviews and merges the Version Packages PR
5. A git tag is automatically created
6. The release is published to GitHub, Homebrew, and npm

You don't need to do anything else!

> **Note for maintainers:** See [RELEASE.md](RELEASE.md) for the complete release workflow, including required manual steps.

## Changesets FAQ

### When should I skip creating a changeset?

Only skip a changeset for:
- Documentation-only changes (README, comments)
- CI/CD configuration changes
- Test-only changes that don't affect the binary

Add `[skip-changeset]` to your PR title if applicable.

### Can I create multiple changesets?

Yes! If your PR includes multiple independent changes, create separate changesets:

```bash
npm run changeset  # First change
npm run changeset  # Second change
```

### What if I forget to add a changeset?

The CI will remind you! You can add a changeset in a follow-up commit:

```bash
npm run changeset
git add .changeset
git commit -m "chore: add changeset"
git push
```

### Can I edit a changeset?

Yes! Changeset files are just markdown. Edit them directly:

```bash
vim .changeset/funny-clouds-dance.md
git add .changeset
git commit -m "chore: update changeset description"
```

## Code Style

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting (automatic with most editors)
- Run `golangci-lint run` before committing
- Keep functions small and focused
- Add comments for exported functions and types

### Error Handling

- Use sentinel errors for expected conditions (see `internal/service/detector.go`)
- Wrap errors with context: `fmt.Errorf("failed to load config: %w", err)`
- Check errors with `errors.Is(err, sentinel)`

### Testing

- Write table-driven tests where appropriate
- Use subtests: `t.Run("test case name", func(t *testing.T) { ... })`
- Mock external dependencies (git commands, file system)
- Aim for high test coverage on core logic

## Project Structure

```
.
├── cmd/dual/              # CLI entry point
├── internal/
│   ├── config/            # Configuration loading
│   ├── context/           # Context detection
│   ├── registry/          # Global registry
│   └── service/           # Service detection and port calculation
├── test/integration/      # End-to-end tests
├── npm/                   # npm wrapper package
├── scripts/               # Helper scripts
└── .changeset/            # Changeset files
```

See `CLAUDE.md` for detailed architecture documentation.

## Running Tests

```bash
# Unit tests only
go test -v ./internal/...

# Integration tests only
go test -v ./test/integration/...

# All tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# With race detector
go test -race ./...
```

## Building

```bash
# Development build
go build -o dual ./cmd/dual

# Build with version info (like releases)
go build -ldflags="-X main.version=dev -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o dual ./cmd/dual

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o dual-linux ./cmd/dual
GOOS=darwin GOARCH=arm64 go build -o dual-macos ./cmd/dual
```

## Documentation

- Update `README.md` for user-facing changes
- Update `CLAUDE.md` for architecture changes
- Update `RELEASE.md` for release process changes
- Add inline code comments for complex logic

## Getting Help

- Check existing issues: https://github.com/lightfastai/dual/issues
- Read the documentation in the repository
- Ask questions in issues or discussions

## Code of Conduct

Be respectful, constructive, and professional. We're all here to make dual better!

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
