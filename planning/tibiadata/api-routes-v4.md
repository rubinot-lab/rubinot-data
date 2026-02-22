# TibiaData API v4 Routes (Comprehensive)

Base URL (public):
- `https://api.tibiadata.com`

Health/system:
- `GET /`
- `GET /ping`
- `GET /health`
- `GET /healthz`
- `GET /readyz`
- `GET /versions`

## Data routes and upstream source mapping

- `GET /v4/boostablebosses`
  - Upstream: `https://www.tibia.com/library/?subtopic=boostablebosses`

- `GET /v4/character/:name`
  - Upstream: `https://www.tibia.com/community/?subtopic=characters&name={name}`

- `GET /v4/creatures`
  - Upstream: `https://www.tibia.com/library/?subtopic=creatures`

- `GET /v4/creature/:race`
  - Upstream: `https://www.tibia.com/library/?subtopic=creatures&race={race}`

- `GET /v4/fansites`
  - Upstream: `https://www.tibia.com/community/?subtopic=fansites`

- `GET /v4/guild/:name`
  - Upstream: `https://www.tibia.com/community/?subtopic=guilds&page=view&GuildName={name}`

- `GET /v4/guilds/:world`
  - Upstream: `https://www.tibia.com/community/?subtopic=guilds&world={world}`

- `GET /v4/highscores/:world/:category/:vocation`
- `GET /v4/highscores/:world/:category/:vocation/:page`
  - Upstream: `https://www.tibia.com/community/?subtopic=highscores&world={world}&category={categoryId}&profession={vocationId}&currentpage={page}`

- `GET /v4/house/:world/:house_id`
  - Upstream: `https://www.tibia.com/community/?subtopic=houses&page=view&world={world}&houseid={house_id}`

- `GET /v4/houses/:world/:town`
  - Upstream flow: world/town-specific houses listing from Tibia houses pages

- `GET /v4/killstatistics/:world`
  - Upstream: `https://www.tibia.com/community/?subtopic=killstatistics&world={world}`

- `GET /v4/news/archive`
- `GET /v4/news/archive/:days`
- `GET /v4/news/latest`
- `GET /v4/news/newsticker`
  - Upstream: `https://www.tibia.com/news/?subtopic=newsarchive` (POST form filters by date/type)

- `GET /v4/news/id/:news_id`
  - Upstream: `https://www.tibia.com/news/?subtopic=newsarchive&id={news_id}`

- `GET /v4/spells`
  - Upstream: `https://www.tibia.com/library/?subtopic=spells&vocation={vocation}`

- `GET /v4/spell/:spell_id`
  - Upstream: `https://www.tibia.com/library/?subtopic=spells&spell={spell_id}`

- `GET /v4/worlds`
  - Upstream: `https://www.tibia.com/community/?subtopic=worlds`

- `GET /v4/world/:name`
  - Upstream: `https://www.tibia.com/community/?subtopic=worlds&world={name}`

## Deprecated compatibility routes

`/v3/*` paths still exist in docs/source for compatibility but are deprecated.

## Notes on behavior

- Input validation happens before network fetch.
- Highscores has restriction behavior in restricted mode.
- Error responses include API metadata + status object (HTTP code + API error code/message).
- All routes are still effectively scraping-backed despite stable JSON contract.
