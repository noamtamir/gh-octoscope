package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
	"github.com/joho/godotenv"
	"github.com/noamtamir/gh-octoscope/internal/api"
	"github.com/noamtamir/gh-octoscope/internal/billing"
	"github.com/noamtamir/gh-octoscope/internal/reports"
	"github.com/rs/zerolog"
)

type Config struct {
	Debug      bool
	ProdLogger bool
	FullReport bool
	CSVReport  bool
	HTMLReport bool
	FromDate   string
	PageSize   int
	Fetch      bool
	Obfuscate  bool
}

type GitHubCLIConfig struct {
	token string
	repo  repository.Repository
}

func parseFlags() Config {
	cfg := Config{
		PageSize: 30,
		Fetch:    true,
	}

	flag.BoolVar(&cfg.FullReport, "report", false, "Generate full report")
	flag.BoolVar(&cfg.Debug, "debug", false, "sets log level to debug")
	flag.BoolVar(&cfg.ProdLogger, "prod-log", false, "Production structured log")
	flag.BoolVar(&cfg.CSVReport, "csv", false, "Generate csv report")
	flag.BoolVar(&cfg.HTMLReport, "html", false, "Generate html report")
	flag.BoolVar(&cfg.Fetch, "fetch", true, "Fetch new data (set to false to use existing data)")
	flag.StringVar(&cfg.FromDate, "from", "", "Generate report from this date. Format: YYYY-MM-DD")
	flag.BoolVar(&cfg.Obfuscate, "obfuscate", false, "Obfuscate sensitive data in the report")
	flag.Parse()

	return cfg
}

func setupLogger(cfg Config) zerolog.Logger {
	var writer io.Writer
	if cfg.ProdLogger {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		writer = os.Stdout
	} else {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	logger := zerolog.New(writer).With().Timestamp().Logger()
	if cfg.Debug {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	return logger
}

func run(cfg Config, ghCLIConfig GitHubCLIConfig) error {
	logger := setupLogger(cfg)

	if err := godotenv.Load(); err != nil {
		logger.Debug().Msg(".env file not found, excepted when not running in development")
	}

	var jobDetails []reports.JobDetails
	var totalCosts reports.TotalCosts

	if cfg.Fetch {
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

	if err := os.MkdirAll("reports", 0755); err != nil {
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

	ghClient := api.NewClient(ghCLIConfig.repo, api.Config{
		PageSize: cfg.PageSize,
		Logger:   logger,
		Token:    ghCLIConfig.token,
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

	repoDetails, err := ghClient.GetRepository(ctx)
	if err != nil {
		return nil, totalCosts, err
	}

	workflows, err := ghClient.ListWorkflows(ctx)
	if err != nil {
		return nil, totalCosts, err
	}

	workflowMap := make(map[int64]*github.Workflow)
	for _, wfl := range workflows.Workflows {
		workflowMap[*wfl.ID] = wfl
	}

	runs, err := ghClient.ListRepositoryRuns(ctx, fromDate)
	if err != nil {
		return nil, totalCosts, err
	}

	if *runs.TotalCount > 0 {
		for _, run := range runs.WorkflowRuns {
			workflowRunUsage, err := ghClient.GetWorkflowRunUsage(ctx, *run.ID)
			if err != nil {
				return nil, totalCosts, err
			}

			jobRunnerMap := make(map[int]billing.RunnerDuration)
			for runnerType, billable := range *workflowRunUsage.Billable {
				for _, job := range billable.JobRuns {
					jobRunnerMap[*job.JobID] = billing.RunnerDuration{
						Runner:   runnerType,
						Duration: job.DurationMS,
					}
				}
			}

			jobs, err := ghClient.ListWorkflowJobs(ctx, *run.ID)
			if err != nil {
				return nil, totalCosts, err
			}

			wfl, exists := workflowMap[*run.WorkflowID]
			if !exists {
				logger.Error().Int64("workflowID", *run.WorkflowID).Msg("workflow ID not found")
				continue
			}

			jobDetails, totalCosts = processJobs(jobDetails, totalCosts, repoDetails, wfl, run, jobs.Jobs, jobRunnerMap, calculator)

			if *run.RunAttempt > 1 {
				for i := 1; i < int(*run.RunAttempt); i++ {
					attemptJobs, err := ghClient.ListWorkflowJobsAttempt(ctx, *run.ID, int64(i))
					if err != nil {
						return nil, totalCosts, err
					}
					jobDetails, totalCosts = processJobs(jobDetails, totalCosts, repoDetails, wfl, run, attemptJobs.Jobs, jobRunnerMap, calculator)
				}
			}
		}
	}

	return jobDetails, totalCosts, nil
}

func loadExistingData() ([]reports.JobDetails, reports.TotalCosts, error) {
	var jobDetails []reports.JobDetails
	var totalCosts reports.TotalCosts

	dataDir := "reports/data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil, totalCosts, fmt.Errorf("data directory %s does not exist. Run with -fetch=true first", dataDir)
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

	if cfg.CSVReport {
		csvGen := reports.NewCSVGenerator("reports/report.csv", "reports/totals.csv", logger)
		if err := csvGen.Generate(reportData); err != nil {
			return err
		}
	}

	if cfg.HTMLReport {
		htmlGen, err := reports.NewHTMLGenerator("reports/report.html", logger)
		if err != nil {
			return err
		}
		if err := htmlGen.Generate(reportData); err != nil {
			return err
		}
	}

	if cfg.FullReport {
		apiBaseUrl := os.Getenv("OCTOSCOPE_API_URL")
		appBaseUrl := os.Getenv("OCTOSCOPE_APP_URL")

		if apiBaseUrl == "" || appBaseUrl == "" {
			return fmt.Errorf("OCTOSCOPE_API_URL and OCTOSCOPE_APP_URL environment variables must be set when using -report flag")
		}

		osClient := api.NewOctoscopeClient(api.OctoscopeConfig{
			BaseUrl:     apiBaseUrl,
			Logger:      logger,
			GitHubToken: ghCLIConfig.token,
		})

		serverGen := reports.NewServerGenerator(osClient, reports.ServerConfig{
			AppURL:    appBaseUrl,
			OwnerName: ghCLIConfig.repo.Owner,
			RepoName:  ghCLIConfig.repo.Name,
		}, logger)

		if err := serverGen.Generate(reportData); err != nil {
			return fmt.Errorf("failed to generate server report: %w", err)
		}
	}

	return nil
}

func processJobs(
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

func main() {
	host, _ := auth.DefaultHost()
	token, _ := auth.TokenForHost(host)
	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ghCLIConfig := GitHubCLIConfig{
		token: token,
		repo:  repo,
	}

	if err := run(parseFlags(), ghCLIConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
