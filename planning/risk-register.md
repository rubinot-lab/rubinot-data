# Risk Register

| ID | Risk | Impact | Likelihood | Early Signal | Mitigation | Contingency |
|---|---|---|---|---|---|---|
| R1 | Cloudflare challenge rate spikes | High | High | sudden increase in 403/challenge markers | adaptive pacing + UA/profile rotation + challenge telemetry | force restricted mode for heavy endpoints |
| R2 | Upstream HTML structure changes | High | High | parser error bursts after deploy/upstream updates | fixture snapshots + parser contract tests | stale-cache fallback + hotfix parser pipeline |
| R3 | Queue backlog saturation | High | Medium | queue latency and pending jobs rising | queue partitioning + autoscaling + retry budget | drop non-critical jobs first |
| R4 | Contract drift between versions | High | Medium | CI schema diff failures | contract-first OpenAPI + compatibility tests | freeze deploy and release patch mapping |
| R5 | Cost growth from browser workloads | Medium | Medium | infra cost and CPU/memory spikes | scrape cadence tuning + endpoint restrictions | reduce frequency and enable maintenance windows |
| R6 | Consumer breakage during migration | High | Medium | support reports and 4xx increase | compatibility aliases + migration docs + canary rollout | rollback routing at gateway |
| R7 | Data freshness regressions | Medium | Medium | stale-age metrics exceeding thresholds | freshness SLOs per entity + priority queues | temporary stale TTL extension with warning headers |
| R8 | Legal/policy concerns with scraping | High | Low/Med | complaints or access restrictions | respectful rate limits + clear user-agent policy + legal review | pause affected collectors while preserving API contract |

## Risk governance cadence

- Weekly risk review during migration
- Severity re-scoring after each milestone
- Post-incident update required for any Sev-1/Sev-2 event
