package build

// Variables in this file are set via ldflags.
var (
	Version    = "0.0.0-dev"
	Commit     = "none"
	CommitDate = "unknown"

	SentryDSN    = "" // Disabled by default
	TelemetryKey = "" // Disabled by default
)
