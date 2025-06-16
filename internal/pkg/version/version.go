package version

import (
	"runtime"
)

var (
	Version   string = ""
	GitBranch string = ""
	GitCommit string = ""
	BuildTime string = ""
	GoVersion string = ""
	Compiler  string = ""
	Platform  string = ""
)

type VersionInfo struct {
	Version   string `json:"Version"`
	GitBranch string `json:"GitBranch"`
	GitCommit string `json:"GitCommit"`
	BuildTime string `json:"BuildTime"`
	GoVersion string `json:"GoVersion"`
	Compiler  string `json:"Compiler"`
	Platform  string `json:"Platform"`
}

func GetVersion() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GitBranch: GitBranch,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}
