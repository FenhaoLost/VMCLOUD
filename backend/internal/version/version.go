package version

var (
	Version = "1.1.0"
	Repo    = "MengMengCode/CLICD"
)

func Current() string {
	if Version == "" {
		return "dev"
	}
	return Version
}











