package api

import (
	"os"
	"strings"
	"testing"
)

// TestSwaggerSpecCannotSilentlyDriftInCI reproduces issue #11: the compiled
// swagger spec drifted from the route annotations (missing /version and the
// api.HealthResponse / api.DependencyCheck schemas) because nothing in CI
// regenerated the spec and failed on a diff. The committed spec being correct
// today is not enough — without a CI guard it can silently drift again.
//
// Named for the observed symptom (silent drift reaching a release), not a
// theorized cause, so it survives a wrong root-cause guess.
func TestSwaggerSpecCannotSilentlyDriftInCI(t *testing.T) {
	const ciPath = "../../.github/workflows/ci.yml"

	data, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read %s: %v", ciPath, err)
	}
	ci := string(data)

	// CI must regenerate the swagger spec from the source annotations.
	if !strings.Contains(ci, "make docs") && !strings.Contains(ci, "swag init") {
		t.Error("CI does not regenerate the swagger spec (no `make docs` / `swag init`); the spec can drift from code silently")
	}

	// Regeneration must be deterministic — pin the swag version that produced
	// the committed spec, otherwise an upgraded swag would itself cause drift.
	if !strings.Contains(ci, "swag/cmd/swag@v1.8.1") {
		t.Error("CI does not pin swag to v1.8.1; an unpinned swag would produce a non-reproducible spec")
	}

	// CI must fail when the regenerated spec differs from what's committed.
	if !strings.Contains(ci, "git diff") || !strings.Contains(ci, "docs/swagger") {
		t.Error("CI does not fail on swagger drift (no `git diff` check against docs/swagger)")
	}
}
