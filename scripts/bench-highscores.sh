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

print("=== Highscores Benchmark ===")
print(f"API: {BASE_URL}")
print(f"World: {WORLD}")
print(f"Concurrency: {CONCURRENCY}")
print()


def fetch_json(url):
    try:
        resp = urllib.request.urlopen(url, timeout=30)
        return json.loads(resp.read()), resp.status
    except urllib.error.HTTPError as e:
        return None, e.code
    except Exception:
        return None, 0


def fetch_status(url):
    try:
        resp = urllib.request.urlopen(url, timeout=30)
        resp.read()
        return url, resp.status
    except urllib.error.HTTPError as e:
        return url, e.code
    except Exception:
        return url, 0


print("[1/3] Fetching categories...")
data, _ = fetch_json(f"{BASE_URL}/v1/highscores/categories")
slugs = [c["slug"] for c in data["categories"]]
print(f"  Categories: {len(slugs)}")
print()

print("[2/3] Discovering total pages per category...")
discovery_urls = {
    slug: f"{BASE_URL}/v1/highscores/{urllib.parse.quote(WORLD)}/{slug}/all/1"
    for slug in slugs
}

pages_per_cat = {}
with ThreadPoolExecutor(max_workers=CONCURRENCY) as pool:
    futures = {}
    for slug, url in discovery_urls.items():
        futures[pool.submit(fetch_json, url)] = slug
    for future in as_completed(futures):
        slug = futures[future]
        result, code = future.result()
        if result and code == 200:
            pages_per_cat[slug] = (
                result.get("highscores", {})
                .get("highscore_page", {})
                .get("total_pages", 0)
            )
        else:
            pages_per_cat[slug] = 0

print("  Pages per category:")
for slug in slugs:
    pages = pages_per_cat.get(slug, 0)
    print(f"    {slug:<25} {pages} pages")

fetch_urls = []
for slug in slugs:
    pages = pages_per_cat.get(slug, 0)
    for page in range(1, pages + 1):
        fetch_urls.append(
            f"{BASE_URL}/v1/highscores/{urllib.parse.quote(WORLD)}/{slug}/all/{page}"
        )

total_requests = len(fetch_urls)
print(f"\n  Total requests: {total_requests}")
print()

print(f"[3/3] Fetching all {total_requests} pages (concurrency={CONCURRENCY})...")
t = time.monotonic()

results = {}
with ThreadPoolExecutor(max_workers=CONCURRENCY) as pool:
    futures = {pool.submit(fetch_status, url): url for url in fetch_urls}
    for future in as_completed(futures):
        url, code = future.result()
        results[url] = code

elapsed = int((time.monotonic() - t) * 1000)

ok = sum(1 for c in results.values() if c == 200)
fail = total_requests - ok

print(f"  Done in {elapsed}ms")
print()
print("=== Results ===")
print(f"  Categories:       {len(slugs)}")
print(f"  Total pages:      {total_requests}")
print(f"  Successful (200): {ok}")
print(f"  Failed:           {fail}")
print(f"  Wall time:        {elapsed}ms")
if total_requests > 0:
    print(f"  Avg per page:     {elapsed // total_requests}ms")
    print(f"  Throughput:       {total_requests / (elapsed / 1000):.1f} req/s")

if fail > 0:
    print()
    print("  Non-200 codes:")
    from collections import Counter
    codes = Counter(c for c in results.values() if c != 200)
    for code, n in codes.most_common():
        print(f"    {code}: {n}")
