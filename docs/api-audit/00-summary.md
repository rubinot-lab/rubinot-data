# API Audit Summary - api.rubinot.dev

**Date:** 2026-02-24
**Version:** v0.2.0 (commit: unknown)
**Base URL:** https://api.rubinot.dev

## Audit Scope

All 25+ endpoints tested against the live API. Every response captured and analyzed for:
- Correct HTTP status codes and envelope structure
- Data completeness (missing fields, empty objects)
- Data quality (wrong types, raw strings, header rows as data)
- Consistency across endpoints (naming, date formats, null vs omitted)

## Endpoint Status Overview

| Endpoint | Status | Issues |
|----------|--------|--------|
| `GET /` | OK | - |
| `GET /ping` | OK | - |
| `GET /healthz` | OK | - |
| `GET /readyz` | OK | - |
| `GET /versions` | OK | commit always "unknown" |
| `GET /v1/worlds` | OK | - |
| `GET /v1/world/:name` | OK | - |
| `GET /v1/character/:name` | ISSUES | many fields missing, empty account_information |
| `GET /v1/guild/:name` | ISSUES | header row as first member, is_online null for offline, missing guild metadata |
| `GET /v1/guilds/:world` | BROKEN | returns empty arrays for active guilds |
| `GET /v1/houses/:world/:town` | OK | - |
| `GET /v1/house/:world/:id` | ISSUES | paid_until raw string, missing owner level/vocation/moving_date |
| `GET /v1/highscores/:world/...` | OK | missing title/traded/auction_url fields |
| `GET /v1/killstatistics/:world` | OK | - |
| `GET /v1/news/id/:id` | OK | - |
| `GET /v1/news/latest` | ISSUES | missing id/url, only 1 old entry |
| `GET /v1/news/newsticker` | ISSUES | missing url field |
| `GET /v1/news/archive` | OK | - |
| `GET /v1/events/schedule` | ISSUES | duplicate day numbers, days from other months mixed in |
| `GET /v1/deaths/:world` | OK | missing assists field |
| `GET /v1/banishments/:world` | OK | 0 bans (may be correct) |
| `GET /v1/transfers` | OK | - |
| `GET /v1/auctions/current/:page` | ISSUES | auction_id is string not int |
| `GET /v1/auctions/history/:page` | ISSUES | auction_id is string, history entries show status "active" |
| `GET /v1/auctions/:id` | ISSUES | auction_id is string, tried current before past |

## Issue Severity

- **Critical (3):** guild members header row, guilds list empty, auction history status wrong
- **High (6):** character missing fields, house paid_until format, guild is_online null, news/latest incomplete, events duplicate days, auction_id type
- **Medium (4):** highscores missing fields, news missing url/id, deaths missing assists, commit "unknown"
- **Low (1):** banishments empty (may be legitimate)

## Documents

- [01-endpoint-responses.md](./01-endpoint-responses.md) - Full response analysis per endpoint
- [02-bugs-and-fixes.md](./02-bugs-and-fixes.md) - Detailed bug list with root causes and fix recommendations
- [03-patterns-and-consistency.md](./03-patterns-and-consistency.md) - Cross-cutting patterns, naming, and consistency issues
