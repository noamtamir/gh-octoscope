package reports

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestData() *ReportData {
	// Create sample job details for testing
	repo := &github.Repository{
		Owner: &github.User{Login: github.String("testowner")},
		Name:  github.String("testrepo"),
		ID:    github.Int64(123456),
	}

	workflow := &github.Workflow{
		ID:   github.Int64(7890),
		Name: github.String("Test Workflow"),
	}

	now := time.Now()

	workflowRun := &github.WorkflowRun{
		ID:           github.Int64(12345),
		Name:         github.String("Test Run"),
		RunNumber:    github.Int(42),
		RunAttempt:   github.Int(1),
		HeadBranch:   github.String("main"),
		HeadSHA:      github.String("abcdef123456"),
		Event:        github.String("push"),
		Status:       github.String("completed"),
		Conclusion:   github.String("success"),
		CreatedAt:    &github.Timestamp{Time: now.Add(-1 * time.Hour)},
		UpdatedAt:    &github.Timestamp{Time: now},
		RunStartedAt: &github.Timestamp{Time: now.Add(-1 * time.Hour)},
		DisplayTitle: github.String("Test Run Display Title"),
		Actor:        &github.User{Login: github.String("testactor")},
	}

	job := &github.WorkflowJob{
		ID:              github.Int64(987654),
		Name:            github.String("Test Job"),
		Status:          github.String("completed"),
		Conclusion:      github.String("success"),
		StartedAt:       &github.Timestamp{Time: now.Add(-30 * time.Minute)},
		CompletedAt:     &github.Timestamp{Time: now.Add(-15 * time.Minute)},
		CreatedAt:       &github.Timestamp{Time: now.Add(-40 * time.Minute)},
		RunnerID:        github.Int64(1),
		RunnerName:      github.String("ubuntu-latest"),
		RunnerGroupID:   github.Int64(1),
		RunnerGroupName: github.String("GitHub Actions"),
		RunAttempt:      github.Int64(1),
		Labels:          []string{"ubuntu-latest"},
	}

	// Create job details
	jobDetails := []JobDetails{
		{
			Repo:                 repo,
			Workflow:             workflow,
			WorkflowRun:          workflowRun,
			Job:                  job,
			JobDuration:          25 * time.Minute,
			RoundedUpJobDuration: 25 * time.Minute,
			PricePerMinuteInUSD:  0.008,
			BillableInUSD:        0.2,
			Runner:               "UBUNTU",
		},
	}

	// Create total costs
	totalCosts := TotalCosts{
		JobDuration:          25 * time.Minute,
		RoundedUpJobDuration: 25 * time.Minute,
		BillableInUSD:        0.2,
	}

	return &ReportData{
		Jobs:          jobDetails,
		Totals:        totalCosts,
		ObfuscateData: false,
	}
}

func TestCSVGenerator(t *testing.T) {
	// Create a temporary directory for test outputs
	tmpDir, err := os.MkdirTemp("", "csv-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create paths for test outputs
	reportPath := filepath.Join(tmpDir, "report.csv")
	totalsPath := filepath.Join(tmpDir, "totals.csv")

	// Create silent logger
	logger := zerolog.New(io.Discard)

	// Create generator
	generator := NewCSVGenerator(reportPath, totalsPath, logger)

	// Generate the report
	require.NoError(t, generator.Generate(setupTestData()))

	// Verify files were created
	assert.FileExists(t, reportPath)
	assert.FileExists(t, totalsPath)

	// Read report.csv
	reportContent, err := os.ReadFile(reportPath)
	require.NoError(t, err)

	// Basic content validation (expecting CSV header and at least one row)
	reportLines := len(splitLines(string(reportContent)))
	assert.GreaterOrEqual(t, reportLines, 2, "Expected at least header and one data row in report.csv")

	// Read totals.csv
	totalsContent, err := os.ReadFile(totalsPath)
	require.NoError(t, err)

	// Basic content validation
	totalsLines := len(splitLines(string(totalsContent)))
	assert.GreaterOrEqual(t, totalsLines, 2, "Expected at least header and one data row in totals.csv")
}

// Mock implementation of octoscopeClient for testing
type mockOctoscopeClient struct {
	batchCreateCalled      bool
	batchCreateJobs        []JobDetails
	batchCreateReportID    string
	batchCreateObfuscation bool
	batchCreateError       error

	deleteReportCalled bool
	deleteReportID     string
	deleteReportError  error
}

func (m *mockOctoscopeClient) BatchCreate(ctx context.Context, jobs []JobDetails, reportID string, shouldObfuscate bool) error {
	m.batchCreateCalled = true
	m.batchCreateJobs = jobs
	m.batchCreateReportID = reportID
	m.batchCreateObfuscation = shouldObfuscate
	return m.batchCreateError
}

func (m *mockOctoscopeClient) DeleteReport(ctx context.Context, reportID string) error {
	m.deleteReportCalled = true
	m.deleteReportID = reportID
	return m.deleteReportError
}

func TestServerGenerator(t *testing.T) {
	// Create silent logger
	logger := zerolog.New(io.Discard)

	// Create mock client
	mockClient := &mockOctoscopeClient{}

	// Create server config
	config := ServerConfig{
		AppURL:    "https://notreal.url",
		OwnerName: "testowner",
		RepoName:  "testrepo",
	}

	// Create generator
	generator := NewServerGenerator(mockClient, config, logger)

	// Generate the report
	require.NoError(t, generator.Generate(setupTestData()))

	// Verify BatchCreate was called with correct parameters
	assert.True(t, mockClient.batchCreateCalled)
	assert.Len(t, mockClient.batchCreateJobs, 1)
	assert.NotEmpty(t, mockClient.batchCreateReportID)
	assert.False(t, mockClient.batchCreateObfuscation)
}

func TestFlattenJobsAndObfuscation(t *testing.T) {
	// Create test data
	testData := setupTestData()

	t.Run("NoObfuscation", func(t *testing.T) {
		flattened := FlattenJobs(testData.Jobs, false)

		assert.Len(t, flattened, 1)
		job := flattened[0]

		// Check that values are preserved
		assert.Equal(t, "testowner", job.OwnerName)
		assert.Equal(t, "testrepo", job.RepoName)
		assert.Equal(t, "Test Workflow", job.WorkflowName)
		assert.Equal(t, "Test Job", job.JobName)
	})

	t.Run("WithObfuscation", func(t *testing.T) {
		flattened := FlattenJobs(testData.Jobs, true)

		assert.Len(t, flattened, 1)
		job := flattened[0]

		// Check that values are obfuscated
		assert.NotEqual(t, "testowner", job.OwnerName)
		assert.NotEqual(t, "testrepo", job.RepoName)

		// Check obfuscation pattern for some fields
		assert.Regexp(t, "^tes\\*+$", job.OwnerName)
		assert.Regexp(t, "^tes\\*+$", job.RepoName)
	})
}

// Helper function to split string by newlines
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}
