// Package k6 holds regression guards for the committed k6 test harness.
//
// These are static source-scan tests: the k6 scripts are JavaScript run by the
// `k6` binary, so there is no JS test runner in this Go repo. The guards assert
// that secrets never get committed back into the harness or its Make targets.
package k6

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// leakedKey is the staging API credential that was committed to source as a
// fallback default in tests/k6/staging.js (Gitea issue Code/ndc-loader#6).
// It must never appear anywhere in the tree again.
const leakedKey = "pk_rxdag_staging_a8f3e1b9c4d7"

// repoRoot resolves the repository root from this test file's location so the
// guards work regardless of the test's working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine caller path")
	}
	// this file lives at <root>/tests/k6/secrets_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readSource(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// Issue #6 symptom: the staging API key is hardcoded as a `||` fallback default
// in tests/k6/staging.js, so anyone with repo read access gets a live key.
func TestStagingK6_HasNoHardcodedAPIKeyFallback(t *testing.T) {
	src := readSource(t, "tests/k6/staging.js")

	if strings.Contains(src, leakedKey) {
		t.Errorf("tests/k6/staging.js contains hardcoded staging API key %q; the key must come from __ENV.API_KEY only", leakedKey)
	}

	// The API_KEY assignment must not carry a `||` fallback default at all.
	if regexp.MustCompile(`API_KEY\s*=\s*__ENV\.API_KEY\s*\|\|`).MatchString(src) {
		t.Error("tests/k6/staging.js still defines API_KEY with a `||` fallback default; require __ENV.API_KEY with no default")
	}
}

// The script must fail loudly when API_KEY is absent rather than silently using
// a baked-in credential.
func TestStagingK6_RequiresAPIKeyEnvVar(t *testing.T) {
	src := readSource(t, "tests/k6/staging.js")

	if !strings.Contains(src, "__ENV.API_KEY") {
		t.Fatal("tests/k6/staging.js no longer reads __ENV.API_KEY")
	}
	if !regexp.MustCompile(`if\s*\(\s*!\s*API_KEY\s*\)`).MatchString(src) {
		t.Error("tests/k6/staging.js does not fail loudly on a missing API_KEY (expected an `if (!API_KEY) throw ...` guard)")
	}
}

// The same leaked key was also duplicated as a Makefile default and re-injected
// via `--env API_KEY=$(K6_API_KEY)`, which would defeat the staging.js fix.
func TestMakefile_HasNoHardcodedAPIKey(t *testing.T) {
	src := readSource(t, "Makefile")

	if strings.Contains(src, leakedKey) {
		t.Errorf("Makefile contains the leaked staging API key %q; remove the hardcoded default", leakedKey)
	}
}
