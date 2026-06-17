package store

import "testing"

// Locks in the cap boundaries shared by the handler and SearchProducts (#4):
// anything outside [1, MaxSearchLimit] collapses to DefaultSearchLimit.
func TestClampSearchLimit(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"within range", 10, 10},
		{"at max", MaxSearchLimit, MaxSearchLimit},
		{"over max falls back", 999, DefaultSearchLimit},
		{"just over max falls back", MaxSearchLimit + 1, DefaultSearchLimit},
		{"zero falls back", 0, DefaultSearchLimit},
		{"negative falls back", -5, DefaultSearchLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClampSearchLimit(tc.in); got != tc.want {
				t.Errorf("ClampSearchLimit(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
