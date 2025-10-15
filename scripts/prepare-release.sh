#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

info() {
    echo -e "${BLUE}$1${NC}"
}

success() {
    echo -e "${GREEN}$1${NC}"
}

warning() {
    echo -e "${YELLOW}$1${NC}"
}

# Usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS] <version>

Prepare a new release for the dual project.

Arguments:
  version       Version to release (e.g., 1.2.3)

Options:
  --dry-run     Show what would be done without making changes
  -h, --help    Show this help message

Examples:
  $0 1.2.3
  $0 --dry-run 1.2.3
EOF
    exit 0
}

# Parse arguments
DRY_RUN=false
VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            if [[ -z "$VERSION" ]]; then
                VERSION=$1
            else
                error "Unexpected argument: $1"
            fi
            shift
            ;;
    esac
done

# Check if version is provided
if [[ -z "$VERSION" ]]; then
    error "Version is required. Usage: $0 <version>"
fi

# Validate version format
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
    error "Invalid version format. Expected: X.Y.Z or X.Y.Z-prerelease (e.g., 1.2.3 or 1.2.3-rc1)"
fi

TAG="v${VERSION}"

info "=== Dual Release Preparation ==="
echo ""
info "Version: ${VERSION}"
info "Tag:     ${TAG}"
if [[ "$DRY_RUN" == true ]]; then
    warning "DRY RUN MODE - No changes will be made"
fi
echo ""

# Pre-flight checks
info "Running pre-flight checks..."

# Check if we're in the right directory
if [[ ! -f "go.mod" ]] || ! grep -q "module github.com/lightfastai/dual" go.mod; then
    error "Must run from dual project root directory"
fi

# Check if git is clean
if [[ "$DRY_RUN" == false ]] && [[ -n $(git status --porcelain) ]]; then
    error "Git working directory is not clean. Commit or stash changes first."
fi

# Check if on main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    warning "Not on main branch (current: ${CURRENT_BRANCH}). Continuing anyway..."
fi

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
    error "Tag ${TAG} already exists locally"
fi

# Check if tag exists on remote
if git ls-remote --tags origin | grep -q "refs/tags/${TAG}$"; then
    error "Tag ${TAG} already exists on remote"
fi

# Check if npm package version already exists
info "Checking npm registry..."
if npm view "@lightfastai/dual@${VERSION}" version 2>/dev/null; then
    error "Version ${VERSION} already published to npm"
fi

success "Pre-flight checks passed"
echo ""

# Show current state
info "Current state:"
echo "  Git branch: ${CURRENT_BRANCH}"
echo "  Latest commit: $(git log -1 --oneline)"
echo "  Current npm version: $(jq -r .version npm/package.json)"
echo ""

# Show what will be changed
info "Changes to be made:"
echo "  1. Update npm/package.json version to ${VERSION}"
echo "  2. Create git commit: 'chore: bump version to ${TAG}'"
echo "  3. Create annotated git tag: '${TAG}'"
echo ""

# Ask for confirmation unless dry-run
if [[ "$DRY_RUN" == false ]]; then
    read -p "Proceed with release preparation? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        warning "Aborted by user"
        exit 0
    fi
    echo ""
fi

# Update npm package.json
info "Updating npm/package.json..."
if [[ "$DRY_RUN" == true ]]; then
    info "Would run: cd npm && npm version ${VERSION} --no-git-tag-version"
else
    cd npm
    npm version "${VERSION}" --no-git-tag-version --allow-same-version
    cd ..
    success "Updated npm/package.json to ${VERSION}"
fi
echo ""

# Create git commit
info "Creating git commit..."
COMMIT_MSG="chore: bump version to ${TAG}"
if [[ "$DRY_RUN" == true ]]; then
    info "Would run: git add npm/package.json"
    info "Would run: git commit -m '${COMMIT_MSG}'"
else
    git add npm/package.json
    git commit -m "${COMMIT_MSG}"
    success "Created commit"
fi
echo ""

# Create git tag
info "Creating git tag..."
TAG_MSG="Release ${TAG}"
if [[ "$DRY_RUN" == true ]]; then
    info "Would run: git tag -a ${TAG} -m '${TAG_MSG}'"
else
    git tag -a "${TAG}" -m "${TAG_MSG}"
    success "Created tag ${TAG}"
fi
echo ""

# Show next steps
success "=== Release preparation complete! ==="
echo ""
info "Next steps:"
echo ""
echo "  1. Review the changes:"
echo "     ${BLUE}git show${NC}"
echo "     ${BLUE}git log --oneline -3${NC}"
echo ""
echo "  2. Push the commit and tag to trigger the release:"
echo "     ${GREEN}git push origin main${NC}"
echo "     ${GREEN}git push origin ${TAG}${NC}"
echo ""
echo "  3. Monitor the release workflow:"
echo "     ${BLUE}gh run watch${NC}"
echo "     Or visit: https://github.com/lightfastai/dual/actions"
echo ""
echo "  4. Verify the release succeeded:"
echo "     - GitHub: https://github.com/lightfastai/dual/releases"
echo "     - Homebrew: https://github.com/lightfastai/homebrew-tap"
echo "     - npm: ${BLUE}npm view @lightfastai/dual@${VERSION}${NC}"
echo ""

if [[ "$DRY_RUN" == true ]]; then
    warning "This was a DRY RUN - no changes were made"
else
    warning "Don't forget to push! The release will only trigger after pushing the tag."
fi
