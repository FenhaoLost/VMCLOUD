package version

var (
	Version = "1.0.12"
	Repo    = "MengMengCode/CLICD"
)

func Current() string {
	if Version == "" {
		return "dev"
	}
	return Version
}







