package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wantedly/valec/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.String())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
