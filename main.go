package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
	"github.com/noamtamir/gh-octoscope/internal/api"
	"github.com/noamtamir/gh-octoscope/internal/billing"
	"github.com/noamtamir/gh-octoscope/internal/reports"
	"github.com/rs/zerolog"
)

// Config holds the application configuration
type Config struct {
	Debug      bool
	ProdLogger bool
	CSVReport  bool
	HTMLReport bool
	FromDate   string
	PageSize   int
}

func parseFlags() Config {
	cfg := Config{
		PageSize: 30, // default page size
	}

	flag.BoolVar(&cfg.Debug, "debug", false, "sets log level to debug")
	flag.BoolVar(&cfg.ProdLogger, "prod-log", false, "Production structured log")
	flag.BoolVar(&cfg.CSVReport, "csv", false, "Generate csv report")
	flag.BoolVar(&cfg.HTMLReport, "html", false, "Generate html report")
	flag.StringVar(&cfg.FromDate, "from", "", "Generate report from this date. Format: YYYY-MM-DD")
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

func run(cfg Config) error {
	logger := setupLogger(cfg)

	// Setup GitHub client
	host, _ := auth.DefaultHost()
	token, _ := auth.TokenForHost(host)
	repo, err := repository.Current()
	if err != nil {
		return err
	}

	// Initialize components
	ghClient := api.NewClient(repo, api.Config{
		PageSize: cfg.PageSize,
		Logger:   logger,
		Token:    token,
	})

	calculator := billing.NewCalculator(nil, logger)

	// Process data
	ctx := context.Background()
	fromDate := time.Now().AddDate(0, 0, -7) // default to last 7 days
	if cfg.FromDate != "" {
		fromDate, err = time.Parse(time.DateOnly, cfg.FromDate)
		if err != nil {
			return err
		}
	}

	repoDetails, err := ghClient.GetRepository(ctx)
	if err != nil {
		return err
	}

	workflows, err := ghClient.ListWorkflows(ctx)
	if err != nil {
		return err
	}

	workflowMap := make(map[int64]*github.Workflow)
	for _, wfl := range workflows.Workflows {
		workflowMap[*wfl.ID] = wfl
	}

	runs, err := ghClient.ListRepositoryRuns(ctx, fromDate)
	if err != nil {
		return err
	}

	var jobDetails []reports.JobDetails
	var totalCosts reports.TotalCosts

	if *runs.TotalCount > 0 {
		for _, run := range runs.WorkflowRuns {
			workflowRunUsage, err := ghClient.GetWorkflowRunUsage(ctx, *run.ID)
			if err != nil {
				return err
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
				return err
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
						return err
					}
					jobDetails, totalCosts = processJobs(jobDetails, totalCosts, repoDetails, wfl, run, attemptJobs.Jobs, jobRunnerMap, calculator)
				}
			}
		}
	}

	// Create reports directory
	if err := os.MkdirAll("reports", 0755); err != nil {
		return err
	}

	// Generate reports
	if cfg.CSVReport {
		csvGen := reports.NewCSVGenerator("reports/report.csv", "reports/totals.csv", logger)
		if err := csvGen.Generate(&reports.ReportData{
			Jobs:   jobDetails,
			Totals: totalCosts,
		}); err != nil {
			return err
		}
	}

	if cfg.HTMLReport {
		htmlGen, err := reports.NewHTMLGenerator("reports/report.html", logger)
		if err != nil {
			return err
		}
		if err := htmlGen.Generate(&reports.ReportData{
			Jobs:   jobDetails,
			Totals: totalCosts,
		}); err != nil {
			return err
		}
	}

	logger.Info().
		Str("total_duration", totalCosts.JobDuration.String()).
		Str("total_billable_duration", totalCosts.RoundedUpJobDuration.String()).
		Float64("total_billable_usd", totalCosts.BillableInUSD).
		Msg("Run completed")

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
	if err := run(parseFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
