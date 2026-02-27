#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://localhost:18080}"
WORLD="${2:-Belaria}"
CONCURRENCY="${3:-10}"

echo "=== Online Characters Benchmark ==="
echo "API: $BASE_URL"
echo "World: $WORLD"
echo "Concurrency: $CONCURRENCY"
echo ""

start_total=$(date +%s%N)

echo "[1/2] Fetching world online list..."
start=$(date +%s%N)
world_json=$(curl -sf "$BASE_URL/v1/world/$WORLD")
elapsed=$(( ($(date +%s%N) - start) / 1000000 ))
echo "  Done in ${elapsed}ms"

names=$(echo "$world_json" | python3 -c "
import json, sys
d = json.load(sys.stdin)
players = d['world']['players_online']
for p in players:
    print(p['name'])
")

count=$(echo "$names" | wc -l | tr -d ' ')
echo "  Players online: $count"
echo ""

echo "[2/2] Fetching each character ($count requests, concurrency=$CONCURRENCY)..."

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

echo "$names" > "$tmpdir/names.txt"

start=$(date +%s%N)

xargs -P "$CONCURRENCY" -I{} bash -c '
  name=$(python3 -c "import urllib.parse; print(urllib.parse.quote(\"{}\")")
  code=$(curl -sf -o /dev/null -w "%{http_code}" "'"$BASE_URL"'/v1/character/$name" 2>/dev/null || echo "000")
  echo "$code {}"
' < "$tmpdir/names.txt" > "$tmpdir/results.txt"

elapsed=$(( ($(date +%s%N) - start) / 1000000 ))

total_ok=$(grep -c "^200 " "$tmpdir/results.txt" || true)
total_err=$(grep -cv "^200 " "$tmpdir/results.txt" || true)

echo "  Done in ${elapsed}ms"
echo ""
echo "=== Results ==="
echo "  Total characters: $count"
echo "  Successful (200): $total_ok"
echo "  Failed:           $total_err"
echo "  Wall time:        ${elapsed}ms"
echo "  Avg per char:     $(( elapsed / count ))ms"
echo "  Throughput:       $(python3 -c "print(f'{$count / ($elapsed / 1000):.1f}')" ) req/s"

elapsed_total=$(( ($(date +%s%N) - start_total) / 1000000 ))
echo "  Total wall time:  ${elapsed_total}ms"

if [ "$total_err" -gt 0 ]; then
  echo ""
  echo "Failed requests:"
  grep -v "^200 " "$tmpdir/results.txt" | head -20
fi
