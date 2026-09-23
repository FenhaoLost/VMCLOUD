package version

var (
	Version = "1.1.32"
	Repo    = "FenhaoLost/VMCLOUD"
)

func Current() string {
	if Version == "" {
		return "dev"
	}
	return Version
}
