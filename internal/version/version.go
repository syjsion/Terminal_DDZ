package version

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func String() string {
	return "terminal-ddz " + Version + " (commit " + Commit + ", built " + Date + ")"
}
