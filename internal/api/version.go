package api

import (
	"encoding/json"
	"net/http"
	"runtime"
)

// BuildInfo holds version and build metadata injected at compile time via ldflags.
var BuildInfo = struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	BuildTime string `json:"build_time"`
}{
	Version:   "dev",
	GitCommit: "unknown",
	GitBranch: "unknown",
	GoVersion: runtime.Version(),
	OS:        runtime.GOOS,
	Arch:      runtime.GOARCH,
	BuildTime: "unknown",
}

// SetBuildInfo sets the build-time injected values.
func SetBuildInfo(version, gitCommit, gitBranch, buildTime string) {
	if version != "" {
		BuildInfo.Version = version
	}
	if gitCommit != "" {
		BuildInfo.GitCommit = gitCommit
	}
	if gitBranch != "" {
		BuildInfo.GitBranch = gitBranch
	}
	if buildTime != "" {
		BuildInfo.BuildTime = buildTime
	}
}

// versionHandler returns the version endpoint handler.
//
//	@Summary		Build and version info
//	@Description	Returns git commit, branch, Go version, OS, architecture, and build time. No authentication required.
//	@Tags			Operations
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Router			/version [get]
func versionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BuildInfo)
	}
}
