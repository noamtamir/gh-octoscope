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
	t.Run("BasicGenerator", func(t *testing.T) {
		// Create a temporary directory for test outputs
		tmpDir, err := os.MkdirTemp("", "csv-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create paths for test outputs
		reportPath := filepath.Join(tmpDir, "report.csv")
		totalsPath := filepath.Join(tmpDir, "totals.csv")

		// Create silent logger
		logger := zerolog.New(io.Discard)

		// Create basic generator
		generator := NewCSVGenerator(reportPath, totalsPath, logger)

		// Generate the report
		err = generator.Generate(setupTestData())
		require.NoError(t, err)

		// Verify paths can be retrieved from generator
		assert.Equal(t, reportPath, generator.GetJobsPath())
		assert.Equal(t, totalsPath, generator.GetTotalsPath())

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

		// Verify new columns are present in the header
		totalsStr := string(totalsContent)
		assert.Contains(t, totalsStr, "report_id")
		assert.Contains(t, totalsStr, "owner")
		assert.Contains(t, totalsStr, "repository")
		assert.Contains(t, totalsStr, "report_created_at")
	})

	t.Run("FormattedGenerator", func(t *testing.T) {
		// Create a temporary directory for test outputs
		tmpDir, err := os.MkdirTemp("", "csv-test-formatted")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create silent logger
		logger := zerolog.New(io.Discard)

		// Test parameters
		owner := "testowner"
		repo := "testrepo"
		reportID := "test-report-id"

		// Create formatted generator
		generator := NewCSVGeneratorWithFormat(tmpDir, owner, repo, reportID, logger)

		// Generate the report
		err2 := generator.Generate(setupTestData())
		require.NoError(t, err2)

		// Verify paths can be retrieved from the generator
		assert.Contains(t, generator.GetJobsPath(), tmpDir)
		assert.Contains(t, generator.GetTotalsPath(), tmpDir)

		// List all files in the directory
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		// Check that we have exactly two files (report and totals)
		require.Equal(t, 2, len(files), "Expected exactly two files in the output directory")

		// Check that filenames match the expected pattern
		foundReport := false
		foundTotals := false

		for _, file := range files {
			name := file.Name()
			// Check for timestamp format (2006-01-02T15:04:05)
			if strings.Contains(name, "_"+owner+"_"+repo+"_"+reportID+"_report.csv") {
				foundReport = true

				// Read file content
				content, err := os.ReadFile(filepath.Join(tmpDir, name))
				require.NoError(t, err)

				// Verify content
				reportLines := len(splitLines(string(content)))
				assert.GreaterOrEqual(t, reportLines, 2, "Expected at least header and one data row in report.csv")
			}

			if strings.Contains(name, "_"+owner+"_"+repo+"_"+reportID+"_totals.csv") {
				foundTotals = true

				// Read file content
				content, err := os.ReadFile(filepath.Join(tmpDir, name))
				require.NoError(t, err)

				// Verify content
				totalsLines := len(splitLines(string(content)))
				assert.GreaterOrEqual(t, totalsLines, 2, "Expected at least header and one data row in totals.csv")

				// Verify new columns are present in the header
				totalsStr := string(content)
				assert.Contains(t, totalsStr, "report_id")
				assert.Contains(t, totalsStr, "owner")
				assert.Contains(t, totalsStr, "repository")
				assert.Contains(t, totalsStr, "report_created_at")
			}
		}

		assert.True(t, foundReport, "Report file with formatted name not found")
		assert.True(t, foundTotals, "Totals file with formatted name not found")
	})
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

func (m *mockOctoscopeClient) SyncJobs(ctx context.Context, jobs []JobDetails, shouldObfuscate bool) error {
	// For tests that don't use sync, just return nil
	return nil
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

	// Custom report ID
	customReportID := "custom-report-id-12345"

	// Create server config with custom report ID
	config := ServerConfig{
		AppURL:    "https://notreal.url",
		OwnerName: "testowner",
		RepoName:  "testrepo",
		ReportID:  customReportID,
	}

	// Create generator
	generator := NewServerGenerator(mockClient, config, logger)

	// Generate the report
	err := generator.Generate(setupTestData())
	require.NoError(t, err)

	// Verify the report URL via the getter
	reportURL := generator.GetReportURL()
	assert.Contains(t, reportURL, customReportID)

	// Verify BatchCreate was called with correct parameters
	assert.True(t, mockClient.batchCreateCalled)
	assert.Len(t, mockClient.batchCreateJobs, 1)
	assert.Equal(t, customReportID, mockClient.batchCreateReportID)
	assert.False(t, mockClient.batchCreateObfuscation)
}

func TestFlattenJobsAndObfuscation(t *testing.T) {
	// Create test data
	testData := setupTestData()

	t.Run("NoObfuscation", func(t *testing.T) {
		flattened := FlattenJobs(testData.Jobs, false)

		assert.Len(t, flattened, 1)
		job := flattened[0]

		// Check that string values are preserved
		assert.Equal(t, "testowner", derefStr(job.OwnerName))
		assert.Equal(t, "testrepo", derefStr(job.RepoName))
		assert.Equal(t, "Test Workflow", derefStr(job.WorkflowName))
		assert.Equal(t, "Test Job", derefStr(job.JobName))

		// Check numeric values are preserved and correct type
		assert.Equal(t, int64(123456), derefInt64(job.RepoID))
		assert.Equal(t, int64(7890), derefInt64(job.WorkflowID))
		assert.Equal(t, int64(12345), derefInt64(job.WorkflowRunID))
		assert.Equal(t, int(42), derefInt(job.WorkflowRunRunNumber))
		assert.Equal(t, int(1), derefInt(job.WorkflowRunRunAttempt))
		assert.Equal(t, int64(987654), derefInt64(job.JobID))

		// Check duration values
		assert.Equal(t, float64(25*60), derefFloat64(job.JobDurationSeconds)) // 25 minutes in seconds
		assert.Equal(t, "25m0s", derefStr(job.JobDurationHumanReadable))
		assert.Equal(t, float64(25*60), derefFloat64(job.RoundedUpJobDurationSeconds))
		assert.Equal(t, "25m0s", derefStr(job.RoundedUpJobDurationHumanReadable))

		// Check price values
		assert.Equal(t, 0.008, derefFloat64(job.PricePerMinuteInUSD))
		assert.Equal(t, 0.2, derefFloat64(job.BillableInUSD))
	})

	t.Run("WithObfuscation", func(t *testing.T) {
		flattened := FlattenJobs(testData.Jobs, true)

		assert.Len(t, flattened, 1)
		job := flattened[0]

		// Check that values are obfuscated
		assert.NotEqual(t, "testowner", derefStr(job.OwnerName))
		assert.NotEqual(t, "testrepo", derefStr(job.RepoName))

		// Check obfuscation pattern for some fields
		assert.Regexp(t, "^tes\\*+$", derefStr(job.OwnerName))
		assert.Regexp(t, "^tes\\*+$", derefStr(job.RepoName))

		// Check numeric values are not affected by obfuscation
		assert.Equal(t, int64(123456), derefInt64(job.RepoID))
		assert.Equal(t, int64(7890), derefInt64(job.WorkflowID))
		assert.Equal(t, float64(25*60), derefFloat64(job.JobDurationSeconds))
	})
}

// Helper function to split string by newlines
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}

// Helper functions for pointer dereferencing in tests
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}
func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
func derefFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
