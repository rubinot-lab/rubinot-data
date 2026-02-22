# Open Questions

## Product and scope

1. Should `rubinot-data` be public from day 1 or stay private until parity is proven?
2. Which current endpoints are truly required by real consumers vs nice-to-have?
3. What deprecation window is acceptable for old paths?

## Technical

4. Keep Node.js stack initially or plan eventual Go migration (TibiaData-like runtime path)?
5. Do we standardize on Playwright or Puppeteer for all collectors?
6. How much static data should be moved to assets pipeline early?
7. Should heavy endpoints (highscores full pages, all-world aggregations) be restricted by tier?

## Operations

8. What are realistic SLOs for live-scraped endpoints under anti-bot pressure?
9. Which alert thresholds should page immediately vs create tickets?
10. How aggressive should autoscaling be for worker pools?

## Governance

11. ADR template and approval flow (who signs off architecture changes)?
12. Release policy for API versions (semantic versioning + sunset policy)?
13. Security policy for API keys and abuse detection before public rollout?

## Immediate next actions (no code)

- Validate this plan against actual endpoint usage from gateway logs
- Confirm preferred migration order by endpoint family
- Approve initial SLOs and restricted-mode rules
- Decide private beta timeline
