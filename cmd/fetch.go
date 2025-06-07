package cmd

import (
	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
)

// newFetchCmd creates and returns the fetch command
func newFetchCmd() *cobra.Command {
	var fetchCmd = &cobra.Command{
		Use:   "fetch",
		Short: "Fetch GitHub Actions usage data",
		Long: `The fetch command retrieves GitHub Actions usage data from the GitHub API.
It only downloads and caches the data, without generating reports.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Force fetch to true since this is the fetch command
			cfg.Fetch = true

			// Disable all report generation
			cfg.FullReport = false
			cfg.CSVReport = false
			cfg.HTMLReport = false

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
			} else {
				cmd.Println("Data successfully fetched and stored for future use")
			}
		},
	}

	return fetchCmd
}
