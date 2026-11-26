#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-.}"
PATTERN='(JWT_SECRET|PASSWORD|SECRET|API_KEY|TOKEN)[[:space:]]*[:=][[:space:]]*[^[:space:]#]+'

matches="$(grep -RInE "$PATTERN" \
  --exclude-dir=.git \
  --exclude-dir=node_modules \
  --exclude-dir=dist \
  --exclude-dir=build \
  --exclude='*.dump' \
  --exclude='*.example' \
  --exclude='scan-secrets.sh' \
  --exclude='*.md' \
  "$ROOT" | grep -viE 'local-development|replace|change|example|placeholder|pharmaos|test|xxx|\$\{' || true)"

if [[ -n "$matches" ]]; then
  echo "$matches"
  exit 1
fi

echo "No inline secret assignments found."
