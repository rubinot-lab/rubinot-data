# Compatibility Mapping: Current Rubinot-Live -> Target Rubinot-Data

## Objective

Allow migration without abrupt consumer breakage by mapping existing routes to target contract paths.

## Suggested mappings

- `/api/v1/worlds` -> `/v1/worlds`
- `/api/v1/worlds/:world` -> `/v1/world/:name`
- `/api/v1/characters/:name` -> `/v1/character/:name`
- `/api/v1/guilds/:name` -> `/v1/guild/:name`
- `/api/v1/guilds?world=:world` -> `/v1/guilds/:world`
- `/api/v1/highscores/:world/:category?page=:page&vocation=:voc` -> `/v1/highscores/:world/:category/:vocation/:page`
- `/api/v1/houses/:world/:town` -> `/v1/houses/:world/:town`
- `/api/v1/killstats?world=:world` -> `/v1/killstatistics/:world`
- `/api/v1/news` -> `/v1/news/latest`
- `/api/v1/bans?world=:world` -> `/v1/banishments/:world`

Rubinot-specific retained routes:

- `/api/v1/deaths...` -> `/v1/deaths/:world` (+filters)
- `/api/v1/transfers` -> `/v1/transfers`
- `/api/v1/events/schedule` -> `/v1/events/schedule`
- `/api/v1/auctions/*` -> `/v1/auctions/*`

## Migration behavior recommendations

1. Keep old endpoints as compatibility aliases for one deprecation window.
2. Add response headers on old paths:
   - `Deprecation: true`
   - `Sunset: <date>`
   - `Link: <new-endpoint>; rel="successor-version"`
3. Track old-path usage and deprecate only after low steady usage.
