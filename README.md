# rubinot-data

Planning and architecture repository for a new Rubinot data platform, using [TibiaData](https://tibiadata.com/) as the primary reference model.

## Purpose

This repository contains **planning only** (no implementation yet) for migrating and consolidating learnings from:

- `techturbid/rubinot-live`
- `techturbid/rubinot-api`

into a new long-term project direction: **rubinot-data**.

## Why this repo exists

Current Rubinot efforts already solved many hard problems (scraping, queueing, cache, anti-bot handling), but the architecture is split and has operational pain points. TibiaData has years of production maturity and a cleaner product shape (stable versions, assets repo, API/docs split, deploy artifacts).

This repo is where we define:

1. Current-state context
2. Gap analysis vs TibiaData
3. Target architecture
4. Migration phases
5. Risk register and contingencies
6. Open decisions and governance model

## Repository structure

- `planning/context-current-state.md`
- `planning/tibiadata-reference-analysis.md`
- `planning/target-architecture.md`
- `planning/migration-plan.md`
- `planning/roadmap-phases.md`
- `planning/risk-register.md`
- `planning/open-questions.md`

## Scope status

- ✅ Repo created and initialized
- ✅ Comprehensive planning docs added
- ⏳ Implementation intentionally deferred

## Notes

- Account owner: `giovannirco`
- Visibility: **private**
- No code changes were made in `rubinot-live` or `rubinot-api` as part of this step.
