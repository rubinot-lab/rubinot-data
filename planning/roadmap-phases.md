# Roadmap Phases (Proposed)

## Quarter-oriented view

### Q1 — Foundation
- Complete endpoint inventory and usage analytics
- Contract-first `/v1` draft
- Canonical entity schema and provenance fields
- Basic architecture decision records (ADRs)

### Q2 — Reliability core
- Parser fixture suite and contract tests
- Challenge detection and resilience policies
- Queue isolation by endpoint criticality
- Observability baseline (dashboards + alerts)

### Q3 — Parallel production
- Shadow traffic and diffing pipeline
- Selected endpoint cutover
- Consumer migration guides
- Deprecation policy in effect

### Q4 — Consolidation
- Full migration for stable endpoints
- Legacy path retirement (where safe)
- Performance optimization and cost tuning
- Publish long-term maintenance guide

## Milestone gates

- M1: Contract baseline approved
- M2: Parser reliability >= target
- M3: Shadow parity >= target
- M4: Production cutover complete
- M5: Legacy deprecation complete

## Suggested SLO starter targets

- API availability: 99.5%
- p95 latency for cached reads: < 300ms
- p95 latency for live-triggered reads: < 3s (where applicable)
- Stable endpoint schema compliance: >= 99%
- Scrape success ratio (critical jobs): >= 97%
