package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/noamtamir/gh-octoscope/internal/api"
	"github.com/noamtamir/gh-octoscope/internal/reports"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// newSyncCmd creates and returns the sync command
func newSyncCmd() *cobra.Command {
	var syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Sync GitHub Actions data to the server without generating a report",
		Long: `The sync command fetches GitHub Actions usage data and uploads it to the server.
Unlike the report command, it does not generate a report_id and does not create shareable reports.
This is useful for continuously syncing data to the server for analysis.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get GitHub CLI configuration
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

			// Run the sync logic
			return runSync(cfg, ghCLIConfig)
		},
	}

	return syncCmd
}

func runSync(cfg Config, ghCLIConfig GitHubCLIConfig) error {
	logger := setupLogger()

	// Fetch and process data from GitHub (without saving locally)
	jobDetails, totalCosts, err := fetchAndProcessData(cfg, ghCLIConfig, logger, false)
	if err != nil {
		return err
	}

	// Upload to server without report_id
	s := createSpinner("Syncing data to server...")
	s.Start()

	apiBaseUrl := os.Getenv("OCTOSCOPE_API_URL")
	if apiBaseUrl == "" {
		apiBaseUrl = "https://octoscope-server-production.up.railway.app"
	}

	osClient := api.NewOctoscopeClient(api.OctoscopeConfig{
		BaseUrl:     apiBaseUrl,
		Logger:      logger,
		GitHubToken: ghCLIConfig.Token,
	})

	reportData := &reports.ReportData{
		Jobs:          jobDetails,
		Totals:        totalCosts,
		ObfuscateData: cfg.Obfuscate,
	}

	err = syncToServer(osClient, reportData, logger)
	s.Stop()

	if err != nil {
		return fmt.Errorf("failed to sync data to server: %w", err)
	}

	fmt.Println(createSuccessMessage("Data synced successfully to server!"))

	logger.Debug().
		Str("total_duration", totalCosts.JobDuration.String()).
		Str("total_billable_duration", totalCosts.RoundedUpJobDuration.String()).
		Float64("total_billable_usd", totalCosts.BillableInUSD).
		Msg("Sync completed")

	return nil
}

func syncToServer(client api.OctoscopeClient, data *reports.ReportData, logger zerolog.Logger) error {
	const batchSize = 25
	const maxRetries = 3

	// Split jobs into batches and upload without report_id
	for i := 0; i < len(data.Jobs); i += batchSize {
		end := i + batchSize
		if end > len(data.Jobs) {
			end = len(data.Jobs)
		}

		batch := data.Jobs[i:end]

		// Retry logic for each batch
		var err error
		for retry := 0; retry < maxRetries; retry++ {
			// Use empty string for report_id to indicate sync operation
			err = client.SyncJobs(context.Background(), batch, data.ObfuscateData)
			if err == nil {
				break
			}
			logger.Warn().
				Int("retry", retry+1).
				Int("batch", i/batchSize+1).
				Err(err).
				Msg("Failed to upload batch, retrying...")

			if retry < maxRetries-1 {
				time.Sleep(time.Second * time.Duration(retry+1))
			}
		}
		if err != nil {
			return fmt.Errorf("failed to upload batch after %d retries: %w", maxRetries, err)
		}

		logger.Debug().
			Int("batch", i/batchSize+1).
			Int("total_batches", (len(data.Jobs)+batchSize-1)/batchSize).
			Msg("Batch uploaded successfully")
	}

	return nil
}
