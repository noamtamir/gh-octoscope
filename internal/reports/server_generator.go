package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	batchSize  = 25 // Size of batches for server upload
	maxRetries = 3  // Maximum number of retries for failed batch uploads
)

// Convert timestamps and avoid circular import by defining the interface we need
type octoscopeClient interface {
	BatchCreate(ctx context.Context, jobs []JobDetails, reportID string, shouldObfuscate bool) error
}

// ServerGenerator generates reports on our servers
type ServerGenerator struct {
	client octoscopeClient
	logger zerolog.Logger
	config ServerConfig
}

type ServerConfig struct {
	AppURL    string
	OwnerName string
	RepoName  string
}

// NewServerGenerator creates a new server report generator
func NewServerGenerator(client octoscopeClient, config ServerConfig, logger zerolog.Logger) *ServerGenerator {
	return &ServerGenerator{
		client: client,
		config: config,
		logger: logger,
	}
}

// Generate implements the Generator interface for Server reports
func (g *ServerGenerator) Generate(data *ReportData) error {
	// Generate report-id
	reportID := uuid.New().String()

	// Split jobs into batches
	for i := 0; i < len(data.Jobs); i += batchSize {
		end := i + batchSize
		if end > len(data.Jobs) {
			end = len(data.Jobs)
		}

		batch := data.Jobs[i:end]

		// Retry logic for each batch
		var err error
		for retry := 0; retry < maxRetries; retry++ {
			err = g.client.BatchCreate(context.Background(), batch, reportID, data.ObfuscateData)
			if err == nil {
				break
			}
			g.logger.Warn().
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

		g.logger.Info().
			Int("batch", i/batchSize+1).
			Int("total_batches", (len(data.Jobs)+batchSize-1)/batchSize).
			Msg("Batch uploaded successfully")
	}

	reportURL := fmt.Sprintf("%s/report/%s",
		g.config.AppURL,
		reportID)

	g.logger.Info().
		Str("report_url", reportURL).
		Msg("Report generated successfully. View your report at: " + reportURL)

	return nil
}
