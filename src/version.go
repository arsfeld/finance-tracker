package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// Version information that can be set at build time
var (
	// Version is the application version
	Version = "dev"
	// BuildTime is the time the binary was built
	BuildTime = "unknown"
	// GitCommit is the git commit hash
	GitCommit = getGitCommit()
)

// getGitCommit returns the current git commit hash
func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// GetVersion returns a formatted version string
func GetVersion() string {
	return fmt.Sprintf("finance_tracker v%s (build: %s, commit: %s)", Version, BuildTime, GitCommit)
}
