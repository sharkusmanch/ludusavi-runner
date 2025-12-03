// Package version provides build-time version information.
package version

import (
	"fmt"
	"runtime"
)

// These variables are set at build time via ldflags.
var (
	// Version is the semantic version of the application.
	Version = "dev"
	// Commit is the git commit hash.
	Commit = "unknown"
	// Date is the build date.
	Date = "unknown"
)

// Info contains version information.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current version information.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a human-readable version string.
func (i Info) String() string {
	return fmt.Sprintf("ludusavi-runner %s (commit: %s, built: %s, %s, %s/%s)",
		i.Version, i.Commit, i.Date, i.GoVersion, i.OS, i.Arch)
}

// Short returns a short version string.
func (i Info) Short() string {
	return i.Version
}
