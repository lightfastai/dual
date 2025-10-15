#!/usr/bin/env bash
#
# demo.sh - Automated demonstration of dual multi-worktree management
#
# This script demonstrates the complete lifecycle of using dual with
# multiple worktrees and custom port assignment via hooks.

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKTREES_DIR="${EXAMPLE_DIR}/../worktrees"
PROJECT_NAME="multi-worktree-example"

# Helper functions
print_header() {
  echo ""
  echo -e "${BLUE}${'='*60}${NC}"
  echo -e "${BLUE}$1${NC}"
  echo -e "${BLUE}${'='*60}${NC}"
  echo ""
}

print_step() {
  echo -e "${GREEN}[DEMO]${NC} $1"
}

print_info() {
  echo -e "${YELLOW}[INFO]${NC} $1"
}

print_error() {
  echo -e "${RED}[ERROR]${NC} $1"
}

pause_for_user() {
  echo ""
  echo -e "${YELLOW}Press Enter to continue...${NC}"
  read -r
}

# Check if dual is installed
check_dual() {
  if ! command -v dual &> /dev/null; then
    print_error "dual command not found"
    print_info "Please install dual first:"
    print_info "  cd /path/to/dual"
    print_info "  go install ./cmd/dual"
    exit 1
  fi

  print_step "Found dual: $(which dual)"
}

# Cleanup function
cleanup() {
  print_header "Cleanup"

  cd "${EXAMPLE_DIR}"

  # Delete worktrees if they exist
  for context in dev feature-1 feature-2; do
    if dual context list 2>/dev/null | grep -q "^${context}$"; then
      print_step "Deleting worktree: ${context}"
      dual delete "${context}" || true
    fi
  done

  # Remove worktrees directory
  if [ -d "${WORKTREES_DIR}" ]; then
    print_step "Removing worktrees directory: ${WORKTREES_DIR}"
    rm -rf "${WORKTREES_DIR}"
  fi

  # Remove .dual directory (will be recreated)
  if [ -d "${EXAMPLE_DIR}/.dual/registry.json" ]; then
    print_step "Removing registry"
    rm -f "${EXAMPLE_DIR}/.dual/registry.json"
    rm -f "${EXAMPLE_DIR}/.dual/registry.json.lock"
  fi

  print_step "Cleanup complete"
}

# Initialize git repository
init_git() {
  print_header "Step 1: Initialize Git Repository"

  cd "${EXAMPLE_DIR}"

  if [ -d .git ]; then
    print_step "Git repository already exists"
  else
    print_step "Initializing git repository..."
    git init
    git add .
    git commit -m "Initial commit: Multi-worktree example" || true
  fi

  print_step "Git repository ready"
}

# Verify dual configuration
verify_config() {
  print_header "Step 2: Verify Dual Configuration"

  cd "${EXAMPLE_DIR}"

  print_step "Checking dual.config.yml..."
  if [ ! -f dual.config.yml ]; then
    print_error "dual.config.yml not found!"
    exit 1
  fi

  print_step "Configuration found:"
  cat dual.config.yml

  print_step "Checking hook scripts..."
  if [ ! -x .dual/hooks/assign-ports.sh ]; then
    print_step "Making assign-ports.sh executable..."
    chmod +x .dual/hooks/assign-ports.sh
  fi

  if [ ! -x .dual/hooks/setup-environment.sh ]; then
    print_step "Making setup-environment.sh executable..."
    chmod +x .dual/hooks/setup-environment.sh
  fi

  print_step "Hook scripts ready"
}

# Create worktrees
create_worktrees() {
  print_header "Step 3: Create Worktrees"

  cd "${EXAMPLE_DIR}"

  local contexts=("dev" "feature-1" "feature-2")

  for context in "${contexts[@]}"; do
    print_step "Creating worktree: ${context}"
    echo ""

    dual create "${context}"

    echo ""
    print_step "Worktree ${context} created successfully"
    echo ""
  done
}

# Show registry contents
show_registry() {
  print_header "Step 4: Show Registry Contents"

  cd "${EXAMPLE_DIR}"

  print_step "Registry file: ${EXAMPLE_DIR}/.dual/registry.json"
  echo ""

  if [ -f .dual/registry.json ]; then
    cat .dual/registry.json | python3 -m json.tool 2>/dev/null || cat .dual/registry.json
  else
    print_error "Registry file not found!"
  fi

  echo ""
}

# Show port assignments
show_port_assignments() {
  print_header "Step 5: Port Assignments"

  local contexts=("dev" "feature-1" "feature-2")

  for context in "${contexts[@]}"; do
    local worktree_path="${WORKTREES_DIR}/${context}"

    if [ -f "${worktree_path}/.dual-port" ]; then
      print_step "Port assignment for ${context}:"
      echo ""
      cat "${worktree_path}/.dual-port"
      echo ""
    else
      print_error "Port file not found for ${context}"
    fi
  done
}

# Show environment files
show_environment_files() {
  print_header "Step 6: Environment Files"

  local contexts=("dev" "feature-1" "feature-2")

  for context in "${contexts[@]}"; do
    local worktree_path="${WORKTREES_DIR}/${context}"

    if [ -f "${worktree_path}/.env.local" ]; then
      print_step "Environment file for ${context}:"
      echo ""
      cat "${worktree_path}/.env.local"
      echo ""
    else
      print_error "Environment file not found for ${context}"
    fi
  done
}

# Show port uniqueness
verify_port_uniqueness() {
  print_header "Step 7: Verify Port Uniqueness"

  local contexts=("dev" "feature-1" "feature-2")
  local all_ports=()

  print_step "Collecting all assigned ports..."
  echo ""

  for context in "${contexts[@]}"; do
    local worktree_path="${WORKTREES_DIR}/${context}"

    if [ -f "${worktree_path}/.dual-port" ]; then
      # shellcheck source=/dev/null
      source "${worktree_path}/.dual-port"

      echo -e "${context}:"
      echo "  Base:   ${BASE_PORT}"
      echo "  Web:    ${WEB_PORT}"
      echo "  API:    ${API_PORT}"
      echo "  Worker: ${WORKER_PORT}"

      all_ports+=("${BASE_PORT}" "${WEB_PORT}" "${API_PORT}" "${WORKER_PORT}")
    fi
  done

  echo ""
  print_step "Checking for port conflicts..."

  # Check for duplicates
  local unique_ports=($(printf '%s\n' "${all_ports[@]}" | sort -u))

  if [ ${#all_ports[@]} -eq ${#unique_ports[@]} ]; then
    print_step "${GREEN}✓ All ports are unique! No conflicts detected.${NC}"
  else
    print_error "✗ Port conflicts detected!"
    exit 1
  fi

  echo ""
}

# Show worktree structure
show_worktree_structure() {
  print_header "Step 8: Worktree Directory Structure"

  if [ -d "${WORKTREES_DIR}" ]; then
    print_step "Worktrees directory: ${WORKTREES_DIR}"
    echo ""

    # Use tree if available, otherwise use find
    if command -v tree &> /dev/null; then
      tree -L 3 -a "${WORKTREES_DIR}"
    else
      find "${WORKTREES_DIR}" -maxdepth 3 -print | sed 's|[^/]*/|  |g'
    fi

    echo ""
  else
    print_error "Worktrees directory not found!"
  fi
}

# Demonstrate running a service
demonstrate_service() {
  print_header "Step 9: Service Demonstration"

  print_info "To run services in any worktree:"
  echo ""
  echo "  # Terminal 1 - Dev worktree web service"
  echo "  cd ${WORKTREES_DIR}/dev/apps/web"
  echo "  npm install"
  echo "  npm start"
  echo ""
  echo "  # Terminal 2 - Feature-1 worktree web service"
  echo "  cd ${WORKTREES_DIR}/feature-1/apps/web"
  echo "  npm install"
  echo "  npm start"
  echo ""
  echo "  # Both services will run on different ports without conflicts!"
  echo ""

  print_info "Each service will display its context and assigned port on startup."
}

# Main demonstration flow
main() {
  print_header "Dual Multi-Worktree Example - Automated Demo"

  print_info "This demo will:"
  print_info "  1. Initialize a git repository"
  print_info "  2. Verify dual configuration"
  print_info "  3. Create 3 worktrees (dev, feature-1, feature-2)"
  print_info "  4. Show the registry contents"
  print_info "  5. Display port assignments"
  print_info "  6. Show generated environment files"
  print_info "  7. Verify port uniqueness"
  print_info "  8. Show worktree structure"
  print_info "  9. Demonstrate running services"
  print_info "  10. Clean up"

  echo ""
  print_info "The demo will pause at each step for you to review."

  pause_for_user

  # Check prerequisites
  check_dual

  # Run demo steps
  init_git
  pause_for_user

  verify_config
  pause_for_user

  create_worktrees
  pause_for_user

  show_registry
  pause_for_user

  show_port_assignments
  pause_for_user

  show_environment_files
  pause_for_user

  verify_port_uniqueness
  pause_for_user

  show_worktree_structure
  pause_for_user

  demonstrate_service
  pause_for_user

  # Ask if user wants to clean up
  print_header "Cleanup"
  echo ""
  echo -e "${YELLOW}Do you want to clean up the created worktrees? (y/N)${NC}"
  read -r response

  if [[ "$response" =~ ^[Yy]$ ]]; then
    cleanup
  else
    print_info "Worktrees preserved. You can explore them or clean up later with:"
    print_info "  ./demo.sh cleanup"
  fi

  print_header "Demo Complete!"
  print_step "Thank you for exploring the dual multi-worktree example!"
}

# Allow running cleanup directly
if [ "$1" = "cleanup" ]; then
  cleanup
  exit 0
fi

# Run main demo
main
