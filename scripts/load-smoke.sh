#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
ENDPOINT="${ENDPOINT:-/healthz}"
DURATION_SECONDS="${DURATION_SECONDS:-30}"
CONCURRENCY="${CONCURRENCY:-4}"
TOKEN="${TOKEN:-}"

deadline=$((SECONDS + DURATION_SECONDS))
results="$(mktemp)"
trap 'rm -f "$results"' EXIT

request() {
  while (( SECONDS < deadline )); do
    args=(-s -o /dev/null -w '%{http_code}\n' "${API_URL}${ENDPOINT}")
    if [[ -n "$TOKEN" ]]; then
      args+=(-H "Authorization: Bearer ${TOKEN}")
    fi
    curl "${args[@]}" >> "$results" || echo "000" >> "$results"
  done
}

for ((i = 0; i < CONCURRENCY; i++)); do
  request &
done
wait

echo "status counts for ${ENDPOINT}:"
sort "$results" | uniq -c | sort -nr
