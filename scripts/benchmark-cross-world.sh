#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C

BASE_URL="${1:-https://data.rubinot.dev}"

WORLDS=(
  "Auroria" "Belaria" "Bellum" "Elysian" "Lunarian" "Mystian"
  "Serenian%20I" "Serenian%20II" "Serenian%20III" "Serenian%20IV"
  "Solarian" "Spectrum" "Tenebrium" "Vesperia"
)

ENDPOINTS=(
  "killstatistics"
  "deaths"
  "banishments"
  "guilds"
)

DEATH_SUFFIX="/all"
BANISHMENT_SUFFIX="/all"
GUILD_SUFFIX="/all"
KILLSTAT_SUFFIX=""

printf "\n=== rubinot-data cross-world benchmark ===\n"
printf "Target: %s\n" "$BASE_URL"
printf "Worlds: %d\n" "${#WORLDS[@]}"
printf "Date:   %s\n\n" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

benchmark_endpoint() {
  local endpoint="$1"
  local suffix="$2"
  local label="$3"

  printf -- "--- %s ---\n" "$label"

  local total_time=0
  local total_bytes=0
  local errors=0
  local times=()

  for world in "${WORLDS[@]}"; do
    local url="${BASE_URL}/v1/${endpoint}/${world}${suffix}"
    local result
    result=$(curl -s -o /dev/null -w "%{http_code} %{size_download} %{time_total}" "$url" 2>/dev/null || echo "000 0 0")
    local code size time_s
    code=$(echo "$result" | awk '{print $1}')
    size=$(echo "$result" | awk '{print $2}')
    time_s=$(echo "$result" | awk '{print $3}')

    if [ "$code" != "200" ]; then
      errors=$((errors + 1))
      printf "  %-20s %s (HTTP %s)\n" "$(echo "$world" | sed 's/%20/ /g')" "FAIL" "$code"
    else
      total_bytes=$(echo "$total_bytes + $size" | bc)
      total_time=$(echo "$total_time + $time_s" | bc)
      times+=("$time_s")
      printf "  %-20s %6.0f KB  %5.2fs\n" "$(echo "$world" | sed 's/%20/ /g')" "$(echo "$size / 1024" | bc -l)" "$time_s"
    fi
  done

  local count=${#times[@]}
  if [ "$count" -gt 0 ]; then
    local avg
    avg=$(echo "$total_time / $count" | bc -l)
    local min max
    min=$(printf '%s\n' "${times[@]}" | sort -n | head -1)
    max=$(printf '%s\n' "${times[@]}" | sort -n | tail -1)

    printf "\n"
    printf "  Sequential total:  %6.2fs (sum of all requests)\n" "$total_time"
    printf "  Avg per world:     %6.2fs\n" "$avg"
    printf "  Min/Max:           %5.2fs / %5.2fs\n" "$min" "$max"
    printf "  Total payload:     %6.0f KB\n" "$(echo "$total_bytes / 1024" | bc -l)"
    printf "  Errors:            %d\n" "$errors"
  fi
  printf "\n"

  echo "$total_time $total_bytes $count $errors"
}

benchmark_parallel() {
  local endpoint="$1"
  local suffix="$2"
  local label="$3"

  printf -- "--- %s (parallel, 14 concurrent) ---\n" "$label"

  local tmpdir
  tmpdir=$(mktemp -d)
  local pids=()

  local wall_start
  wall_start=$(python3 -c "import time; print(time.time())")

  for i in "${!WORLDS[@]}"; do
    local world="${WORLDS[$i]}"
    local url="${BASE_URL}/v1/${endpoint}/${world}${suffix}"
    curl -s -o /dev/null -w "%{http_code} %{size_download} %{time_total}" "$url" > "${tmpdir}/${i}.txt" 2>/dev/null &
    pids+=($!)
  done

  for pid in "${pids[@]}"; do
    wait "$pid" 2>/dev/null || true
  done

  local wall_end
  wall_end=$(python3 -c "import time; print(time.time())")
  local wall_time
  wall_time=$(echo "$wall_end - $wall_start" | bc)

  local total_bytes=0
  local errors=0
  local max_time=0

  for i in "${!WORLDS[@]}"; do
    local result
    result=$(cat "${tmpdir}/${i}.txt" 2>/dev/null || echo "000 0 0")
    local code size time_s
    code=$(echo "$result" | awk '{print $1}')
    size=$(echo "$result" | awk '{print $2}')
    time_s=$(echo "$result" | awk '{print $3}')

    if [ "$code" != "200" ]; then
      errors=$((errors + 1))
    else
      total_bytes=$(echo "$total_bytes + $size" | bc)
      if (( $(echo "$time_s > $max_time" | bc -l) )); then
        max_time=$time_s
      fi
    fi
  done

  printf "  Wall-clock time:   %6.2fs\n" "$wall_time"
  printf "  Slowest request:   %6.2fs\n" "$max_time"
  printf "  Total payload:     %6.0f KB\n" "$(echo "$total_bytes / 1024" | bc -l)"
  printf "  Errors:            %d\n" "$errors"
  printf "\n"

  rm -rf "$tmpdir"
  echo "$wall_time $total_bytes $max_time $errors"
}

benchmark_single_all() {
  local path="$1"
  local label="$2"

  printf -- "--- %s (single /all request) ---\n" "$label"

  local url="${BASE_URL}${path}"
  local result
  result=$(curl -s -o /dev/null -w "%{http_code} %{size_download} %{time_total}" "$url" 2>/dev/null || echo "000 0 0")
  local code size time_s
  code=$(echo "$result" | awk '{print $1}')
  size=$(echo "$result" | awk '{print $2}')
  time_s=$(echo "$result" | awk '{print $3}')

  if [ "$code" != "200" ]; then
    printf "  Status:            HTTP %s (not yet implemented)\n" "$code"
  else
    printf "  Response time:     %6.2fs\n" "$time_s"
    printf "  Payload size:      %6.0f KB\n" "$(echo "$size / 1024" | bc -l)"
  fi
  printf "\n"

  echo "$code $size $time_s"
}

printf "========================================\n"
printf " PHASE 1: Current per-world (sequential)\n"
printf "========================================\n\n"

benchmark_endpoint "killstatistics" "" "killstatistics/:world"
benchmark_endpoint "deaths" "/all" "deaths/:world/all"
benchmark_endpoint "banishments" "/all" "banishments/:world/all"
benchmark_endpoint "guilds" "/all" "guilds/:world/all"

printf "========================================\n"
printf " PHASE 2: Current per-world (parallel)\n"
printf "========================================\n\n"

benchmark_parallel "killstatistics" "" "killstatistics/:world"
benchmark_parallel "deaths" "/all" "deaths/:world/all"
benchmark_parallel "banishments" "/all" "banishments/:world/all"
benchmark_parallel "guilds" "/all" "guilds/:world/all"

printf "========================================\n"
printf " PHASE 3: Cross-world /all endpoints\n"
printf " (tests if they exist yet)\n"
printf "========================================\n\n"

benchmark_single_all "/v1/killstatistics/all" "killstatistics/all"
benchmark_single_all "/v1/deaths/all/all" "deaths/all/all"
benchmark_single_all "/v1/banishments/all/all" "banishments/all/all"
benchmark_single_all "/v1/guilds/all/all" "guilds/all/all"

printf "========================================\n"
printf " PHASE 4: Already-working cross-world\n"
printf "========================================\n\n"

benchmark_single_all "/v1/world/all/details" "world/all/details"
benchmark_single_all "/v1/auctions/current/all/details" "auctions/current/all/details"

printf "\n=== Done. Re-run after deploying /all endpoints to compare. ===\n"
