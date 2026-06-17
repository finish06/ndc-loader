package store

import "testing"

// TestExcludeFilter verifies the SQL fragment that hides ndc_exclude=TRUE rows
// by default and opts back in when includeExcluded is true (issue #10).
func TestExcludeFilter(t *testing.T) {
	if got := excludeFilter(false); got != " AND ndc_exclude = FALSE" {
		t.Errorf("excludeFilter(false) = %q, want the exclude clause", got)
	}
	if got := excludeFilter(true); got != "" {
		t.Errorf("excludeFilter(true) = %q, want empty string", got)
	}
}
