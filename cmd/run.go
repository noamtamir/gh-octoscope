package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/google/uuid"
	"github.com/noamtamir/gh-octoscope/internal/api"
	"github.com/noamtamir/gh-octoscope/internal/billing"
	"github.com/noamtamir/gh-octoscope/internal/reports"
	"github.com/rs/zerolog"
)

const reportsDirName string = ".reports"

// Run executes the main application logic
func Run(cfg Config, ghCLIConfig GitHubCLIConfig, fetchMode bool) error {
	logger := setupLogger()

	var jobDetails []reports.JobDetails
	var totalCosts reports.TotalCosts

	if fetchMode {
		var err error
		jobDetails, totalCosts, err = fetchData(cfg, ghCLIConfig, logger)
		if err != nil {
			return err
		}
	} else {
		var err error
		jobDetails, totalCosts, err = loadExistingData()
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(reportsDirName, 0755); err != nil {
		return err
	}

	if err := generateReports(cfg, ghCLIConfig, jobDetails, totalCosts, logger); err != nil {
		return err
	}

	logger.Info().
		Str("total_duration", totalCosts.JobDuration.String()).
		Str("total_billable_duration", totalCosts.RoundedUpJobDuration.String()).
		Float64("total_billable_usd", totalCosts.BillableInUSD).
		Msg("Run completed")

	return nil
}

func fetchData(cfg Config, ghCLIConfig GitHubCLIConfig, logger zerolog.Logger) ([]reports.JobDetails, reports.TotalCosts, error) {
	var jobDetails []reports.JobDetails
	var totalCosts reports.TotalCosts

	// Create new throttled client with appropriate rate limits
	ghClient := api.NewThrottledClient(ghCLIConfig.Repo, api.ThrottledClientConfig{
		Config: api.Config{
			PageSize: cfg.PageSize,
			Logger:   logger,
			Token:    ghCLIConfig.Token,
		},
		MaxConcurrentRequests: 5,               // Concurrent API calls
		RequestsPerSecond:     5,               // 300 per minute (below GitHub's 5000/hour primary limit)
		Burst:                 8,               // Allow small bursts
		RetryLimit:            3,               // Retry failed requests up to 3 times
		RetryBackoff:          time.Second * 1, // Start with 1 second backoff
	})

	calculator := billing.NewCalculator(nil, logger)
	ctx := context.Background()

	fromDate := time.Now().AddDate(0, 0, -7) // default to last 7 days
	if cfg.FromDate != "" {
		var err error
		fromDate, err = time.Parse(time.DateOnly, cfg.FromDate)
		if err != nil {
			return nil, totalCosts, err
		}
	}

	// Get repository information
	repoDetails, err := ghClient.GetRepository(ctx)
	if err != nil {
		return nil, totalCosts, err
	}

	// Fetch runs with all their jobs and data concurrently
	runsWithJobs, err := ghClient.(api.ThrottledClient).FetchRunsWithJobs(ctx, fromDate)
	if err != nil {
		return nil, totalCosts, err
	}

	// Process the fetched runs and jobs
	jobRunnerMap := make(map[int]billing.RunnerDuration)
	for _, runWithJobs := range runsWithJobs {
		run := runWithJobs.Run
		workflow := runWithJobs.Workflow
		workflowRunUsage := runWithJobs.UsageData

		// Create job runner map from usage data
		if workflowRunUsage.Billable != nil {
			for runnerType, billable := range *workflowRunUsage.Billable {
				for _, job := range billable.JobRuns {
					jobRunnerMap[*job.JobID] = billing.RunnerDuration{
						Runner:   runnerType,
						Duration: job.DurationMS,
					}
				}
			}
		}

		// Process main jobs
		jobDetails, totalCosts = ProcessJobs(jobDetails, totalCosts, repoDetails, workflow, run, runWithJobs.Jobs, jobRunnerMap, calculator)

		// Process jobs from previous attempts
		for _, attemptJobs := range runWithJobs.AttemptJobs {
			jobDetails, totalCosts = ProcessJobs(jobDetails, totalCosts, repoDetails, workflow, run, attemptJobs, jobRunnerMap, calculator)
		}
	}

	// Save the data for future use without fetching again
	if err := saveData(jobDetails, totalCosts); err != nil {
		logger.Warn().Err(err).Msg("Failed to save data for future use")
	}

	return jobDetails, totalCosts, nil
}

// saveData saves the fetched data to disk
func saveData(jobDetails []reports.JobDetails, totalCosts reports.TotalCosts) error {
	// Create data directory if it doesn't exist
	dataDir := reportsDirName + "/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Save summary
	summary := struct {
		Totals reports.TotalCosts `json:"totals"`
	}{
		Totals: totalCosts,
	}
	summaryData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dataDir, "summary.json"), summaryData, 0644); err != nil {
		return err
	}

	// Split job details into chunks to avoid very large files
	const chunkSize = 100
	for i := 0; i < len(jobDetails); i += chunkSize {
		end := i + chunkSize
		if end > len(jobDetails) {
			end = len(jobDetails)
		}

		chunk := jobDetails[i:end]
		chunkData, err := json.MarshalIndent(chunk, "", "  ")
		if err != nil {
			return err
		}

		chunkFile := filepath.Join(dataDir, fmt.Sprintf("jobs-%d.json", i/chunkSize+1))
		if err := os.WriteFile(chunkFile, chunkData, 0644); err != nil {
			return err
		}
	}

	return nil
}

func loadExistingData() ([]reports.JobDetails, reports.TotalCosts, error) {
	var jobDetails []reports.JobDetails
	var totalCosts reports.TotalCosts

	dataDir := reportsDirName + "/data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil, totalCosts, fmt.Errorf("data directory %s does not exist. Run 'gh octoscope fetch' first", dataDir)
	}

	summaryFile, err := os.ReadFile(filepath.Join(dataDir, "summary.json"))
	if err != nil {
		return nil, totalCosts, fmt.Errorf("failed to read summary.json: %w", err)
	}

	var summary struct {
		Totals reports.TotalCosts `json:"totals"`
	}
	if err := json.Unmarshal(summaryFile, &summary); err != nil {
		return nil, totalCosts, fmt.Errorf("failed to parse summary.json: %w", err)
	}
	totalCosts = summary.Totals

	for i := 1; ; i++ {
		jobsPath := filepath.Join(dataDir, fmt.Sprintf("jobs-%d.json", i))
		jobsFile, err := os.ReadFile(jobsPath)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return nil, totalCosts, fmt.Errorf("failed to read %s: %w", jobsPath, err)
		}

		var chunk []reports.JobDetails
		if err := json.Unmarshal(jobsFile, &chunk); err != nil {
			return nil, totalCosts, fmt.Errorf("failed to parse %s: %w", jobsPath, err)
		}
		jobDetails = append(jobDetails, chunk...)
	}

	if len(jobDetails) == 0 {
		return nil, totalCosts, fmt.Errorf("no job data found in %s", dataDir)
	}

	return jobDetails, totalCosts, nil
}

func generateReports(cfg Config, ghCLIConfig GitHubCLIConfig, jobDetails []reports.JobDetails, totalCosts reports.TotalCosts, logger zerolog.Logger) error {
	reportData := &reports.ReportData{
		Jobs:          jobDetails,
		Totals:        totalCosts,
		ObfuscateData: cfg.Obfuscate,
	}

	// Generate a single reportID to be used for both CSV and full report if needed
	reportID := uuid.New().String()

	if cfg.CSVReport {
		csvGen := reports.NewCSVGeneratorWithFormat(
			reportsDirName,
			ghCLIConfig.Repo.Owner,
			ghCLIConfig.Repo.Name,
			reportID,
			logger,
		)
		if err := csvGen.Generate(reportData); err != nil {
			return err
		}

		// Get current working directory to create absolute paths
		cwd, err := os.Getwd()
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to get current working directory")
			cwd = "" // Use relative paths if we can't get the current directory
		}

		// Convert to absolute paths
		jobsPath := csvGen.GetJobsPath()
		totalsPath := csvGen.GetTotalsPath()

		if cwd != "" && !filepath.IsAbs(jobsPath) {
			jobsPath = filepath.Join(cwd, jobsPath)
			totalsPath = filepath.Join(cwd, totalsPath)
		}

		fmt.Printf("\nCSV reports generated successfully:\n")
		fmt.Printf("  Report file: %s\n", jobsPath)
		fmt.Printf("  Totals file: %s\n\n", totalsPath)
	}

	if cfg.FullReport {
		apiBaseUrl := os.Getenv("OCTOSCOPE_API_URL")
		appBaseUrl := os.Getenv("OCTOSCOPE_APP_URL")

		if apiBaseUrl == "" {
			apiBaseUrl = "https://octoscope-server-production.up.railway.app"
		}
		if appBaseUrl == "" {
			appBaseUrl = "https://octoscope.netlify.app"
		}

		osClient := api.NewOctoscopeClient(api.OctoscopeConfig{
			BaseUrl:     apiBaseUrl,
			Logger:      logger,
			GitHubToken: ghCLIConfig.Token,
		})

		serverGen := reports.NewServerGenerator(osClient, reports.ServerConfig{
			AppURL:    appBaseUrl,
			OwnerName: ghCLIConfig.Repo.Owner,
			RepoName:  ghCLIConfig.Repo.Name,
			ReportID:  reportID, // Pass the same reportID used for CSV
		}, logger)

		err := serverGen.Generate(reportData)
		if err != nil {
			return fmt.Errorf("failed to generate server report: %w", err)
		}

		// Print the report URL for easy access
		fmt.Printf("\nFull report generated successfully:\n")
		fmt.Printf("  Report URL: %s\n\n", serverGen.GetReportURL())
	}

	return nil
}

// ProcessJobs processes workflow jobs and calculates costs
// This function is exported for testing purposes
func ProcessJobs(
	jobDetails []reports.JobDetails,
	totalCosts reports.TotalCosts,
	repo *github.Repository,
	workflow *github.Workflow,
	run *github.WorkflowRun,
	jobs []*github.WorkflowJob,
	jobRunnerMap map[int]billing.RunnerDuration,
	calculator *billing.Calculator,
) ([]reports.JobDetails, reports.TotalCosts) {
	for _, job := range jobs {
		runnerDuration, exists := jobRunnerMap[int(*job.ID)]
		if !exists {
			continue
		}

		cost, err := calculator.CalculateJobCost(job, billing.RunnerType(runnerDuration.Runner))
		if err != nil {
			continue
		}

		jobDetails = append(jobDetails, reports.JobDetails{
			Repo:                 repo,
			Workflow:             workflow,
			WorkflowRun:          run,
			Job:                  job,
			JobDuration:          cost.ActualDuration,
			RoundedUpJobDuration: cost.BillableDuration,
			PricePerMinuteInUSD:  cost.PricePerMinute,
			BillableInUSD:        cost.TotalBillableUSD,
			Runner:               string(runnerDuration.Runner),
		})

		totalCosts.JobDuration += cost.ActualDuration
		totalCosts.RoundedUpJobDuration += cost.BillableDuration
		totalCosts.BillableInUSD += cost.TotalBillableUSD
	}

	return jobDetails, totalCosts
}
