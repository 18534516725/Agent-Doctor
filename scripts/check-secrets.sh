#!/usr/bin/env sh
set -eu

if git grep -nE 'sk-[A-Za-z0-9_-]{16,}|Bearer [A-Za-z0-9._-]{16,}|api[_-]?key[[:space:]]*[:=][[:space:]]*[A-Za-z0-9._-]{16,}' -- ':!testdata' ':!**/*_test.go' ':!internal/privacy' ':!scripts/check-secrets.sh'; then
  echo "potential credential-shaped value found" >&2
  exit 1
fi
