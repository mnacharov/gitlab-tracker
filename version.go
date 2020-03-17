package main

var (
	Version string
)

func GetVersion() string {
	if len(Version) == 0 {
		return "X.X.X-dev"
	}
	return Version
}
