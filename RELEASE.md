# RELEASE.md

Quick release flow (tag-driven deploy):

```bash
git checkout main
git pull --ff-only
git push origin main

TAG=vX.Y.Z
git tag "$TAG"
git push origin "$TAG"
```

What happens automatically:
1. GitHub Actions builds and pushes image to GHCR.
2. Workflow updates GitOps manifests in `omni-cddlabs-casa`.
3. ArgoCD (`rubinot-apps`) auto-sync deploys.

Useful checks:
```bash
gh run list --repo giovannirco/rubinot-data --limit 5
kubectl get applications -n argocd | grep rubinot-apps
```
