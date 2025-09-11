package version

import (
	"fmt"
	"runtime"
)

// Build information. Populated at build-time via ldflags.
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
	GoVersion = runtime.Version()
)

// BuildInfo contains all the build-time information.
type BuildInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	GoVersion string `json:"go_version"`
}

// Get returns the build information.
func Get() BuildInfo {
	return BuildInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		GoVersion: GoVersion,
	}
}

// String returns a formatted string containing version information.
func String() string {
	info := Get()
	return fmt.Sprintf("Version: %s\nBuild Time: %s\nGit Commit: %s\nGit Branch: %s\nGo Version: %s",
		info.Version, info.BuildTime, info.GitCommit, info.GitBranch, info.GoVersion)
}

// Short returns a short version string.
func Short() string {
	if GitCommit != "unknown" && len(GitCommit) > 7 {
		return fmt.Sprintf("%s (%s)", Version, GitCommit[:7])
	}
	return Version
}