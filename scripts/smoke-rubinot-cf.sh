#!/usr/bin/env bash
set -euo pipefail

BASE="${1:-https://api.rubinot.dev}"
WORLD="${2:-Belaria}"
TOWN="${3:-Venore}"
HOUSE_ID="${4:-70}"
TIMEOUT="${TIMEOUT:-20}"

# Cloudflare-safe defaults
UA="Mozilla/5.0 (X11; Linux x86_64)"

# Highscore categories (as exposed by validator)
CATEGORIES=(
  achievements battle-pass bounty axe club charm distance drome linked-tasks experience daily-xp fishing fist loyalty magic prestige shielding sword weekly-tasks
)
# highscore vocations
VOCATIONS=("(all)" none knights paladins sorcerers druids monks)

ROUTES=(
  "/"
  "/ping"
  "/healthz"
  "/readyz"
  "/versions"
  "/metrics"
  "/v1/worlds"
  "/v1/world/${WORLD}"
  "/v1/highscores/${WORLD}"
  "/v1/highscores/${WORLD}/experience"
  "/v1/highscores/${WORLD}/experience/all/1"
  "/v1/killstatistics/${WORLD}"
  "/v1/news/id/1"
  "/v1/news/archive"
  "/v1/news/latest"
  "/v1/news/newsticker"
  "/v1/events/schedule"
  "/v1/auctions/current/1"
  "/v1/auctions/history/1"
  "/v1/auctions/1"
  "/v1/deaths/${WORLD}"
  "/v1/banishments/${WORLD}"
  "/v1/transfers"
  "/v1/character/Prensa"
  "/v1/guild/Ascended%20Belaria"
  "/v1/guilds/${WORLD}"
  "/v1/houses/${WORLD}/${TOWN}"
  "/v1/house/${WORLD}/${HOUSE_ID}"
)

fetch_one() {
  local path="$1"
  local url="${BASE}${path}"
  local tmp
  tmp="$(mktemp)"

  local code redirs ttfb total bytes
  read -r code redirs ttfb total bytes < <(curl -ksS -L --max-time "${TIMEOUT}" \
    -A "${UA}" \
    -H "Cache-Control: no-cache" -H "Pragma: no-cache" \
    -w '%{http_code} %{num_redirects} %{time_starttransfer} %{time_total} %{size_download}\n' \
    "${url}" -o "${tmp}" | tr -d '\r')

  printf "\n== %s ==\n" "${path}"
  printf "status=%s redirs=%s ttfb=%ss total=%ss bytes=%s\n" "${code}" "${redirs}" "${ttfb}" "${total}" "${bytes}"

  if command -v jq >/dev/null 2>&1 && jq -e . "${tmp}" >/dev/null 2>&1; then
    printf "json_top_level_keys=%s\n" "$(jq -c 'keys' "${tmp}")"
    printf "preview=%s\n" "$(jq -c '.' "${tmp}" | head -c 260)"
  else
    echo "json_top_level_keys=non-json-or-invalid"
    echo -n "preview="
    head -c 260 "${tmp}" | tr '\n' ' '
    echo
  fi

  rm -f "${tmp}"
}

for p in "${ROUTES[@]}"; do
  fetch_one "${p}"
done

echo

echo "== highscores: all categories + vocations =="
for category in "${CATEGORIES[@]}"; do
  for voc in "${VOCATIONS[@]}"; do
    p="/v1/highscores/${WORLD}/${category}/${voc}/1"
    fetch_one "${p}"
  done
done

