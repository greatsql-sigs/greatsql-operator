package version

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-04-02 15:35:46
 * @file: version.go
 * @description: version
 */

var (
	Version   string = ""
	GitBranch string = ""
	GitCommit string = ""
	BuildTime string = ""
	GoVersion string = ""
	Compiler  string = ""
	Platform  string = ""
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the application version information",
	Run: func(cmd *cobra.Command, args []string) {
		v := getVersion()
		fmt.Println(string(v.json()))
	},
}

type versionInfo struct {
	Version   string `json:"Version"`
	GitBranch string `json:"GitBranch"`
	GitCommit string `json:"GitCommit"`
	BuildTime string `json:"BuildTime"`
	GoVersion string `json:"GoVersion"`
	Compiler  string `json:"Compiler"`
	Platform  string `json:"Platform"`
}

func (v *versionInfo) String() string {
	return v.GitCommit
}

func getVersion() versionInfo {
	return versionInfo{
		Version:   Version,
		GitBranch: GitBranch,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

func (v *versionInfo) json() json.RawMessage {
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	return jsonData
}
