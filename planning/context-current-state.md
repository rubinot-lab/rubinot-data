# Current State Context (Rubinot)

## Snapshot

This summary is based on the existing repositories:

- `techturbid/rubinot-live` (microservices + Redis cache + worker/registry/metrics split)
- `techturbid/rubinot-api` (Fastify monolith + PostgreSQL + Bull jobs + Puppeteer scraping)

## What exists today

### rubinot-live

- Focus: live scraping API with distributed queue behavior and cache-first responses
- Runtime: Node.js + TypeScript + Fastify + Playwright
- Internal services: API, Worker, Registry, Metrics
- Infra patterns: Redis, Prometheus, Kubernetes deployment, API keys/rate limits
- Public shape: `/api/v1/*` endpoints for worlds, characters, guilds, highscores, deaths, etc.
- Known pain: Cloudflare challenges and selector fragility under website changes/anti-bot pressure

### rubinot-api

- Focus: broader API + background jobs + PostgreSQL persistence
- Runtime: Node.js + TypeScript + Fastify + Puppeteer
- Data: PostgreSQL (Prisma), Redis (Bull)
- Pattern: live `*Page` scraping endpoints + db-backed strategy for faster reads
- Known pain: complexity from mixed live/db models, operational overhead for scraper reliability

## Strengths to preserve

1. Large endpoint coverage and domain understanding
2. Existing parser/explorer know-how
3. Queue and scheduling experience
4. Caching discipline and TTL strategy
5. Existing ops knowledge (Docker/K8s/metrics)

## Current key challenges

1. **Anti-bot/Cloudflare resistance** impacts reliability
2. **Architecture split** creates cognitive load
3. **Contract drift risk** (endpoints and schema not fully version-governed)
4. **Fragile scraping adapters** against HTML changes
5. **Unclear long-term boundary** between "live scrape" and "stabilized API contract"

## Why this migration now

TibiaData demonstrates a mature OSS model that separates concerns clearly (API runtime, static assets, docs, helm/deploy), with versioned contracts and operational safeguards. Reusing that shape as reference reduces architectural guesswork.
