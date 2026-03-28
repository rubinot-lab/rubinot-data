# Highscores V2 Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix rubinot-data's broken v2 cross-world highscores endpoint using batch CDP fetching, then switch rubinot-api's processor to consume v2 for a ~6x fetch speedup.

**Architecture:** rubinot-data adds `V2FetchHighscoresBatch` (same pattern as `V2FetchKillstatisticsBatch`) and rewires the `world=all` handler. rubinot-api adds a thin v2 client method and feature-flagged adapter in the processor.

**Tech Stack:** Go 1.23 (rubinot-data), TypeScript/BullMQ (rubinot-api), CDP batch fetch, Prometheus metrics

---

## Repo 1: rubinot-data (Go)

### Task 1: Add `V2FetchHighscoresBatch` function

**Files:**
- Modify: `internal/scraper/v2_fetch.go` (add function after `V2FetchKillstatisticsBatch` ~line 557)
- Test: `internal/scraper/v2_fetch_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/scraper/v2_fetch_test.go`:

```go
func TestV2FetchHighscoresBatch(t *testing.T) {
	makePayload := func(worldName string) highscoresAPIResponse {
		return highscoresAPIResponse{
			Players: []struct {
				Rank      int         `json:"rank"`
				ID        int         `json:"id"`
				Name      string      `json:"name"`
				Level     int         `json:"level"`
				Vocation  int         `json:"vocation"`
				WorldID   int         `json:"world_id"`
				WorldName string      `json:"worldName"`
				Value     interface{} `json:"value"`
			}{
				{Rank: 1, ID: 1, Name: "Player1", Level: 500, Vocation: 4, WorldID: 1, WorldName: worldName, Value: 999},
				{Rank: 2, ID: 2, Name: "Player2", Level: 400, Vocation: 5, WorldID: 1, WorldName: worldName, Value: 888},
			},
			TotalCount: 2,
		}
	}

	body1, _ := json.Marshal(makePayload("Elysian"))
	body2, _ := json.Marshal(makePayload("Belaria"))

	q1 := url.Values{}
	q1.Set("world", "1")
	q1.Set("category", "experience")
	q1.Set("vocation", "5")

	q2 := url.Values{}
	q2.Set("world", "15")
	q2.Set("category", "experience")
	q2.Set("vocation", "5")

	oc := newTestOC(t, map[string]string{
		"/api/highscores?" + q1.Encode(): string(body1),
		"/api/highscores?" + q2.Encode(): string(body2),
	})

	worlds := []validation.World{
		{ID: 1, Name: "Elysian"},
		{ID: 15, Name: "Belaria"},
	}
	category := validation.HighscoreCategory{ID: 1, Name: "Experience", Slug: "experience"}
	vocation := validation.HighscoreVocation{Name: "Knights", ProfessionID: 5}

	results, sources, err := V2FetchHighscoresBatch(context.Background(), oc, "http://test.local", worlds, category, vocation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if results[0].World != "Elysian" {
		t.Errorf("expected first world Elysian, got %s", results[0].World)
	}
	if results[1].World != "Belaria" {
		t.Errorf("expected second world Belaria, got %s", results[1].World)
	}
	if len(results[0].HighscoreList) != 2 {
		t.Errorf("expected 2 entries for Elysian, got %d", len(results[0].HighscoreList))
	}
	if results[0].Category != "experience" {
		t.Errorf("expected category experience, got %s", results[0].Category)
	}
	if results[0].Vocation != "Knights" {
		t.Errorf("expected vocation Knights, got %s", results[0].Vocation)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scraper/ -run TestV2FetchHighscoresBatch -v -count=1`
Expected: FAIL — `V2FetchHighscoresBatch` undefined

- [ ] **Step 3: Implement `V2FetchHighscoresBatch`**

Add to `internal/scraper/v2_fetch.go` after `V2FetchKillstatisticsBatch`:

```go
func V2FetchHighscoresBatch(
	ctx context.Context,
	oc *OptimizedClient,
	baseURL string,
	worlds []validation.World,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
) ([]domain.HighscoresResult, []string, error) {
	base := strings.TrimRight(baseURL, "/")
	apiURLs := make([]string, 0, len(worlds))
	for _, world := range worlds {
		query := url.Values{}
		query.Set("world", strconv.Itoa(world.ID))
		query.Set("category", category.Slug)
		query.Set("vocation", fmt.Sprintf("%d", vocation.ProfessionID))
		apiURLs = append(apiURLs, fmt.Sprintf("%s/api/highscores?%s", base, query.Encode()))
	}

	bodies, err := oc.BatchFetchJSON(ctx, apiURLs)
	if err != nil {
		return nil, apiURLs, err
	}

	results := make([]domain.HighscoresResult, 0, len(worlds))
	for i, apiURL := range apiURLs {
		body, ok := bodies[apiURL]
		if !ok {
			return nil, apiURLs, fmt.Errorf("missing response for %s", apiURL)
		}
		var payload highscoresAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			return nil, apiURLs, parseErr
		}

		worldName := worlds[i].Name
		items := make([]domain.Highscore, 0, len(payload.Players))
		for _, row := range payload.Players {
			items = append(items, domain.Highscore{
				Rank:       row.Rank,
				ID:         row.ID,
				Name:       strings.TrimSpace(row.Name),
				Vocation:   fallbackString(vocationNameByID(row.Vocation), "Unknown"),
				VocationID: row.Vocation,
				World:      resolveHighscoreWorldName(row.WorldName, row.WorldID, worldName),
				WorldID:    row.WorldID,
				Level:      row.Level,
				Value:      fmt.Sprintf("%v", row.Value),
			})
		}

		totalRecords := payload.TotalCount
		if totalRecords <= 0 {
			totalRecords = len(items)
		}

		results = append(results, domain.HighscoresResult{
			World:         worldName,
			Category:      category.Slug,
			Vocation:      vocation.Name,
			CachedAt:      payload.CachedAt,
			HighscoreList: items,
			HighscorePage: domain.HighscorePage{
				CurrentPage:  1,
				TotalPages:   1,
				TotalRecords: totalRecords,
			},
			AvailableSeasons: payload.AvailableSeasons,
		})
	}

	return results, apiURLs, nil
}
```

- [ ] **Step 4: Check `resolveHighscoreWorldName` exists — if not, inline the fallback**

Run: `grep -rn "func resolveHighscoreWorldName" internal/scraper/`

If not found, the function is inlined in `V2FetchHighscores`. In that case, replace the `resolveHighscoreWorldName` call in the implementation above with:

```go
World: fallbackString(strings.TrimSpace(row.WorldName), fallbackString(worldNameByID(row.WorldID), worldName)),
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/scraper/ -run TestV2FetchHighscoresBatch -v -count=1`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `make test`
Expected: All tests pass, no regressions

- [ ] **Step 7: Commit**

```bash
git add internal/scraper/v2_fetch.go internal/scraper/v2_fetch_test.go
git commit -m "feat(scraper): add V2FetchHighscoresBatch for parallel cross-world fetch"
```

---

### Task 2: Fix v2 highscores handler to use batch fetch

**Files:**
- Modify: `internal/api/handlers_v2.go:145-161` (replace sequential loop)

- [ ] **Step 1: Replace sequential loop with batch call**

In `internal/api/handlers_v2.go`, replace lines 145-161 (the `isAllWorldsToken` block in `v2GetHighscores`):

**Before:**
```go
	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.HighscoresResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			highscores, sourceURL, err := scraper.V2FetchHighscores(c.Request.Context(), oc, resolvedBaseURL, world.Name, world.ID, category, vocation)
			if err != nil {
				return endpointResult{Sources: append(allSources, sourceURL)}, err
			}
			results = append(results, highscores)
			allSources = append(allSources, sourceURL)
		}
		return endpointResult{
			PayloadKey: "highscores",
			Payload:    results,
			Sources:    allSources,
		}, nil
	}
```

**After:**
```go
	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results, sources, err := scraper.V2FetchHighscoresBatch(c.Request.Context(), oc, resolvedBaseURL, worlds, category, vocation)
		if err != nil {
			return endpointResult{Sources: sources}, err
		}
		return endpointResult{
			PayloadKey: "highscores",
			Payload:    results,
			Sources:    sources,
		}, nil
	}
```

- [ ] **Step 2: Build and lint**

Run: `make build && make lint`
Expected: Clean build, no lint errors

- [ ] **Step 3: Run full test suite**

Run: `make test`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/api/handlers_v2.go
git commit -m "fix(api): use batch fetch for v2 cross-world highscores"
```

---

### Task 3: Deploy rubinot-data and verify

**Files:** None (deployment task)

- [ ] **Step 1: Push to main**

```bash
git push origin main
```

- [ ] **Step 2: Tag and deploy**

```bash
git tag v2.3.8
git push origin v2.3.8
```

- [ ] **Step 3: Wait for pipeline**

```bash
gh run list --repo rubinot-lab/rubinot-data --limit 3
```

Wait for build-and-deploy to succeed.

- [ ] **Step 4: Verify v2 cross-world highscores works**

```bash
curl -s --max-time 30 "https://data.rubinot.dev/v2/highscores/all/experience/knights" | python3 -c "
import sys, json
d = json.load(sys.stdin)
hs = d.get('highscores', [])
print('status:', d['information']['status'])
print('worlds:', len(hs))
for w in hs[:3]:
    print(f'  {w[\"world\"]}: {len(w[\"highscore_list\"])} entries')
if len(hs) > 3:
    print(f'  ... + {len(hs)-3} more')
"
```

Expected: 14 worlds, ~1000 entries each, HTTP 200.

- [ ] **Step 5: Verify single-world still works**

```bash
curl -s --max-time 15 "https://data.rubinot.dev/v2/highscores/Elysian/experience/all" | python3 -c "
import sys, json
d = json.load(sys.stdin)
hs = d.get('highscores', {})
print('world:', hs.get('world'), 'entries:', len(hs.get('highscore_list', [])))
"
```

Expected: world=Elysian, ~1000 entries.

---

## Repo 2: rubinot-api (TypeScript)

> **Note:** Tasks 4-6 are in the `rubinot-lab/rubinot-api` repo. Clone/switch to that repo before proceeding.

### Task 4: Add v2 client method and feature flag

**Files:**
- Modify: `src/services/rubinot-data-client.ts` (add method after `getHighscoresCrossWorldByVocation`)
- Modify: `src/config/env.ts` (add `HIGHSCORE_USE_V2` flag)

- [ ] **Step 1: Add `HIGHSCORE_USE_V2` env var**

In `src/config/env.ts`, add to the env schema (near other `HIGHSCORE_*` vars):

```typescript
HIGHSCORE_USE_V2: z.coerce.boolean().default(false),
```

- [ ] **Step 2: Add `getHighscoresCrossWorldV2` method**

In `src/services/rubinot-data-client.ts`, add after `getHighscoresCrossWorldByVocation`:

```typescript
async getHighscoresCrossWorldV2(
  category: string,
  vocation: HighscoreVocation,
): Promise<HighscoresPayload[]> {
  return this.fetchPayload<HighscoresPayload[]>(
    `/v2/highscores/all/${encodeURIComponent(category)}/${encodeURIComponent(vocation)}`,
    {
      endpoint: "highscores_cross_world_v2",
      timeoutMs: 30000,
      attempts: 3,
      retryDelayMs: 3000,
    },
  );
}
```

- [ ] **Step 3: Build to verify compilation**

Run: `npm run build` (or `yarn build`)
Expected: Clean build

- [ ] **Step 4: Commit**

```bash
git add src/services/rubinot-data-client.ts src/config/env.ts
git commit -m "feat(client): add v2 cross-world highscores method and feature flag"
```

---

### Task 5: Switch processor to use v2 behind feature flag

**Files:**
- Modify: `src/jobs/processors/highscores.processor.ts` (~line 230)

- [ ] **Step 1: Add v2 fetch path in processor**

In `src/jobs/processors/highscores.processor.ts`, find the vocation fetch block (~line 230):

**Before:**
```typescript
if (env.HIGHSCORE_VOCATIONS_ENABLED) {
  for (const vocation of HIGHSCORE_VOCATIONS) {
    const payload = await client.getHighscoresCrossWorldByVocation(category, vocation);
    for (const worldData of payload.worlds) {
```

**After:**
```typescript
if (env.HIGHSCORE_VOCATIONS_ENABLED) {
  for (const vocation of HIGHSCORE_VOCATIONS) {
    const worldResults = env.HIGHSCORE_USE_V2
      ? await client.getHighscoresCrossWorldV2(category, vocation)
      : (await client.getHighscoresCrossWorldByVocation(category, vocation)).worlds;
    for (const worldData of worldResults) {
```

This is the only change. The `worldData` iteration body stays identical because both v1's `payload.worlds[i]` and v2's array elements have the same shape: `{ world, highscore_list, highscore_page }`.

- [ ] **Step 2: Build to verify compilation**

Run: `npm run build` (or `yarn build`)
Expected: Clean build

- [ ] **Step 3: Run existing tests**

Run: `npm test` (or `yarn test`)
Expected: All existing tests pass (flag defaults to false, v1 path unchanged)

- [ ] **Step 4: Commit**

```bash
git add src/jobs/processors/highscores.processor.ts
git commit -m "feat(highscores): switch processor to v2 cross-world fetch behind HIGHSCORE_USE_V2 flag"
```

---

### Task 6: Deploy rubinot-api and validate

**Files:** None (deployment task)

- [ ] **Step 1: Deploy with flag off**

Deploy rubinot-api with `HIGHSCORE_USE_V2=false`. Verify existing v1 path works normally — check `highscorePhaseSeconds` fetch metric for baseline.

- [ ] **Step 2: Enable v2 flag**

Set `HIGHSCORE_USE_V2=true` in the environment. Restart workers.

- [ ] **Step 3: Monitor first cycle**

Watch for:
- `highscorePhaseSeconds{phase="fetch"}` — expect ~2-3s (was ~12s)
- No errors in worker logs
- `highscoreSnapshotSize` — same world count as before
- Ranking output unchanged (compare `character_rank` row counts)

- [ ] **Step 4: Validate data correctness**

After one full cycle with v2:
- Compare highscore entry counts per world vs previous cycle
- Verify change detection still fires (check `highscore_change_events` in ClickHouse)
- Spot-check a few world/category combos: entry counts, top player names

- [ ] **Step 5: If stable for 24h, remove flag**

Remove the ternary in the processor, remove `HIGHSCORE_USE_V2` from env config, remove `getHighscoresCrossWorldByVocation` method. Commit:

```bash
git commit -m "refactor(highscores): remove v1 cross-world fetch path and HIGHSCORE_USE_V2 flag"
```
