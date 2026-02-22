# Current Rubinot-Live Endpoints (Actual URL shape)

Primary public base:
- `https://live.rubinot.dev/api/v1`

> Note: host routing can change, but this reflects the current documented shape from `rubinot-live`.

## Health/metrics

- `GET https://live.rubinot.dev/api/v1/ping`
- `GET https://live.rubinot.dev/api/v1/health`
- `GET https://live.rubinot.dev/api/v1/ready`
- `GET https://live.rubinot.dev/api/v1/health/live`
- `GET https://live.rubinot.dev/api/v1/health/ready`
- `GET https://live.rubinot.dev/api/v1/metrics`
- `GET https://live.rubinot.dev/api/v1/metrics/collectors`
- `GET https://live.rubinot.dev/api/v1/metrics/collectors/jobs`

## Core game data

- `GET https://live.rubinot.dev/api/v1/worlds`
- `GET https://live.rubinot.dev/api/v1/worlds/:world`
- `GET https://live.rubinot.dev/api/v1/characters/:name`
- `GET https://live.rubinot.dev/api/v1/guilds?world=:world`
- `GET https://live.rubinot.dev/api/v1/guilds/:name`
- `GET https://live.rubinot.dev/api/v1/highscores?world=:world&category=:category&page=:page&vocation=:vocation`
- `GET https://live.rubinot.dev/api/v1/highscores/categories`
- `GET https://live.rubinot.dev/api/v1/highscores/:world/:category?page=:page`
- `GET https://live.rubinot.dev/api/v1/deaths?world=:world&guild=:guild&minLevel=:minLevel&pvpOnly=:pvpOnly`
- `GET https://live.rubinot.dev/api/v1/killstats?world=:world`
- `GET https://live.rubinot.dev/api/v1/transfers`
- `GET https://live.rubinot.dev/api/v1/houses?world=:world&town=:town`
- `GET https://live.rubinot.dev/api/v1/houses/:world/:town`
- `GET https://live.rubinot.dev/api/v1/houses/towns`
- `GET https://live.rubinot.dev/api/v1/bans?world=:world`
- `GET https://live.rubinot.dev/api/v1/news`
- `GET https://live.rubinot.dev/api/v1/events/schedule`
- `GET https://live.rubinot.dev/api/v1/auctions/current?page=:page`
- `GET https://live.rubinot.dev/api/v1/auctions/history?page=:page`
- `GET https://live.rubinot.dev/api/v1/auctions/:id`

## Account/subscription

- `POST https://live.rubinot.dev/api/v1/guilds/subscribe`
- `DELETE https://live.rubinot.dev/api/v1/guilds/subscribe`
- `GET https://live.rubinot.dev/api/v1/guilds/subscriptions`
- `GET https://live.rubinot.dev/api/v1/account/usage`
- `GET https://live.rubinot.dev/api/v1/account/quota`

## Current behavior notes

- Largely scraping-backed, with cache and background jobs.
- Endpoint richness is strong, but stability can vary under Cloudflare/challenge pressure.
- Some routes expose richer product features than TibiaData baseline (events/auctions/subscriptions/account usage).
