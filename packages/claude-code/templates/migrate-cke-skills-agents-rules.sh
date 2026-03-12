#!/usr/bin/env bash
# migrate-cke.sh — Copy and adapt CKE skills/agents/rules to Solon templates
# String replacements:
#   ck: → solon: (skill prefix only, not "slack:" etc.)
#   .ck.json → .solon.json
#   CK_SESSION → SOLON_SESSION
#   CK_REPORTS_PATH → SOLON_REPORTS_PATH
#   ~/.claude/skills/.venv → ~/.solon/venv

set -euo pipefail

SRC_SKILLS="/home/phuc/.claude/skills"
SRC_AGENTS="/home/phuc/.claude/agents"
SRC_RULES="/home/phuc/.claude/rules"
DEST="/home/phuc/Projects/solon/packages/claude-code/templates"

# ─── Apply string replacements to all text files in a directory ───────────────
apply_replacements() {
  local dir="$1"
  find "$dir" -type f \( -name "*.md" -o -name "*.json" -o -name "*.ts" -o -name "*.js" -o -name "*.txt" \) | while read -r f; do
    sed -i \
      -e 's|\.ck\.json|.solon.json|g' \
      -e 's|CK_SESSION|SOLON_SESSION|g' \
      -e 's|CK_REPORTS_PATH|SOLON_REPORTS_PATH|g' \
      -e 's|~/.claude/skills/\.venv|~/.solon/venv|g' \
      -e 's|/ck:|\x00PROTECTED_SLASH_CK\x00|g' \
      -e 's|`ck:|`solon:|g' \
      -e 's|/ck:|/solon:|g' \
      -e 's|\bck:\([a-z_-]\)|\bsolon:\1|g' \
      "$f" 2>/dev/null || true
    # Restore slash-based protected patterns that should NOT be changed (like "slack:")
    sed -i 's|\x00PROTECTED_SLASH_CK\x00|/ck:|g' "$f" 2>/dev/null || true
  done
}

# ─── Simpler replacement using perl (more reliable for word boundaries) ────────
apply_replacements_perl() {
  local dir="$1"
  find "$dir" -type f \( -name "*.md" -o -name "*.json" -o -name "*.ts" -o -name "*.js" -o -name "*.txt" \) | while read -r f; do
    perl -i \
      -e 's/\.ck\.json/.solon.json/g;' \
      -e 's/CK_SESSION/SOLON_SESSION/g;' \
      -e 's/CK_REPORTS_PATH/SOLON_REPORTS_PATH/g;' \
      -e 's|~/.claude/skills/\.venv|~/.solon/venv|g;' \
      -e 's|`ck:|`solon:|g;' \
      -e 's|/ck:|/solon:|g;' \
      -e 's/(?<![a-z])ck:(?=[a-z])/solon:/g;' \
      "$f" 2>/dev/null || true
  done
}

echo "=== Creating directory structure ==="
mkdir -p "$DEST/skills/core"
mkdir -p "$DEST/skills/utility"
mkdir -p "$DEST/skills/optional/frontend"
mkdir -p "$DEST/skills/optional/backend"
mkdir -p "$DEST/skills/optional/mobile"
mkdir -p "$DEST/skills/optional/ai-tools"
mkdir -p "$DEST/skills/optional/devops"
mkdir -p "$DEST/skills/optional/media"
mkdir -p "$DEST/agents"
mkdir -p "$DEST/rules"

echo ""
echo "=== Copying CORE skills (12) ==="
CORE_SKILLS=(cook fix plan debug test research scout docs git brainstorm code-review bootstrap)
for skill in "${CORE_SKILLS[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/core/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying UTILITY skills (7 specified + extras) ==="
UTILITY_SKILLS=(sequential-thinking ask watzup preview context-engineering find-skills problem-solving)
for skill in "${UTILITY_SKILLS[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/utility/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying remaining UTILITY skills (solon-specific) ==="
SOLON_UTILITY=(skill-creator coding-level journal markdown-novel-viewer docs-seeker gkg tanstack mintlify)
for skill in "${SOLON_UTILITY[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/utility/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying OPTIONAL/frontend skills ==="
FRONTEND_SKILLS=(frontend-development react-best-practices ui-styling frontend-design web-frameworks web-design-guidelines web-testing)
for skill in "${FRONTEND_SKILLS[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/optional/frontend/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying OPTIONAL/backend skills ==="
BACKEND_SKILLS=(backend-development databases better-auth shopify payment-integration)
for skill in "${BACKEND_SKILLS[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/optional/backend/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying OPTIONAL/mobile skills ==="
MOBILE_SKILLS=(mobile-development)
for skill in "${MOBILE_SKILLS[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/optional/mobile/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying OPTIONAL/ai-tools skills ==="
AI_TOOLS=(ai-multimodal ai-artist google-adk-python use-mcp mcp-management mcp-builder)
# claude-api may not exist, check individually
for skill in "${AI_TOOLS[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/optional/ai-tools/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying OPTIONAL/devops skills ==="
DEVOPS_SKILLS=(devops kanban plans-kanban repomix project-management worktree agent-browser chrome-devtools)
for skill in "${DEVOPS_SKILLS[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/optional/devops/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying OPTIONAL/media skills ==="
MEDIA_SKILLS=(media-processing threejs shader remotion)
for skill in "${MEDIA_SKILLS[@]}"; do
  if [ -d "$SRC_SKILLS/$skill" ]; then
    cp -r "$SRC_SKILLS/$skill" "$DEST/skills/optional/media/"
    echo "  ✓ $skill"
  else
    echo "  ✗ MISSING: $skill"
  fi
done

echo ""
echo "=== Copying AGENTS ==="
for agent in "$SRC_AGENTS"/*.md; do
  name=$(basename "$agent")
  cp "$agent" "$DEST/agents/$name"
  echo "  ✓ $name"
done

echo ""
echo "=== Copying RULES ==="
for rule in "$SRC_RULES"/*.md; do
  name=$(basename "$rule")
  cp "$rule" "$DEST/rules/$name"
  echo "  ✓ $name"
done

echo ""
echo "=== Applying string replacements to all text files ==="
apply_replacements_perl "$DEST/skills"
apply_replacements_perl "$DEST/rules"
apply_replacements_perl "$DEST/agents"
echo "  Done."

echo ""
echo "=== Migration complete ==="
