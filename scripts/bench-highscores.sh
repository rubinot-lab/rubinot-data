#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://localhost:18080}"
WORLD="${2:-Belaria}"
CONCURRENCY="${3:-10}"

echo "=== Highscores Benchmark ==="
echo "API: $BASE_URL"
echo "World: $WORLD"
echo "Concurrency: $CONCURRENCY"
echo ""

echo "[1/3] Fetching categories..."
categories_json=$(curl -sf "$BASE_URL/v1/highscores/categories")
categories=$(echo "$categories_json" | python3 -c "
import json, sys
d = json.load(sys.stdin)
for c in d['categories']:
    print(c['slug'])
")
cat_count=$(echo "$categories" | wc -l | tr -d ' ')
echo "  Categories: $cat_count"

echo ""
echo "[2/3] Fetching page 1 of each category to discover total pages..."

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

echo "$categories" | xargs -P "$CONCURRENCY" -I{} bash -c '
  resp=$(curl -sf "'"$BASE_URL"'/v1/highscores/'"$WORLD"'/{}/all/1" 2>/dev/null || echo "{}")
  pages=$(echo "$resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get(\"highscores\",{}).get(\"highscore_page\",{}).get(\"total_pages\",0))" 2>/dev/null || echo "0")
  echo "{} $pages"
' > "$tmpdir/page_counts.txt"

total_requests=0
while IFS=' ' read -r slug pages; do
  for page in $(seq 1 "$pages"); do
    echo "$slug $page"
  done
  total_requests=$((total_requests + pages))
done < "$tmpdir/page_counts.txt" > "$tmpdir/requests.txt"

echo ""
echo "  Page counts per category:"
while IFS=' ' read -r slug pages; do
  printf "    %-25s %s pages\n" "$slug" "$pages"
done < <(sort "$tmpdir/page_counts.txt")

echo ""
echo "[3/3] Fetching all $total_requests pages (concurrency=$CONCURRENCY)..."

start=$(date +%s%N)

xargs -P "$CONCURRENCY" -I{} bash -c '
  slug=$(echo "{}" | cut -d" " -f1)
  page=$(echo "{}" | cut -d" " -f2)
  code=$(curl -sf -o /dev/null -w "%{http_code}" "'"$BASE_URL"'/v1/highscores/'"$WORLD"'/$slug/all/$page" 2>/dev/null || echo "000")
  echo "$code $slug $page"
' < "$tmpdir/requests.txt" > "$tmpdir/results.txt"

elapsed=$(( ($(date +%s%N) - start) / 1000000 ))

total_ok=$(grep -c "^200 " "$tmpdir/results.txt" || true)
total_err=$(grep -cv "^200 " "$tmpdir/results.txt" || true)

echo "  Done in ${elapsed}ms"
echo ""
echo "=== Results ==="
echo "  Categories:       $cat_count"
echo "  Total pages:      $total_requests"
echo "  Successful (200): $total_ok"
echo "  Failed:           $total_err"
echo "  Wall time:        ${elapsed}ms"
echo "  Avg per page:     $(( elapsed / (total_requests > 0 ? total_requests : 1) ))ms"
echo "  Throughput:       $(python3 -c "print(f'{$total_requests / ($elapsed / 1000):.1f}')" ) req/s"

if [ "$total_err" -gt 0 ]; then
  echo ""
  echo "Failed requests:"
  grep -v "^200 " "$tmpdir/results.txt" | head -20
fi
