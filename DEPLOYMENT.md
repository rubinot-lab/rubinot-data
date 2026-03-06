# DEPLOYMENT.md — rubinot-data

## Goal
Automated tag-driven deploys with GitOps + ArgoCD.

## Deployment architecture
- **Build/CI:** GitHub Actions (`.github/workflows/build-and-deploy-rubinot-data.yml`)
- **Registry:** GHCR (`ghcr.io/giovannirco/rubinot-data`)
- **GitOps repo:** `cddlabs-casa/omni-cddlabs-casa`
- **K8s rollout:** Argo app `rubinot-apps` (path contains rubinot-data manifests)

## Trigger model
- Triggers on any pushed tag and `workflow_dispatch`.

## Workflow behavior
On tag push:
1. Builds + pushes image tags (`short_sha`, `tag`, `latest`).
2. Regenerates CDP patch ConfigMap manifest from `scripts/uc_patched_init.py`.
3. Updates `manifests/cddlabs/apps/rubinot/rubinot-data.yaml` image + `APP_VERSION` + `APP_COMMIT`.
4. Commits and pushes to GitOps repo.
5. ArgoCD auto-sync applies rollout.

## Required secrets
- `OMNI_GITOPS_TOKEN` (required)
- `GHCR_TOKEN` (recommended)
- `GHCR_USERNAME` (optional)

Fallback to `GITHUB_TOKEN` exists, but PAT is more reliable for package push permissions.

## Release usage
```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

## Troubleshooting
### GHCR 403 on push
- Validate `GHCR_TOKEN` exists and has `write:packages`.
- Re-run workflow.

### GitOps commit not created
- Check `OMNI_GITOPS_TOKEN` permission to `cddlabs-casa/omni-cddlabs-casa`.
- Verify workflow logs for commit/push step.

### Argo app does not rollout
- Check app status:
  - `kubectl get applications -n argocd | grep rubinot-apps`
- Force refresh:
  - `kubectl -n argocd annotate application rubinot-apps argocd.argoproj.io/refresh=hard --overwrite`

### ConfigMap patch not updated
- Ensure `scripts/uc_patched_init.py` exists and workflow can generate configmap YAML.
- Validate `manifests/cddlabs/apps/rubinot/rubinot-data-cdp-patch-configmap.yaml` changed in GitOps commit.

## ArgoCD Image Updater note
Current delivery is GitOps write-back from CI (explicit and auditable). Image Updater can be layered later, but needs GHCR credentials and per-app annotations.

## Argo rollout verification in CI (new)
The release workflow now waits for ArgoCD to confirm deployment success after the GitOps commit.

Success criteria checked by workflow:
- `status.sync.revision == <gitops commit sha pushed by the workflow>`
- `status.sync.status == Synced`
- `status.health.status == Healthy`

Timeout: 15 minutes. If criteria are not met, workflow fails.

### Required secrets for verification
- `ARGOCD_SERVER` (e.g., `https://argocd.cddlabs.casa`)
- `ARGOCD_AUTH_TOKEN` (read-only Argo API token)

Optional (only if Cloudflare Access protects Argo endpoint):
- `CF_ACCESS_CLIENT_ID`
- `CF_ACCESS_CLIENT_SECRET`

This makes GitHub Actions reflect real deployment success, not only image push + GitOps commit.
