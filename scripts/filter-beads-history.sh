#!/usr/bin/env bash
#
# filter-beads-history.sh — Remove .beads/ directory from gastown repo history
#
# The .beads/ directory contains runtime beads data (issue tracking state,
# formulas, backup data) that was committed to the repo before .gitignore
# rules were added. This script scrubs .beads/ from the entire history.
#
# PREREQUISITES:
#   - git-filter-repo installed (pip install git-filter-repo, or download
#     https://raw.githubusercontent.com/newren/git-filter-repo/main/git-filter-repo)
#   - Run on a FRESH CLONE (filter-repo requires this)
#   - No in-flight polecats or active worktrees (history rewrite changes all SHAs)
#
# USAGE:
#   git clone <gastown-repo-url> gastown-clean
#   cd gastown-clean
#   bash scripts/filter-beads-history.sh
#
# After running, force-push the rewritten history:
#   git push origin --force --all
#   git push origin --force --tags
#
# All collaborators must re-clone or rebase onto the rewritten history.

set -euo pipefail

# Verify we're in a git repo
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    echo "ERROR: Not in a git repository" >&2
    exit 1
fi

# Verify git-filter-repo is available
if ! command -v git-filter-repo >/dev/null 2>&1; then
    echo "ERROR: git-filter-repo not found. Install with: pip install git-filter-repo" >&2
    exit 1
fi

# Count affected commits
affected=$(git log --oneline --all -- '.beads/' | wc -l)
total=$(git rev-list --count --all)
echo "Will rewrite history to remove .beads/ from $affected of $total commits."
echo ""

# Confirm
read -p "This rewrites ALL commit SHAs. Continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo "Running git-filter-repo..."
git-filter-repo --path .beads/ --invert-paths

echo ""
echo "Done. .beads/ has been removed from all history."
echo ""
echo "Next steps:"
echo "  1. Verify: git log --all --name-only | grep '^\\.beads/' | wc -l  (should be 0)"
echo "  2. Force-push: git push origin --force --all && git push origin --force --tags"
echo "  3. All collaborators must re-clone or: git fetch origin && git reset --hard origin/main"
