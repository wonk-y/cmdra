// Package buildinfo exposes build and release metadata for the binaries.
package buildinfo

import "strings"

// These variables are intended to be overridden with -ldflags during release builds.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Summary renders a compact human-readable build string.
func Summary(binary string) string {
	parts := []string{binary, Version}
	if Commit != "" && Commit != "unknown" {
		parts = append(parts, "commit="+Commit)
	}
	if Date != "" && Date != "unknown" {
		parts = append(parts, "date="+Date)
	}
	return strings.Join(parts, " ")
}
