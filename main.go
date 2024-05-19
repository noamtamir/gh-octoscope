package main

import (
	"flag"
	"time"

	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
)

var logger zerolog.Logger = CreateLogger()

func getData(repo repository.Repository, client *github.Client, from *string) ([]JobDetails, Totals, int64) {
	// Note: github suggest to query sequentially to avoid hitting rate limits

	logger.Info().Msg("Fetching data...")
	repoDetails := getRepoDetails(repo, client)

	var jobsDetails []JobDetails
	totals := Totals{}
	wfls := getWorkflows(repo, client)
	logger.Debug().Msg(formatJson(wfls))

	wflMap := make(map[int64]*github.Workflow)
	for _, wfl := range wfls.Workflows {
		wflMap[*wfl.ID] = wfl
	}

	runs := getRepositoryRuns(repo, client, *from)
	logger.Debug().Msg(formatJson(runs))

	var totalDuration int64 = 0

	if *runs.TotalCount > 0 {
		for _, run := range runs.WorkflowRuns {
			jobs := getJobs(repo, client, *run.ID)
			logger.Debug().Msg(formatJson(jobs))
			wfl, exists := wflMap[*run.WorkflowID]
			if !exists {
				logger.Fatal().Stack().Msgf("WorkflowID %d does not exist...", *run.WorkflowID)
			}
			jobsDetails, totals = appendJobsDetails(jobsDetails, totals, repoDetails, wfl, run, jobs.Jobs)
			if *run.RunAttempt > 1 {
				for i := 1; i < int(*run.RunAttempt); i++ {
					attemptJobs := getAttempts(repo, client, *run.ID, int64(i))
					logger.Debug().Msg(formatJson(attemptJobs))
					jobsDetails, totals = appendJobsDetails(jobsDetails, totals, repoDetails, wfl, run, attemptJobs.Jobs)
				}
			}

			workflowRunUsage := getRunDurationInMS(repo, client, *run.ID)
			totalDuration += *workflowRunUsage.RunDurationMS
		}
	}
	return jobsDetails, totals, totalDuration
}

func main() {
	// cli
	debug := flag.Bool("debug", false, "sets log level to debug")
	csvFile := flag.Bool("csv", false, "Generate csv report")
	htmlFile := flag.Bool("html", false, "Generate html report")
	from := flag.String("from", "", "Generate report from this date. Format: YYYY-MM-DD")
	flag.Parse()

	if *from != "" {
		_, err := time.Parse(time.DateOnly, *from)
		if err != nil {
			logger.Fatal().Stack().Err(err).Msg("-from flag is not in YYYY-MM-DD format")
		}
	}

	// logging
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// setup http client
	host, _ := auth.DefaultHost()
	token, _ := auth.TokenForHost(host)
	client := github.NewClient(nil).WithAuthToken(token)
	repo, err := repository.Current()
	checkErr(err)

	// get data
	jobsDetails, totals, fetchedDuration := getData(repo, client, from)
	logger.Info().Interface("totals", totals.toTotalsString()).Msg("")

	// sanity check
	// todo: raise this to 100%
	calculatedDuration := totals.JobDuration.Milliseconds()
	precision := float64(calculatedDuration) / float64(fetchedDuration) * 100
	logger.Debug().Msgf("Data is %.2f%% precise", precision)

	// generate report
	if *csvFile {
		generateCsvFile(jobsDetails)
		generateTotalsCsvFile(totals)
	}

	if *htmlFile {
		generateHtmlFile(jobsDetails)
	}
}
