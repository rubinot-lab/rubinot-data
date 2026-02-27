#!/usr/bin/env python3
import json
import sys
import time
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:18080"
WORLD = sys.argv[2] if len(sys.argv) > 2 else "Belaria"
CONCURRENCY = int(sys.argv[3]) if len(sys.argv) > 3 else 10

print("=== Online Characters Benchmark ===")
print(f"API: {BASE_URL}")
print(f"World: {WORLD}")
print(f"Concurrency: {CONCURRENCY}")
print()

total_start = time.monotonic()

print("[1/2] Fetching world online list...")
t = time.monotonic()
resp = urllib.request.urlopen(f"{BASE_URL}/v1/world/{urllib.parse.quote(WORLD)}")
world_data = json.loads(resp.read())
elapsed = int((time.monotonic() - t) * 1000)
print(f"  Done in {elapsed}ms")

players = world_data["world"]["players_online"]
count = len(players)
print(f"  Players online: {count}")
print()


def fetch_character(name):
    encoded = urllib.parse.quote(name, safe="")
    url = f"{BASE_URL}/v1/character/{encoded}"
    try:
        req = urllib.request.urlopen(url, timeout=30)
        return name, req.status
    except urllib.error.HTTPError as e:
        return name, e.code
    except Exception:
        return name, 0


print(f"[2/2] Fetching each character ({count} requests, concurrency={CONCURRENCY})...")
t = time.monotonic()

results = {}
with ThreadPoolExecutor(max_workers=CONCURRENCY) as pool:
    futures = {pool.submit(fetch_character, p["name"]): p["name"] for p in players}
    for future in as_completed(futures):
        name, code = future.result()
        results[name] = code

elapsed = int((time.monotonic() - t) * 1000)

ok = sum(1 for c in results.values() if c == 200)
fail = count - ok

print(f"  Done in {elapsed}ms")
print()
print("=== Results ===")
print(f"  Total characters: {count}")
print(f"  Successful (200): {ok}")
print(f"  Failed:           {fail}")
print(f"  Wall time:        {elapsed}ms")
if count > 0:
    print(f"  Avg per char:     {elapsed // count}ms")
    print(f"  Throughput:       {count / (elapsed / 1000):.1f} req/s")

total_elapsed = int((time.monotonic() - total_start) * 1000)
print(f"  Total wall time:  {total_elapsed}ms")

if fail > 0:
    print()
    print("  Non-200 codes:")
    from collections import Counter
    codes = Counter(c for c in results.values() if c != 200)
    for code, n in codes.most_common():
        print(f"    {code}: {n}")
