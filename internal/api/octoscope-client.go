package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/noamtamir/gh-octoscope/internal/reports"
	"github.com/rs/zerolog"
)

type OctoscopeClient interface {
	BatchCreate(ctx context.Context, jobs []reports.JobDetails, reportID string, shouldObfuscate bool) error
}

type octoscopeClient struct {
	osClient    *http.Client
	baseUrl     string
	logger      zerolog.Logger
	githubToken string
}

type OctoscopeConfig struct {
	BaseUrl     string
	Logger      zerolog.Logger
	GitHubToken string // GitHub token to be used for authentication
}

// NewOctoscopeClient creates a new Octoscope API client
func NewOctoscopeClient(cfg OctoscopeConfig) OctoscopeClient {
	return &octoscopeClient{
		osClient:    &http.Client{},
		baseUrl:     cfg.BaseUrl,
		logger:      cfg.Logger,
		githubToken: cfg.GitHubToken,
	}
}

func (c *octoscopeClient) BatchCreate(ctx context.Context, jobs []reports.JobDetails, reportID string, shouldObfuscate bool) error {
	flattened := reports.FlattenJobs(jobs, shouldObfuscate)

	payload := struct {
		ReportID string                   `json:"report_id"`
		Jobs     []reports.FlatJobDetails `json:"jobs"`
	}{
		ReportID: reportID,
		Jobs:     flattened,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal batch data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseUrl+"/jobs", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add GitHub token as Bearer token if available
	if c.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.githubToken)
	}

	resp, err := c.osClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send batch request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error: status=%d body=%s", resp.StatusCode, string(body))
	}

	c.logger.Debug().
		Int("job_count", len(jobs)).
		Str("report_id", reportID).
		Msg("Successfully uploaded job batch")

	return nil
}
