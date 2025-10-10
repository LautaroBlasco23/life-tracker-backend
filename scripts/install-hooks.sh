#!/bin/bash
set -e

HOOKS_DIR=".git/hooks"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🔧 Installing Git hooks..."

if [ ! -d "$PROJECT_ROOT/.git" ]; then
  echo "❌ Error: .git directory not found. Are you in the project root?"
  exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/$HOOKS_DIR"

cat >"$PROJECT_ROOT/$HOOKS_DIR/pre-push" <<'EOF'
#!/bin/sh
echo "🎯 Running pre-push checks..."

# Find the Makefile location
if [ -f "Makefile" ]; then
    MAKEFILE_DIR="."
elif [ -f "backend/Makefile" ]; then
    MAKEFILE_DIR="backend"
else
    echo "❌ Makefile not found"
    exit 1
fi

cd "$MAKEFILE_DIR" || exit 1

if ! make code-check; then
    echo "❌ Code check failed. Push aborted."
    exit 1
fi

echo "✅ Pre-push checks passed!"
EOF

chmod +x "$PROJECT_ROOT/$HOOKS_DIR/pre-push"

echo "✅ Git hook installed!"
echo ""
echo "Pre-push hook will run: make code-check"
echo ""
echo "To skip the hook (not recommended):"
echo "  git push --no-verify"
