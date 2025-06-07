package cmd

import (
	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
)

// newReportCmd creates and returns the report command
func newReportCmd() *cobra.Command {
	var reportCmd = &cobra.Command{
		Use:   "report",
		Short: "Generate reports based on GitHub Actions usage data",
		Long: `The report command generates various types of reports based on GitHub Actions usage data.
It can generate CSV, HTML, or full reports with different levels of detail.`,
		Run: func(cmd *cobra.Command, args []string) {
			// By default, if no subcommand is specified, we'll set the full report flag to true
			cfg.FullReport = true

			// Execute the root command's logic
			host, _ := auth.DefaultHost()
			token, _ := auth.TokenForHost(host)
			repo, err := repository.Current()
			if err != nil {
				cmd.PrintErrf("Failed to get current repository: %v\n", err)
				return
			}

			ghCLIConfig := GitHubCLIConfig{
				Token: token,
				Repo:  repo,
			}

			// Run the application
			if err := Run(cfg, ghCLIConfig); err != nil {
				cmd.PrintErrf("Error: %v\n", err)
			}
		},
	}

	// Add flags specific to the report command
	reportCmd.Flags().BoolVar(&cfg.CSVReport, "csv", false, "Generate CSV report")
	reportCmd.Flags().BoolVar(&cfg.HTMLReport, "html", false, "Generate HTML report")
	reportCmd.Flags().BoolVar(&cfg.Obfuscate, "obfuscate", false, "Obfuscate sensitive data in the report")

	return reportCmd
}
