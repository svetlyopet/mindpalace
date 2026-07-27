package version

import "fmt"

// Set at link time via -X (see .goreleaser.yaml).
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info returns a string suitable for cobra.Command.Version.
func Info() string {
	if Commit != "none" && len(Commit) >= 7 {
		return fmt.Sprintf("%s (%s)", Version, Commit[:7])
	}
	if Commit != "none" {
		return fmt.Sprintf("%s (%s)", Version, Commit)
	}
	return Version
}
