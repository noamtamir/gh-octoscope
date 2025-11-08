package cmd

import (
	"fmt"

	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
)

// newReportCmd creates and returns the report command
func newReportCmd() *cobra.Command {
	var fetch bool = true // By default, fetch is true

	var reportCmd = &cobra.Command{
		Use:   "report",
		Short: "Generate reports based on GitHub Actions usage data",
		Long: `The report command generates various types of reports based on GitHub Actions usage data.
It can generate CSV or full reports with different levels of detail.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// By default, if no subcommand is specified, we'll set the full report flag to true
			cfg.FullReport = true

			// Execute the root command's logic
			host, _ := auth.DefaultHost()
			token, _ := auth.TokenForHost(host)
			repo, err := repository.Current()
			if err != nil {
				return fmt.Errorf("failed to get current repository: %w", err)
			}

			ghCLIConfig := GitHubCLIConfig{
				Token: token,
				Repo:  repo,
			}

			// Run the application with fetchMode determined by the fetch flag
			return Run(cfg, ghCLIConfig, fetch)
		},
	}

	// Add flags specific to the report command
	reportCmd.Flags().BoolVar(&cfg.CSVReport, "csv", false, "Generate CSV report")
	reportCmd.Flags().BoolVar(&fetch, "fetch", true, "Whether to fetch new data or use existing data")
	// Note: obfuscate is now a persistent flag defined in the root command

	// Add subcommands
	reportCmd.AddCommand(
		newDeleteCmd(),
	)

	return reportCmd
}
