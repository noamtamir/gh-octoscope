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
	SyncJobs(ctx context.Context, jobs []reports.JobDetails, shouldObfuscate bool) error
	DeleteReport(ctx context.Context, reportID string) error
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

// doJSONRequest is a helper method for making JSON POST requests with authentication
func (c *octoscopeClient) doJSONRequest(ctx context.Context, method, endpoint string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+endpoint, bytes.NewBuffer(jsonData))
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
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
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

	if err := c.doJSONRequest(ctx, "POST", "/report-jobs", payload); err != nil {
		return err
	}

	c.logger.Debug().
		Int("job_count", len(jobs)).
		Str("report_id", reportID).
		Msg("Successfully uploaded job batch")

	return nil
}

func (c *octoscopeClient) SyncJobs(ctx context.Context, jobs []reports.JobDetails, shouldObfuscate bool) error {
	flattened := reports.FlattenJobs(jobs, shouldObfuscate)

	payload := struct {
		Jobs []reports.FlatJobDetails `json:"jobs"`
	}{
		Jobs: flattened,
	}

	if err := c.doJSONRequest(ctx, "POST", "/jobs", payload); err != nil {
		return err
	}

	c.logger.Debug().
		Int("job_count", len(jobs)).
		Msg("Successfully uploaded job batch (sync)")

	return nil
}

func (c *octoscopeClient) DeleteReport(ctx context.Context, reportID string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseUrl+"/report-jobs", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("report_id", reportID)
	req.URL.RawQuery = q.Encode()

	// Add GitHub token as Bearer token if available
	if c.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.githubToken)
	}

	resp, err := c.osClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error: status=%d body=%s", resp.StatusCode, string(body))
	}

	c.logger.Debug().
		Str("report_id", reportID).
		Msg("Successfully deleted report")

	return nil
}
