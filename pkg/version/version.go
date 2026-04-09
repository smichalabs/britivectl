package version

//nolint:revive // these vars are set via ldflags at build time, not in code
var (
	Version   = "0.0.1-alpha"
	Commit    = "dev"
	BuildDate = "unknown"
)

// String returns the full version string.
func String() string {
	return "bctl " + Version + " (commit: " + Commit + ", built: " + BuildDate + ")"
}
