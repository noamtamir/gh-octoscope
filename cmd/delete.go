package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/cli/go-gh/pkg/auth"
	"github.com/noamtamir/gh-octoscope/internal/api"
	"github.com/spf13/cobra"
)

// newDeleteCmd creates and returns the delete subcommand for the report command
func newDeleteCmd() *cobra.Command {
	var deleteCmd = &cobra.Command{
		Use:   "delete [reportID]",
		Short: "Delete a report from Octoscope server",
		Long:  `Delete a report with the specified ID from the Octoscope server.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reportID := args[0]

			// Setup Octoscope client
			host, _ := auth.DefaultHost()
			token, _ := auth.TokenForHost(host)

			// Get the API URL from environment or default
			apiURL := os.Getenv("OCTOSCOPE_API_URL")
			if apiURL == "" {
				apiURL = "https://octoscope-server-production.up.railway.app"
			}

			// Setup logger
			logger := setupLogger()

			// Create Octoscope client
			octoscopeClient := api.NewOctoscopeClient(api.OctoscopeConfig{
				BaseUrl:     apiURL,
				Logger:      logger,
				GitHubToken: token,
			})

			// Delete the report
			ctx := context.Background()
			if err := octoscopeClient.DeleteReport(ctx, reportID); err != nil {
				return fmt.Errorf("error deleting report: %w", err)
			}

			cmd.Printf("Report %s deleted successfully\n", reportID)
			return nil
		},
	}

	return deleteCmd
}
