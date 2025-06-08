package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
	"github.com/noamtamir/gh-octoscope/cmd"
	"github.com/noamtamir/gh-octoscope/internal/api"
	"github.com/noamtamir/gh-octoscope/internal/billing"
	"github.com/noamtamir/gh-octoscope/internal/reports"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock GitHub API client
type mockGitHubClient struct {
	mock.Mock
}

func (m *mockGitHubClient) GetRepository(ctx context.Context) (*github.Repository, error) {
	args := m.Called(ctx)
	return args.Get(0).(*github.Repository), args.Error(1)
}

func (m *mockGitHubClient) ListWorkflows(ctx context.Context) (*github.Workflows, error) {
	args := m.Called(ctx)
	return args.Get(0).(*github.Workflows), args.Error(1)
}

func (m *mockGitHubClient) ListRepositoryRuns(ctx context.Context, from time.Time) (*github.WorkflowRuns, error) {
	args := m.Called(ctx, from)
	return args.Get(0).(*github.WorkflowRuns), args.Error(1)
}

func (m *mockGitHubClient) ListWorkflowJobs(ctx context.Context, runID int64) (*github.Jobs, error) {
	args := m.Called(ctx, runID)
	return args.Get(0).(*github.Jobs), args.Error(1)
}

func (m *mockGitHubClient) ListWorkflowJobsAttempt(ctx context.Context, runID, attempt int64) (*github.Jobs, error) {
	args := m.Called(ctx, runID, attempt)
	return args.Get(0).(*github.Jobs), args.Error(1)
}

func (m *mockGitHubClient) GetWorkflowRunUsage(ctx context.Context, runID int64) (*github.WorkflowRunUsage, error) {
	args := m.Called(ctx, runID)
	return args.Get(0).(*github.WorkflowRunUsage), args.Error(1)
}

// Mock Octoscope API client
type mockOctoscopeClient struct {
	mock.Mock
}

func (m *mockOctoscopeClient) BatchCreate(ctx context.Context, jobs []reports.JobDetails, reportID string, shouldObfuscate bool) error {
	args := m.Called(ctx, jobs, reportID, shouldObfuscate)
	return args.Error(0)
}

func (m *mockOctoscopeClient) DeleteReport(ctx context.Context, reportID string) error {
	args := m.Called(ctx, reportID)
	return args.Error(0)
}

func TestCobraCommands(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name     string
		args     []string
		expected cmd.Config
	}{
		{
			name: "Default values",
			args: []string{"gh-octoscope"},
			expected: cmd.Config{
				PageSize: 30,
			},
		},
		{
			name: "Debug mode",
			args: []string{"gh-octoscope", "--debug"},
			expected: cmd.Config{
				Debug:    true,
				PageSize: 30,
			},
		},
		{
			name: "Production logger",
			args: []string{"gh-octoscope", "--prod-log"},
			expected: cmd.Config{
				ProdLogger: true,
				PageSize:   30,
			},
		},
		{
			name: "From date",
			args: []string{"gh-octoscope", "--from=2025-04-01"},
			expected: cmd.Config{
				FromDate: "2025-04-01",
				PageSize: 30,
			},
		},
		{
			name: "With obfuscation",
			args: []string{"gh-octoscope", "--obfuscate"},
			expected: cmd.Config{
				Obfuscate: true,
				PageSize:  30,
			},
		},
		{
			name: "Multiple options",
			args: []string{"gh-octoscope", "--debug", "--from=2025-03-01", "--obfuscate"},
			expected: cmd.Config{
				Debug:     true,
				FromDate:  "2025-03-01",
				PageSize:  30,
				Obfuscate: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the command for this test
			rootCmd := initializeRootCmd()

			// Parse the command line arguments
			rootCmd.SetArgs(tc.args[1:]) // Skip program name

			// Execute the command in a way that doesn't actually run the command function
			// We're just testing the flag parsing
			rootCmd.SilenceErrors = true
			rootCmd.SilenceUsage = true
			err := rootCmd.ParseFlags(tc.args[1:])
			require.NoError(t, err)

			// Extract the Config from the command
			var cfg cmd.Config
			rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
				switch f.Name {
				case "debug":
					if f.Changed {
						val, _ := rootCmd.Flags().GetBool(f.Name)
						cfg.Debug = val
					}
				case "prod-log":
					if f.Changed {
						val, _ := rootCmd.Flags().GetBool(f.Name)
						cfg.ProdLogger = val
					}
				case "from":
					if f.Changed {
						val, _ := rootCmd.Flags().GetString(f.Name)
						cfg.FromDate = val
					}
				case "page-size":
					val, _ := rootCmd.Flags().GetInt(f.Name)
					cfg.PageSize = val
				case "obfuscate":
					if f.Changed {
						val, _ := rootCmd.Flags().GetBool(f.Name)
						cfg.Obfuscate = val
					}
				}
			})

			// Check results
			assert.Equal(t, tc.expected.Debug, cfg.Debug)
			assert.Equal(t, tc.expected.ProdLogger, cfg.ProdLogger)
			assert.Equal(t, tc.expected.FromDate, cfg.FromDate)
			assert.Equal(t, tc.expected.PageSize, cfg.PageSize)
			assert.Equal(t, tc.expected.Obfuscate, cfg.Obfuscate)
		})
	}
}

func TestProcessJobs(t *testing.T) {
	// Setup mock data
	repo := &github.Repository{
		Name:  github.String("testrepo"),
		Owner: &github.User{Login: github.String("testowner")},
	}

	wfl := &github.Workflow{
		ID:   github.Int64(1234),
		Name: github.String("Test Workflow"),
	}

	run := &github.WorkflowRun{
		ID:         github.Int64(5678),
		RunAttempt: github.Int(1),
	}

	conclusion := "success"
	now := time.Now()

	jobs := []*github.WorkflowJob{
		{
			ID:          github.Int64(9012),
			Name:        github.String("Job 1"),
			Status:      github.String("completed"),
			Conclusion:  &conclusion,
			CreatedAt:   &github.Timestamp{Time: now.Add(-30 * time.Minute)},
			CompletedAt: &github.Timestamp{Time: now.Add(-25 * time.Minute)},
		},
		{
			ID:          github.Int64(9013),
			Name:        github.String("Job 2"),
			Status:      github.String("completed"),
			Conclusion:  &conclusion,
			CreatedAt:   &github.Timestamp{Time: now.Add(-20 * time.Minute)},
			CompletedAt: &github.Timestamp{Time: now.Add(-10 * time.Minute)},
		},
	}

	jobRunnerMap := map[int]billing.RunnerDuration{
		9012: {Runner: "UBUNTU", Duration: github.Int64(300000)},  // 5 minutes
		9013: {Runner: "WINDOWS", Duration: github.Int64(600000)}, // 10 minutes
	}

	// Setup logger
	logger := zerolog.New(io.Discard)

	// Setup calculator
	calculator := billing.NewCalculator(nil, logger)

	// Initialize empty job details and total costs
	var jobDetails []reports.JobDetails
	totalCosts := reports.TotalCosts{}

	// Run the function under test - use the exported function from cmd package
	newJobDetails, newTotalCosts := cmd.ProcessJobs(jobDetails, totalCosts, repo, wfl, run, jobs, jobRunnerMap, calculator)

	// Check results
	assert.Len(t, newJobDetails, 2)
	assert.Equal(t, 5*time.Minute, newJobDetails[0].JobDuration)
	assert.Equal(t, 10*time.Minute, newJobDetails[1].JobDuration)

	// Check that runner types are set correctly
	assert.Equal(t, "UBUNTU", newJobDetails[0].Runner)
	assert.Equal(t, "WINDOWS", newJobDetails[1].Runner)

	// Check total costs
	assert.Equal(t, 15*time.Minute, newTotalCosts.JobDuration)

	// Check billable amounts
	// Ubuntu: 5min at $0.008/min = $0.04
	// Windows: 10min, at $0.016/min = $0.16
	// Total: $0.208
	assert.InDelta(t, 0.2, newTotalCosts.BillableInUSD, 0.001)
}

func TestRun_FetchMode(t *testing.T) {
	t.Skip() // Skip this test as it requires a live GitHub API connection
	// TODO: Implement a proper mocks
	// Create temporary directory for output files
	tmpDir, err := os.MkdirTemp("", "octoscope-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Change working directory to the temporary directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create reports directory
	err = os.Mkdir("reports", 0755)
	require.NoError(t, err)

	// Set up mock environment variables
	os.Setenv("OCTOSCOPE_API_URL", "https://api.notreal.url")
	os.Setenv("OCTOSCOPE_APP_URL", "https://app.notreal.url")
	defer os.Unsetenv("OCTOSCOPE_API_URL")
	defer os.Unsetenv("OCTOSCOPE_APP_URL")

	// Mock data
	repo := &github.Repository{
		Name:  github.String("testrepo"),
		Owner: &github.User{Login: github.String("testowner")},
	}

	workflows := &github.Workflows{
		TotalCount: github.Int(1),
		Workflows: []*github.Workflow{
			{
				ID:   github.Int64(1234),
				Name: github.String("Test Workflow"),
			},
		},
	}

	runs := &github.WorkflowRuns{
		TotalCount: github.Int(1),
		WorkflowRuns: []*github.WorkflowRun{
			{
				ID:         github.Int64(5678),
				WorkflowID: github.Int64(1234),
				Name:       github.String("Test Run"),
				RunNumber:  github.Int(1),
				RunAttempt: github.Int(1),
				Status:     github.String("completed"),
				Conclusion: github.String("success"),
				CreatedAt:  &github.Timestamp{Time: time.Now().Add(-1 * time.Hour)},
				UpdatedAt:  &github.Timestamp{Time: time.Now()},
			},
		},
	}

	conclusion := "success"
	now := time.Now()

	jobs := &github.Jobs{
		TotalCount: github.Int(1),
		Jobs: []*github.WorkflowJob{
			{
				ID:          github.Int64(9012),
				Name:        github.String("Job 1"),
				Status:      github.String("completed"),
				Conclusion:  &conclusion,
				CreatedAt:   &github.Timestamp{Time: now.Add(-30 * time.Minute)},
				CompletedAt: &github.Timestamp{Time: now.Add(-25 * time.Minute)},
				RunnerName:  github.String("ubuntu-latest"),
			},
		},
	}

	usage := &github.WorkflowRunUsage{
		Billable: &github.WorkflowRunBillMap{
			"UBUNTU": &github.WorkflowRunBill{
				TotalMS: github.Int64(300000), // 5 minutes
				Jobs:    github.Int(1),
			},
		},
		RunDurationMS: github.Int64(300000),
	}

	// Create mock clients
	mockGH := new(mockGitHubClient)
	mockGH.On("GetRepository", mock.Anything).Return(repo, nil)
	mockGH.On("ListWorkflows", mock.Anything).Return(workflows, nil)
	mockGH.On("ListRepositoryRuns", mock.Anything, mock.Anything).Return(runs, nil)
	mockGH.On("GetWorkflowRunUsage", mock.Anything, int64(5678)).Return(usage, nil)
	mockGH.On("ListWorkflowJobs", mock.Anything, int64(5678)).Return(jobs, nil)

	mockOS := new(mockOctoscopeClient)
	mockOS.On("BatchCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Store original factories
	origNewClient := newClient
	origNewOctoscopeClient := newOctoscopeClient

	// Replace with mock factories
	newClient = func(repo repository.Repository, cfg api.Config) api.Client {
		return mockGH
	}

	newOctoscopeClient = func(cfg api.OctoscopeConfig) api.OctoscopeClient {
		return mockOS
	}

	// Restore original factories after test
	defer func() {
		newClient = origNewClient
		newOctoscopeClient = origNewOctoscopeClient
	}()

	// GitHub CLI config mock
	ghCLIConfig := cmd.GitHubCLIConfig{
		Repo: repository.Repository{
			Owner: "testowner",
			Name:  "testrepo",
		},
		Token: "testtoken",
	}

	// Run the function with various configurations
	t.Run("CSV_Report", func(t *testing.T) {
		cfg := cmd.Config{
			CSVReport: true,
			PageSize:  10,
		}

		err := cmd.Run(cfg, ghCLIConfig, true)
		require.NoError(t, err)

		// Check that files were created
		assert.FileExists(t, filepath.Join("reports", "report.csv"))
		assert.FileExists(t, filepath.Join("reports", "totals.csv"))
	})

	t.Run("Full_Report", func(t *testing.T) {
		cfg := cmd.Config{
			FullReport: true,
			PageSize:   10,
		}

		err := cmd.Run(cfg, ghCLIConfig, true)
		require.NoError(t, err)

		// Check that server API was called
		mockOS.AssertCalled(t, "BatchCreate", mock.Anything, mock.Anything, mock.Anything, false)
	})

	t.Run("Obfuscated_Report", func(t *testing.T) {
		cfg := cmd.Config{
			FullReport: true,
			PageSize:   10,
			Obfuscate:  true,
		}

		err := cmd.Run(cfg, ghCLIConfig, true)
		require.NoError(t, err)

		// Check that server API was called with obfuscation
		mockOS.AssertCalled(t, "BatchCreate", mock.Anything, mock.Anything, mock.Anything, true)
	})
}

func TestRun_NoFetchMode(t *testing.T) {
	t.Skip() // Skip this test as it requires a live GitHub API connection
	// TODO: Implement a proper mocks
	// Create temporary directory for input and output files
	tmpDir, err := os.MkdirTemp("", "octoscope-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Change working directory to the temporary directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create data directory structure
	reportsDir := filepath.Join(tmpDir, "reports")
	dataDir := filepath.Join(reportsDir, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))

	// Create sample data files
	summaryData := `{"totals":{"job_duration":1500000000000,"rounded_up_job_duration":1800000000000,"billable_in_usd":0.24}}`
	jobsData := `[{"owner_name":"testowner","repo_id":"123456","repo_name":"testrepo","workflow_id":"1234","workflow_name":"Test Workflow","workflow_run_id":"5678","workflow_run_name":"Test Run","job_id":"9012","job_name":"Job 1","job_duration":"300","rounded_up_job_duration":"360","price_per_minute_in_usd":"0.008","billable_in_usd":"0.048","runner":"UBUNTU"}]`

	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "summary.json"), []byte(summaryData), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "jobs-1.json"), []byte(jobsData), 0644))

	// Set environment variables
	os.Setenv("OCTOSCOPE_API_URL", "https://api.example.com")
	os.Setenv("OCTOSCOPE_APP_URL", "https://app.example.com")
	defer os.Unsetenv("OCTOSCOPE_API_URL")
	defer os.Unsetenv("OCTOSCOPE_APP_URL")

	// Create mock octoscope client
	mockOS := new(mockOctoscopeClient)
	mockOS.On("BatchCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Store original factory
	origNewOctoscopeClient := newOctoscopeClient

	// Replace with mock factory
	newOctoscopeClient = func(cfg api.OctoscopeConfig) api.OctoscopeClient {
		return mockOS
	}

	// Restore original factory after test
	defer func() {
		newOctoscopeClient = origNewOctoscopeClient
	}()

	// GitHub CLI config mock
	ghCLIConfig := cmd.GitHubCLIConfig{
		Repo: repository.Repository{
			Owner: "testowner",
			Name:  "testrepo",
		},
		Token: "testtoken",
	}

	// Run the function with various configurations
	t.Run("CSV_Report", func(t *testing.T) {
		cfg := cmd.Config{
			CSVReport: true,
		}

		err := cmd.Run(cfg, ghCLIConfig, false)
		require.NoError(t, err)

		// Check that files were created
		assert.FileExists(t, filepath.Join("reports", "report.csv"))
		assert.FileExists(t, filepath.Join("reports", "totals.csv"))
	})

	t.Run("Full_Report", func(t *testing.T) {
		cfg := cmd.Config{
			FullReport: true,
		}

		err := cmd.Run(cfg, ghCLIConfig, false)
		require.NoError(t, err)

		// Check that server API was called
		mockOS.AssertCalled(t, "BatchCreate", mock.Anything, mock.Anything, mock.Anything, false)
	})
}

func TestReportCommand(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test report command with different flags
	tests := []struct {
		name        string
		args        []string
		expectedCSV bool
		expectFetch bool
	}{
		{
			name:        "Report command default",
			args:        []string{"gh-octoscope", "report"},
			expectedCSV: false,
			expectFetch: true,
		},
		{
			name:        "Report with CSV flag",
			args:        []string{"gh-octoscope", "report", "--csv"},
			expectedCSV: true,
			expectFetch: true,
		},
		{
			name:        "Report with fetch=false",
			args:        []string{"gh-octoscope", "report", "--fetch=false"},
			expectedCSV: false,

			expectFetch: false,
		},
		{
			name:        "Report with multiple flags",
			args:        []string{"gh-octoscope", "report", "--csv", "", "--fetch=false"},
			expectedCSV: true,

			expectFetch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set args for this test
			os.Args = tc.args

			// Create a new command for this test
			cmd := initializeRootCmd()

			// Get the report command
			reportCmd, _, err := cmd.Find([]string{"report"})
			require.NoError(t, err)
			require.NotNil(t, reportCmd)

			// Parse flags
			err = reportCmd.ParseFlags(tc.args[2:])
			require.NoError(t, err)

			// Check flag values
			csvFlag, _ := reportCmd.Flags().GetBool("csv")
			fetchFlag, _ := reportCmd.Flags().GetBool("fetch")

			assert.Equal(t, tc.expectedCSV, csvFlag)
			assert.Equal(t, tc.expectFetch, fetchFlag)
		})
	}
}

func TestReportDeleteCommand(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Mock data
	testReportID := "abcd1234-5678-efgh-9012"

	// Create mock clients
	mockOS := new(mockOctoscopeClient)
	mockOS.On("DeleteReport", mock.Anything, testReportID).Return(nil)

	// Store original factory
	origNewOctoscopeClient := newOctoscopeClient

	// Replace with mock factory
	newOctoscopeClient = func(cfg api.OctoscopeConfig) api.OctoscopeClient {
		return mockOS
	}

	// Restore original factory after test
	defer func() {
		newOctoscopeClient = origNewOctoscopeClient
	}()

	// Test for correct args validation
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "Delete with report ID",
			args:          []string{"gh-octoscope", "report", "delete", testReportID},
			expectedError: "",
		},
		{
			name:          "Delete without report ID",
			args:          []string{"gh-octoscope", "report", "delete"},
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name:          "Delete with too many args",
			args:          []string{"gh-octoscope", "report", "delete", testReportID, "extra-arg"},
			expectedError: "accepts 1 arg(s), received 2",
		},
	}

	// Temporarily redirect stdout/stderr to suppress cobra output
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = nil
	os.Stderr = nil
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set args for this test
			os.Args = tc.args

			// Create a command for testing
			rootCmd := initializeRootCmd()

			// Get the delete command
			var deleteCmd *cobra.Command
			var err error

			reportCmd, _, _ := rootCmd.Find([]string{"report"})
			require.NotNil(t, reportCmd)

			if len(tc.args) > 2 {
				deleteCmd, _, err = reportCmd.Find([]string{"delete"})
				require.NoError(t, err)
				require.NotNil(t, deleteCmd)
			}

			// Execute the command to test arg validation
			if tc.expectedError != "" {
				// Set the command arguments excluding the command name
				reportCmd.SetArgs(tc.args[2:])
				err = reportCmd.Execute()
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				// For successful case, just verify we can find the command and its structure
				assert.NotNil(t, deleteCmd)
				assert.Equal(t, "delete", deleteCmd.Name())
				// Verify the command exists and matches our expectations
				assert.NotNil(t, deleteCmd.Args)
			}
		})
	}

	// Test the actual functionality with valid arguments
	t.Run("Delete report functionality", func(t *testing.T) {
		t.Skip() // Skip to avoid calling the real API

		// This would be how we'd execute and verify, but we're skipping for now
		// Create a temporary directory for output
		// tmpDir, err := os.MkdirTemp("", "octoscope-test")
		// require.NoError(t, err)
		// defer os.RemoveAll(tmpDir)
		/*
			// Setup env for test
			os.Setenv("OCTOSCOPE_API_URL", "https://api.example.com")
			defer os.Unsetenv("OCTOSCOPE_API_URL")

			// Execute delete command
			rootCmd := initializeRootCmd()
			rootCmd.SetArgs([]string{"report", "delete", testReportID})
			err = rootCmd.Execute()
			require.NoError(t, err)

			// Verify the client was called properly
			mockOS.AssertCalled(t, "DeleteReport", mock.Anything, testReportID)
		*/
	})
}

// Helper variables and functions for mocking
var (
	newClient = func(repo repository.Repository, cfg api.Config) api.Client {
		// This will be replaced with a mock in tests
		return nil
	}

	newOctoscopeClient = func(cfg api.OctoscopeConfig) api.OctoscopeClient {
		// This will be replaced with a mock in tests
		return nil
	}
)

// initializeRootCmd creates a fresh root command for testing
func initializeRootCmd() *cobra.Command {
	return cmd.NewRootCmd()
}
