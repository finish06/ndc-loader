# Spec: Release Tagging

**Version:** 0.1.0
**Created:** 2026-04-04
**PRD Reference:** docs/prd.md
**Status:** Done
**Completed:** 2026-04-04
**Milestone:** M4 — Production Readiness

## 1. Overview

Semantic versioning strategy and automated release workflow. When a `v*` tag is pushed, CI builds versioned Docker images, pushes to both registries, and generates a GitHub Release with changelog.

### User Story

As a **platform operator**, I want tagged releases with version-stamped Docker images and GitHub Releases, so that I can deploy specific versions to production and roll back to known-good releases.

## 2. Acceptance Criteria

| ID | Criterion | Priority |
|----|-----------|----------|
| AC-001 | Tags follow semantic versioning: `v{major}.{minor}.{patch}` (e.g., v0.1.0, v1.0.0) | Must |
| AC-002 | Pushing a `v*` tag triggers CI publish job | Must |
| AC-003 | CI builds Docker image with version injected via ldflags | Must |
| AC-004 | Image pushed as `:version` and `:latest` to dockerhub.calebdunn.tech + ghcr.io | Must |
| AC-005 | GitHub Release created automatically with changelog excerpt | Must |
| AC-006 | `/version` endpoint returns the tagged version (not "dev") | Must |
| AC-007 | CHANGELOG.md `[Unreleased]` section moved to `[version] - date` on release | Should |
| AC-008 | `make release VERSION=0.2.0` creates tag + pushes (convenience target) | Should |

## 3. Workflow

### Tagging a Release

```bash
# Option 1: Manual
git tag -a v0.2.0 -m "v0.2.0 — description"
git push origin v0.2.0

# Option 2: Makefile
make release VERSION=0.2.0
```

### CI Pipeline (already implemented for v0.1.0)

```
tag v* → lint + test → build image with ldflags
  → push rx-dag:0.2.0 + rx-dag:latest to both registries
  → create GitHub Release with changelog
```

### GitHub Release (new)

CI creates a GitHub Release using `gh release create` with:
- Tag name as title
- Changelog excerpt from `[Unreleased]` section
- Docker image references
- Link to Swagger docs

## 4. Makefile Target

```makefile
release: ## Tag and push a release (usage: make release VERSION=0.2.0)
	@test -n "$(VERSION)" || (echo "Usage: make release VERSION=x.y.z" && exit 1)
	@echo "Tagging v$(VERSION)..."
	git tag -a v$(VERSION) -m "v$(VERSION)"
	git push origin v$(VERSION)
	@echo "Release v$(VERSION) tagged and pushed. CI will build and publish."
```

## 5. CI Addition — GitHub Release

Add to `.github/workflows/ci.yml` publish job after the release image push:

```yaml
- name: Create GitHub Release
  if: steps.version.outputs.is_release == 'true'
  run: |
    gh release create v${{ steps.version.outputs.tag }} \
      --title "v${{ steps.version.outputs.tag }}" \
      --notes "Docker: dockerhub.calebdunn.tech/finish06/rx-dag:${{ steps.version.outputs.tag }}"
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## 6. Dependencies

- Existing CI pipeline (already handles `v*` tags)
- Existing ldflags injection in Dockerfile
- `gh` CLI available in CI runners

## 7. Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-04-04 | 0.1.0 | calebdunn | Initial spec |
