package api

import (
	"testing"
)

func TestSetBuildInfo_SetsValues(t *testing.T) {
	// Save original and restore after test.
	orig := BuildInfo
	defer func() { BuildInfo = orig }()

	SetBuildInfo("1.0.0", "abc123", "main", "2026-01-01T00:00:00Z")

	if BuildInfo.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", BuildInfo.Version)
	}
	if BuildInfo.GitCommit != "abc123" {
		t.Errorf("expected commit abc123, got %s", BuildInfo.GitCommit)
	}
	if BuildInfo.GitBranch != "main" {
		t.Errorf("expected branch main, got %s", BuildInfo.GitBranch)
	}
	if BuildInfo.BuildTime != "2026-01-01T00:00:00Z" {
		t.Errorf("expected build time, got %s", BuildInfo.BuildTime)
	}
}

func TestSetBuildInfo_SkipsEmpty(t *testing.T) {
	orig := BuildInfo
	defer func() { BuildInfo = orig }()

	BuildInfo.Version = "existing"
	BuildInfo.GitCommit = "existing"

	SetBuildInfo("", "", "", "")

	if BuildInfo.Version != "existing" {
		t.Errorf("expected version unchanged, got %s", BuildInfo.Version)
	}
	if BuildInfo.GitCommit != "existing" {
		t.Errorf("expected commit unchanged, got %s", BuildInfo.GitCommit)
	}
}
