package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd creates and returns the version command
func newVersionCmd() *cobra.Command {
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version of gh-octoscope",
		Long:  "Print the version of gh-octoscope",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gh-octoscope version: %s\n", version)
			fmt.Printf("Git commit: %s\n", commit)
			fmt.Printf("Built: %s\n", date)
		},
	}
	return versionCmd
}
