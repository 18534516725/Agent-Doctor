#!/usr/bin/env sh
set -eu

required="README.md docs/install.md docs/privacy.md docs/cost-methodology.md docs/diagnosis-methodology.md docs/compatibility.md docs/troubleshooting.md examples/demo-data.json"
for file in $required; do
  test -s "$file" || { echo "missing documentation artifact: $file" >&2; exit 1; }
done

for command in setup start dashboard diagnose compare context costs doctor pause export forget run uninstall version; do
  grep -q "agent-doctor $command\|\`$command\`" README.md || { echo "README does not document $command" >&2; exit 1; }
done

for client in "Codex" "Claude Code" "Cline" "OpenCode" "Cursor" "Windsurf" "Roo Code" "Continue" "Aider" "Cherry Studio"; do
  grep -q "$client" docs/compatibility.md || { echo "compatibility matrix missing $client" >&2; exit 1; }
done

grep -q "exact" docs/cost-methodology.md
grep -q "estimated" docs/cost-methodology.md
grep -q "unavailable" docs/cost-methodology.md
grep -q "local" docs/privacy.md

if grep -RniE 'sk-[A-Za-z0-9_-]{16,}|Bearer [A-Za-z0-9._-]{16,}' README.md docs examples; then
  echo "documentation contains credential-shaped example" >&2
  exit 1
fi
