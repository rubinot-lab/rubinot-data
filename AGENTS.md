# Agents — rubinot-data

## Deploy agent

Release flow (tag-driven):
```bash
gh auth switch --hostname github.com --user giovannirco
git checkout main && git pull --ff-only
git tag vX.Y.Z && git push origin vX.Y.Z
```

Post-deploy verification:
1. `gh run list --repo giovannirco/rubinot-data --limit 3` — confirm workflow success
2. `kubectl get applications -n argocd | grep rubinot-apps` — check ArgoCD sync
3. Check pod health in `rubinot` namespace via k8s MCP

## Investigation agent

rubinot-data is the upstream proxy for rubinot-api. When rubinot-api workers report fetch failures or timeouts, check rubinot-data:
1. Pod status and restart count in `rubinot` namespace
2. Pod logs for error rates, latency spikes, upstream tibia.com failures
3. Metrics endpoint for request volume and error rates

## Code change agent

Before making changes:
- `make build` — compile
- `make test` — run tests
- `make lint` — go vet
- Use semantic commits
- All commits via `giovannirco` GitHub account
