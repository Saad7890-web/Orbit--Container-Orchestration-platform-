package version

var (
	// Set by ldflags at release time:
	//   -X github.com/saad/orbit/internal/version.Version=v0.1.0
	Version = "dev"
)

func String() string {
	return Version
}