#!/usr/bin/env bash
set -euo pipefail

URL="${1:?URL required}"
OUT="${2:?OUT path required}"

mkdir -p "$(dirname "$OUT")"

curl -s -X POST "${FLARESOLVERR_URL:-http://localhost:8191/v1}" \
  -H "Content-Type: application/json" \
  -d '{
    "cmd": "request.get",
    "url": "'"$URL"'",
    "maxTimeout": '"${SCRAPE_MAX_TIMEOUT_MS:-120000}"',
    "headers": {
      "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
      "Accept-Language": "en-US,en;q=0.9,pt-BR;q=0.8"
    }
  }' | jq -r '.solution.response' > "$OUT"

echo "Captured $(wc -c < "$OUT") bytes to $OUT"
