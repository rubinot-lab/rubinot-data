# rubinot-data

Go proxy service that fetches and caches Tibia game data from upstream APIs (tibia.com, TibiaData). Serves as the data source for rubinot-api workers.

## Stack

- Go 1.23
- HTTP server (`cmd/server`)
- Makefile-based build

## Development

```bash
make build          # compile
make test           # go test ./... -v
make lint           # go vet
make run            # go run ./cmd/server
make docker-up      # docker compose up
```

## Architecture

- `cmd/server/` — HTTP server entrypoint
- `scripts/uc_patched_init.py` — CDP patch ConfigMap generator (used in CI)
- `scripts/bench-*.sh` — benchmark scripts

## Release & Deploy

Tag-driven. Push a tag to trigger build + deploy:

```bash
git checkout main && git pull --ff-only
git tag vX.Y.Z && git push origin vX.Y.Z
```

Pipeline: GitHub Actions → GHCR image (`ghcr.io/rubinot-lab/rubinot-data`) → GitOps update (`rubinot-lab/platform-gitops`, path `apps/rubinot/manifests/prod/`) → ArgoCD app `rubinot-lab-rubinot-prod` auto-sync.

See `DEPLOYMENT.md` for full details and troubleshooting.

### Git identity

All commits and pushes use the `unwashed-and-dazed` GitHub account:
```bash
gh auth switch --hostname github.com --user unwashed-and-dazed
```

### Verify deploy

```bash
gh run list --repo rubinot-lab/rubinot-data --limit 5
kubectl get applications -n argocd | grep rubinot-apps
```

## Conventions

- Semantic commits: `feat`, `fix`, `refactor`, `perf`, `chore`, `test`, `docs`
- No comments in code unless logic is non-obvious