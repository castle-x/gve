package version

var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func Full() string {
	return "gve v" + Version + " (" + GitCommit + ") " + BuildDate
}
