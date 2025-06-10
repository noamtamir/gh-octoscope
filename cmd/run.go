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
		// Start spinner for loading existing data
		s := createSpinner("Loading existing data...")
		s.Start()

		var err error
		jobDetails, totalCosts, err = loadExistingData()

		// Stop spinner and show success or error message
		s.Stop()
		if err != nil {
			return err
		}
		fmt.Println(createSuccessMessage("Data loaded successfully."))
	}

	if err := os.MkdirAll(reportsDirName, 0755); err != nil {
		return err
	}

	// Start spinner for report generation
	s := createSpinner("Generating reports...")
	s.Start()

	err := generateReports(cfg, ghCLIConfig, jobDetails, totalCosts, logger)

	// Stop spinner and show message
	s.Stop()
	if err != nil {
		return err
	}
	fmt.Println(createSuccessMessage("Report generation completed."))

	logger.Debug().
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
	s := createSpinner("Fetching GitHub Actions data...")
	s.Start()
	repoDetails, err := ghClient.GetRepository(ctx)
	if err != nil {
		return nil, totalCosts, err
	}

	// Fetch runs with all their jobs and data concurrently
	runsWithJobs, err := ghClient.(api.ThrottledClient).FetchRunsWithJobs(ctx, fromDate)
	if err != nil {
		return nil, totalCosts, err
	}
	fmt.Println(createSuccessMessage("Data fetching completed!"))
	// Process the fetched runs and jobs
	s = createSpinner("Processing data...")
	s.Start()

	for _, runWithJobs := range runsWithJobs {
		run := runWithJobs.Run
		workflow := runWithJobs.Workflow

		// Process main jobs
		jobDetails, totalCosts = ProcessJobs(jobDetails, totalCosts, repoDetails, workflow, run, runWithJobs.Jobs, calculator)

		// Process jobs from previous attempts
		for _, attemptJobs := range runWithJobs.AttemptJobs {
			jobDetails, totalCosts = ProcessJobs(jobDetails, totalCosts, repoDetails, workflow, run, attemptJobs, calculator)
		}
	}

	s.Stop()
	fmt.Println(createSuccessMessage(fmt.Sprintf("Successfully processed data!")))

	// Save the data for future use without fetching again
	s = createSpinner("Saving data for future use...")
	s.Start()
	err = saveData(jobDetails, totalCosts)
	s.Stop()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to save data for future use")
	} else {
		fmt.Println(createSuccessMessage("Data successfully saved for future use!"))
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

	s := createSpinner("Checking for existing data...")
	s.Start()

	dataDir := reportsDirName + "/data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		s.Stop()
		return nil, totalCosts, fmt.Errorf("data directory %s does not exist. Run 'gh octoscope fetch' first", dataDir)
	}
	s.Stop()
	fmt.Println(createInfoMessage("Found existing data directory."))

	s = createSpinner("Loading data files...")
	s.Start()

	summaryFile, err := os.ReadFile(filepath.Join(dataDir, "summary.json"))
	if err != nil {
		s.Stop()
		return nil, totalCosts, fmt.Errorf("failed to read summary.json: %w", err)
	}

	var summary struct {
		Totals reports.TotalCosts `json:"totals"`
	}
	if err := json.Unmarshal(summaryFile, &summary); err != nil {
		s.Stop()
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
			s.Stop()
			return nil, totalCosts, fmt.Errorf("failed to read %s: %w", jobsPath, err)
		}

		var chunk []reports.JobDetails
		if err := json.Unmarshal(jobsFile, &chunk); err != nil {
			s.Stop()
			return nil, totalCosts, fmt.Errorf("failed to parse %s: %w", jobsPath, err)
		}
		jobDetails = append(jobDetails, chunk...)
	}

	s.Stop()

	if len(jobDetails) == 0 {
		return nil, totalCosts, fmt.Errorf("no job data found in %s", dataDir)
	}

	fmt.Println(createSuccessMessage(fmt.Sprintf("Successfully loaded %d jobs from existing data.", len(jobDetails))))
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
		// Start spinner for CSV report generation
		s := createSpinner("Generating CSV reports...")
		s.Start()

		csvGen := reports.NewCSVGeneratorWithFormat(
			reportsDirName,
			ghCLIConfig.Repo.Owner,
			ghCLIConfig.Repo.Name,
			reportID,
			logger,
		)
		err := csvGen.Generate(reportData)

		// Stop spinner
		s.Stop()
		if err != nil {
			return err
		}
		fmt.Println(createSuccessMessage("CSV reports generated."))

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

		// Make the file paths clickable using the OSC 8 ANSI escape sequence
		// Format: \033]8;;file:///path/to/file\033\\file path\033]8;;\033\\
		jobsPathLink := fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", jobsPath, jobsPath)
		totalsPathLink := fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", totalsPath, totalsPath)

		fmt.Printf("\nCSV Report: %s", jobsPathLink)
		fmt.Printf("\nCSV Totals: %s\n\n", totalsPathLink)
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

		// Start spinner for server report generation
		s := createSpinner("Generating full report on server...")
		s.Start()

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

		// Stop spinner
		s.Stop()
		if err != nil {
			return fmt.Errorf("failed to generate server report: %w", err)
		}
		fmt.Println(createSuccessMessage("Full report generated successfully on server."))

		reportURL := serverGen.GetReportURL()
		fmt.Printf("\nReport URL: %s\n\n", reportURL)
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
	calculator *billing.Calculator,
) ([]reports.JobDetails, reports.TotalCosts) {
	for _, job := range jobs {
		// Calculate job costs based on labels
		cost, runnerType, err := calculator.CalculateJobCost(job)
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
			Runner:               string(runnerType),
		})

		totalCosts.JobDuration += cost.ActualDuration
		totalCosts.RoundedUpJobDuration += cost.BillableDuration
		totalCosts.BillableInUSD += cost.TotalBillableUSD
	}

	return jobDetails, totalCosts
}
