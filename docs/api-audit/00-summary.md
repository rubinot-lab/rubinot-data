# API Audit Summary - api.rubinot.dev

**Date:** 2026-02-24 (post-fix deployment)
**Version:** v1.3.1 (commit: ebe7b00)
**Base URL:** https://api.rubinot.dev

## Audit Scope

All 25+ endpoints tested against the live API. Every response captured and analyzed for:
- Correct HTTP status codes and envelope structure
- Data completeness (missing fields, empty objects)
- Data quality (wrong types, raw strings, header rows as data)
- Consistency across endpoints (naming, date formats, null vs omitted)
- Response times

## Response Time Summary

| Category | Time | Endpoints |
|----------|------|-----------|
| Instant | <0.2s | /ping, /healthz, /versions, highscores redirect (302) |
| Fast | 2-4s | worlds, world detail, character, guilds list, guild detail, highscores, killstatistics, news (all), events, auctions current/history, transfers |
| Slow | 4-12s | houses list (~4s), deaths (~12s), banishments (~12s), auction detail (~4s) |
| Very slow | 30-43s | house detail 404s (FlareSolverr timeout on invalid IDs) |

## Endpoint Status Overview

| Endpoint | Status | Response Time | Issues |
|----------|--------|---------------|--------|
| `GET /` | OK | - | - |
| `GET /ping` | OK | instant | - |
| `GET /healthz` | OK | instant | - |
| `GET /readyz` | OK | instant | not tested (k8s probe) |
| `GET /versions` | OK | instant | - |
| `GET /v1/worlds` | OK | 3.1s | - |
| `GET /v1/world/:name` | OK | 2.4s | `status`/`pvp_type` nested under `info` sub-object |
| `GET /v1/character/:name` | ISSUES | 2.2s | empty `account_information`, some fields missing |
| `GET /v1/guild/:name` | OK | 2.2s | some members missing `rank` field |
| `GET /v1/guilds/:world` | **FIXED** | 2.4s | 212 active guilds returned (was empty) |
| `GET /v1/houses/:world/:town` | OK | 4.3s | - |
| `GET /v1/house/:world/:id` | OK | 2.3s | `paid_until` now ISO 8601; 404s take 30-40s |
| `GET /v1/highscores/:world/...` | OK | 2.6s | - |
| `GET /v1/highscores/:world` | OK | 0.14s | 302 redirect works |
| `GET /v1/killstatistics/:world` | OK | 2.8s | - |
| `GET /v1/news/id/:id` | OK | 2.5s | - |
| `GET /v1/news/latest` | ISSUES | 2.3s | still missing `id` and `url` fields |
| `GET /v1/news/newsticker` | ISSUES | 2.6s | has `id` but missing `url` field |
| `GET /v1/news/archive` | OK | 2.1s | has both `id` and `url` |
| `GET /v1/events/schedule` | OK | 2.4s | no duplicate days (16 entries for Feb) |
| `GET /v1/deaths/:world` | ISSUES | 11.7s | **filters not applied** (pvp/level ignored) |
| `GET /v1/banishments/:world` | OK | 12.4s | 0 bans (may be correct for Belaria) |
| `GET /v1/transfers` | OK | 2.1s | - |
| `GET /v1/auctions/current/:page` | OK | 3.2s | `auction_id` now int |
| `GET /v1/auctions/history/:page` | **FIXED** | 2.2s | status now "ended" (was "active") |
| `GET /v1/auctions/:id` | OK | 4.2s | - |

## Issues Fixed Since Last Audit

| Bug | Status |
|-----|--------|
| BUG-02: Guilds list returns empty arrays | **FIXED** — data URI form submission for POST |
| BUG-03: Auction history status always "active" | **FIXED** — status now "ended" |
| BUG-05: House `paid_until` raw string | **FIXED** — now ISO 8601 |
| BUG-09: Events duplicate day numbers | **FIXED** — no duplicates observed |
| BUG-10: `auction_id` is string | **FIXED** — now integer |
| BUG-14: Commit SHA always "unknown" | **FIXED** — now shows `ebe7b00` |
| BUG-15: World status casing inconsistency | **FIXED** — normalized |

## Remaining Issues

| Severity | Bug | Description |
|----------|-----|-------------|
| **Critical** | NEW | Deaths filters (`pvp`, `level`) not applied — returns unfiltered results |
| High | BUG-04 (partial) | Character `account_information` always `{}` |
| High | BUG-08 (partial) | News `/latest` still missing `id` and `url` fields |
| Medium | NEW | Guild members sometimes missing `rank` field |
| Medium | BUG-13 | News newsticker missing `url` field |
| Low | NEW | Guild list `logo_url` uses relative paths (`./images/...`) |

## Documents

- [01-endpoint-responses.md](./01-endpoint-responses.md) - Full response analysis per endpoint
- [02-bugs-and-fixes.md](./02-bugs-and-fixes.md) - Detailed bug list with root causes and fix recommendations
- [03-patterns-and-consistency.md](./03-patterns-and-consistency.md) - Cross-cutting patterns, naming, and consistency issues
