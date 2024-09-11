package main

import (
	"fmt"
	"os"

	"github.com/gagraler/greatsql-operator/internal/pkg/version"
	"github.com/spf13/cobra"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-09-11 19:26:30
 * @file: main.go
 * @description: cli tool for greatsql
 */

var (
	rootCmd = &cobra.Command{
		Use:   "GreatSQL Operator",
		Short: "GreatSQL Operator CLI Tool",
		Long:  "GreatSQL Operator CLI Tool is a command line interface for GreatSQL Operator.",
		Run: func(cmd *cobra.Command, args []string) {

			err := cmd.Help()
			if err != nil {
				cobraRootError(cmd, args, err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(version.VersionCmd)
}

// cobraRootError is a helper function to print error message and exit with code 1
func cobraRootError(cmd *cobra.Command, args []string, err error) {
	fmt.Fprintf(os.Stderr, "Execute %s args: %v error: %v\n", cmd.Name(), args, err)
	os.Exit(1)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
