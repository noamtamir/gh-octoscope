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
	client    octoscopeClient
	logger    zerolog.Logger
	config    ServerConfig
	reportID  string // Custom report ID, if provided
	reportURL string // Generated report URL
}

type ServerConfig struct {
	AppURL    string
	OwnerName string
	RepoName  string
	ReportID  string // Optional custom report ID
}

func NewServerGenerator(client octoscopeClient, config ServerConfig, logger zerolog.Logger) *ServerGenerator {
	reportURL := fmt.Sprintf("%s/report/%s",
		config.AppURL,
		config.ReportID)
	return &ServerGenerator{
		client:    client,
		config:    config,
		logger:    logger,
		reportID:  config.ReportID,
		reportURL: reportURL,
	}
}

func (g *ServerGenerator) GetReportURL() string {
	return g.reportURL
}

func (g *ServerGenerator) Generate(data *ReportData) error {
	// Use provided report ID or generate a new one
	reportID := g.reportID
	if reportID == "" {
		reportID = uuid.New().String()
	}

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

		g.logger.Debug().
			Int("batch", i/batchSize+1).
			Int("total_batches", (len(data.Jobs)+batchSize-1)/batchSize).
			Msg("Batch uploaded successfully")
	}

	g.logger.Debug().
		Str("report_url", g.reportURL).
		Msg("Report generated successfully. View your report at: " + g.reportURL)

	return nil
}
