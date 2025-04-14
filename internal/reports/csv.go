package reports

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func (g *CSVGenerator) generateJobsReport(jobs []JobDetails) error {
	if len(jobs) == 0 {
		g.logger.Info().Msg("No runs in the requested time frame")
		return nil
	}

	flattened := g.flattenJobs(jobs)
	data := g.prepareCSVData(flattened)

	return g.writeCSVFile(g.jobsPath, data)
}

func (g *CSVGenerator) generateTotalsReport(totals TotalCosts) error {
	headers := []string{"total_job_duration", "total_rounded_up_job_duration", "total_billable_in_usd"}
	data := [][]string{
		headers,
		{
			totals.JobDuration.String(),
			totals.RoundedUpJobDuration.String(),
			strconv.FormatFloat(totals.BillableInUSD, 'f', 3, 64),
		},
	}

	return g.writeCSVFile(g.totalsPath, data)
}

func (g *CSVGenerator) writeCSVFile(path string, data [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	for _, record := range data {
		if err := w.Write(record); err != nil {
			return err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	g.logger.Info().Msgf("%s created successfully!", path)
	return nil
}

func (g *CSVGenerator) flattenJobs(jobs []JobDetails) []flatJobDetails {
	var flattened []flatJobDetails
	for _, job := range jobs {
		flattened = append(flattened, g.flattenJob(job))
	}
	return flattened
}

func (g *CSVGenerator) flattenJob(job JobDetails) flatJobDetails {
	stepsBytes, _ := json.Marshal(job.Job.Steps)
	steps := string(stepsBytes)

	return flatJobDetails{
		OwnerName:                   *job.Repo.Owner.Login,
		RepoID:                      strconv.FormatInt(*job.Repo.ID, 10),
		RepoName:                    *job.Repo.Name,
		WorkflowID:                  strconv.FormatInt(*job.Workflow.ID, 10),
		WorkflowName:                *job.Workflow.Name,
		WorkflowRunID:               strconv.FormatInt(*job.WorkflowRun.ID, 10),
		WorkflowRunName:             *job.WorkflowRun.Name,
		HeadBranch:                  *job.WorkflowRun.HeadBranch,
		HeadSHA:                     *job.WorkflowRun.HeadSHA,
		WorkflowRunRunNumber:        strconv.Itoa(*job.WorkflowRun.RunNumber),
		WorkflowRunRunAttempt:       strconv.Itoa(*job.WorkflowRun.RunAttempt),
		WorkflowRunEvent:            *job.WorkflowRun.Event,
		WorkflowRunDisplayTitle:     *job.WorkflowRun.DisplayTitle,
		WorkflowRunStatus:           *job.WorkflowRun.Status,
		WorkflowRunConclusion:       *job.WorkflowRun.Conclusion,
		WorkflowRunCreatedAt:        job.WorkflowRun.CreatedAt.String(),
		WorkflowRunUpdatedAt:        job.WorkflowRun.UpdatedAt.String(),
		WorkflowRunRunStartedAt:     job.WorkflowRun.RunStartedAt.String(),
		ActorLogin:                  *job.WorkflowRun.Actor.Login,
		JobID:                       strconv.FormatInt(*job.Job.ID, 10),
		JobName:                     *job.Job.Name,
		JobStatus:                   *job.Job.Status,
		JobConclusion:               *job.Job.Conclusion,
		JobCreatedAt:                job.Job.CreatedAt.String(),
		JobStartedAt:                job.Job.StartedAt.String(),
		JobCompletedAt:              job.Job.CompletedAt.String(),
		JobSteps:                    steps,
		JobLabels:                   strings.Join(job.Job.Labels, "; "),
		JobRunnerID:                 strconv.FormatInt(*job.Job.RunnerID, 10),
		JobRunnerName:               *job.Job.RunnerName,
		JobRunnerGroupID:            strconv.FormatInt(*job.Job.RunnerGroupID, 10),
		JobRunnerGroupName:          *job.Job.RunnerGroupName,
		JobRunAttempt:               strconv.FormatInt(*job.Job.RunAttempt, 10),
		JobDurationSeconds:          strconv.FormatFloat(job.JobDuration.Seconds(), 'f', 0, 64),
		RoundedUpJobDurationSeconds: strconv.FormatFloat(job.RoundedUpJobDuration.Seconds(), 'f', 0, 64),
		PricePerMinuteInUSD:         strconv.FormatFloat(job.PricePerMinuteInUSD, 'f', 3, 64),
		BillableInUSD:               strconv.FormatFloat(job.BillableInUSD, 'f', 3, 64),
		Runner:                      job.Runner,
	}
}

func (g *CSVGenerator) prepareCSVData(flattened []flatJobDetails) [][]string {
	var data [][]string

	// Add headers
	t := reflect.TypeOf(flattened[0])
	headers := make([]string, t.NumField())
	for i := range headers {
		headers[i] = t.Field(i).Name
	}
	data = append(data, headers)

	// Add data rows
	for _, fj := range flattened {
		row := g.structToStringSlice(fj)
		data = append(data, row)
	}

	return data
}

func (g *CSVGenerator) structToStringSlice(fj flatJobDetails) []string {
	v := reflect.ValueOf(fj)
	n := v.NumField()
	values := make([]string, n)

	for i := 0; i < n; i++ {
		val := v.Field(i).Interface()
		values[i] = val.(string)
	}

	return values
}
