package buildinfo

import "strings"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func FormattedVersion() string {
	version := strings.TrimSpace(Version)
	if version == "" {
		version = "dev"
	}

	parts := []string{version}

	if commit := strings.TrimSpace(Commit); commit != "" && commit != "unknown" {
		parts = append(parts, "commit="+commit)
	}

	if buildTime := strings.TrimSpace(BuildTime); buildTime != "" && buildTime != "unknown" {
		parts = append(parts, "built="+buildTime)
	}

	return strings.Join(parts, " ")
}
