package version

import (
	"runtime"
)

var (
	Version   = ""
	GitBranch = ""
	GitCommit = ""
	BuildTime = ""
	GoVersion = ""
	Compiler  = ""
	Platform  = ""
)

type Info struct {
	Version   string `json:"Version"`
	GitBranch string `json:"GitBranch"`
	GitCommit string `json:"GitCommit"`
	BuildTime string `json:"BuildTime"`
	GoVersion string `json:"GoVersion"`
	Compiler  string `json:"Compiler"`
	Platform  string `json:"Platform"`
}

func GetVersion() Info {
	return Info{
		Version:   Version,
		GitBranch: GitBranch,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}
