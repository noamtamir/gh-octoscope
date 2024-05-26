package main

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
)

type FlattenedJob struct {
	OwnerName               string `json:"owner_name,omitempty"`
	RepoID                  string `json:"repo_id,omitempty"`
	RepoName                string `json:"repo_name,omitempty"`
	WorkflowID              string `json:"workflow_id,omitempty"`
	WorkflowName            string `json:"workflow_name,omitempty"`
	WorkflowRunID           string `json:"workflow_run_id,omitempty"`
	WorkflowRunName         string `json:"workflow_run_name,omitempty"`
	HeadBranch              string `json:"head_branch,omitempty"`
	HeadSHA                 string `json:"head_sha,omitempty"`
	WorkflowRunRunNumber    string `json:"workflow_run_run_number,omitempty"`
	WorkflowRunRunAttempt   string `json:"workflow_run_run_attempt,omitempty"`
	WorkflowRunEvent        string `json:"workflow_run_event,omitempty"`
	WorkflowRunDisplayTitle string `json:"workflow_run_display_title,omitempty"`
	WorkflowRunStatus       string `json:"workflow_run_status,omitempty"`
	WorkflowRunConclusion   string `json:"workflow_run_conclusion,omitempty"`
	WorkflowRunCreatedAt    string `json:"workflow_run_created_at,omitempty"`
	WorkflowRunUpdatedAt    string `json:"workflow_run_updated_at,omitempty"`
	WorkflowRunRunStartedAt string `json:"workflow_run_run_started_at,omitempty"`
	ActorLogin              string `json:"actor_login,omitempty"`
	JobID                   string `json:"job_id,omitempty"`
	JobName                 string `json:"job_name,omitempty"`
	JobStatus               string `json:"job_status,omitempty"`
	JobConclusion           string `json:"job_conclusion,omitempty"`
	JobCreatedAt            string `json:"job_created_at,omitempty"`
	JobStartedAt            string `json:"job_started_at,omitempty"`
	JobCompletedAt          string `json:"job_completed_at,omitempty"`
	JobSteps                string `json:"job_steps,omitempty"`
	JobLabels               string `json:"job_labels,omitempty"`
	JobRunnerID             string `json:"job_runner_id,omitempty"`
	JobRunnerName           string `json:"job_runner_name,omitempty"`
	JobRunnerGroupID        string `json:"job_runner_group_id,omitempty"`
	JobRunnerGroupName      string `json:"job_runner_group_name,omitempty"`
	JobRunAttempt           string `json:"job_run_attempt,omitempty"`
	JobDuration             string `json:"job_duration,omitempty"`
	RoundedUpJobDuration    string `json:"rounded_up_job_duration,omitempty"`
	PricePerMinuteInUSD     string `json:"price_per_minute_in_usd,omitempty"`
	BillableInUSD           string `json:"billable_in_usd,omitempty"`
	Runner                  string `json:"runner,omitempty"`
}

func (fj *FlattenedJob) toCsv() []string {
	v := reflect.ValueOf(*fj)
	n := v.NumField()

	values := make([]string, n)

	for i := 0; i < n; i++ {
		val := v.Field(i).Interface()
		if val, ok := val.(string); ok {
			values[i] = val
			continue
		}
		panic("Not a string! yikes!")
	}

	return values
}

type JobDetails struct {
	Repo                 *github.Repository  `json:"repo,omitempty"`
	Workflow             *github.Workflow    `json:"workflow,omitempty"`
	WorkflowRun          *github.WorkflowRun `json:"workflow_run,omitempty"`
	Job                  *github.WorkflowJob `json:"job,omitempty"`
	JobDuration          time.Duration       `json:"job_duration,omitempty"`
	RoundedUpJobDuration time.Duration       `json:"rounded_up_job_duration,omitempty"`
	PricePerMinuteInUSD  float64             `json:"price_per_minute_in_usd,omitempty"`
	BillableInUSD        float64             `json:"billable_in_usd,omitempty"`
	Runner               string              `json:"runner,omitempty"`
}

func (j *JobDetails) flatten() FlattenedJob {
	stepsBytes, err := json.Marshal(j.Job.Steps)
	checkErr(err)
	steps := string(stepsBytes)

	return FlattenedJob{
		OwnerName:               *j.Repo.Owner.Login,
		RepoID:                  strconv.FormatInt(*j.Repo.ID, 10),
		RepoName:                *j.Repo.Name,
		WorkflowID:              strconv.FormatInt(*j.Workflow.ID, 10),
		WorkflowName:            *j.Workflow.Name,
		WorkflowRunID:           strconv.FormatInt(*j.WorkflowRun.ID, 10),
		WorkflowRunName:         *j.WorkflowRun.Name,
		HeadBranch:              *j.WorkflowRun.HeadBranch,
		HeadSHA:                 *j.WorkflowRun.HeadSHA,
		WorkflowRunRunNumber:    strconv.Itoa(*j.WorkflowRun.RunNumber),
		WorkflowRunRunAttempt:   strconv.Itoa(*j.WorkflowRun.RunAttempt),
		WorkflowRunEvent:        *j.WorkflowRun.Event,
		WorkflowRunDisplayTitle: *j.WorkflowRun.DisplayTitle,
		WorkflowRunStatus:       *j.WorkflowRun.Status,
		WorkflowRunConclusion:   *j.WorkflowRun.Conclusion,
		WorkflowRunCreatedAt:    j.WorkflowRun.CreatedAt.String(),
		WorkflowRunUpdatedAt:    j.WorkflowRun.UpdatedAt.String(),
		WorkflowRunRunStartedAt: j.WorkflowRun.RunStartedAt.String(),
		ActorLogin:              *j.WorkflowRun.Actor.Login,
		JobID:                   strconv.FormatInt(*j.Job.ID, 10),
		JobName:                 *j.Job.Name,
		JobStatus:               *j.Job.Status,
		JobConclusion:           *j.Job.Conclusion,
		JobCreatedAt:            j.Job.CreatedAt.String(),
		JobStartedAt:            j.Job.StartedAt.String(),
		JobCompletedAt:          j.Job.CompletedAt.String(),
		JobSteps:                steps,
		JobLabels:               strings.Join(j.Job.Labels, "; "),
		JobRunnerID:             strconv.FormatInt(*j.Job.RunnerID, 10),
		JobRunnerName:           *j.Job.RunnerName,
		JobRunnerGroupID:        strconv.FormatInt(*j.Job.RunnerGroupID, 10),
		JobRunnerGroupName:      *j.Job.RunnerGroupName,
		JobRunAttempt:           strconv.FormatInt(*j.Job.RunAttempt, 10),
		JobDuration:             j.JobDuration.String(),
		RoundedUpJobDuration:    j.RoundedUpJobDuration.String(),
		PricePerMinuteInUSD:     strconv.FormatFloat(j.PricePerMinuteInUSD, 'f', 3, 64),
		BillableInUSD:           strconv.FormatFloat(j.BillableInUSD, 'f', 3, 64),
		Runner:                  j.Runner,
	}
}

func flattenJobs(jobs []JobDetails) []FlattenedJob {
	var flattened []FlattenedJob
	for _, job := range jobs {
		flattened = append(flattened, job.flatten())
	}
	return flattened
}

type Totals struct {
	JobDuration          time.Duration `json:"total_job_duration,omitempty"`
	RoundedUpJobDuration time.Duration `json:"total_rounded_up_job_duration,omitempty"`
	BillableInUSD        float64       `json:"total_billable_in_usd,omitempty"`
}

func (t *Totals) toTotalsString() TotalsString {
	return TotalsString{
		JobDuration:          t.JobDuration.String(),
		RoundedUpJobDuration: t.RoundedUpJobDuration.String(),
		BillableInUSD:        strconv.FormatFloat(t.BillableInUSD, 'f', 3, 64),
	}
}

type TotalsString struct {
	JobDuration          string `json:"total_job_duration,omitempty"`
	RoundedUpJobDuration string `json:"total_rounded_up_job_duration,omitempty"`
	BillableInUSD        string `json:"total_billable_in_usd,omitempty"`
}

func (ts *TotalsString) toCsv() []string {
	v := reflect.ValueOf(*ts)
	n := v.NumField()

	values := make([]string, n)

	for i := 0; i < n; i++ {
		val := v.Field(i).Interface()
		if val, ok := val.(string); ok {
			values[i] = val
			continue
		}
		panic("Not a string! yikes!")
	}

	return values
}

func appendJobsDetails(
	jobsDetails []JobDetails,
	totals Totals,
	repoDetails *github.Repository,
	wfl *github.Workflow,
	run *github.WorkflowRun,
	jobs []*github.WorkflowJob,
	jobRunnerMap map[int]RunnerDuration,
) ([]JobDetails, Totals) {
	for _, job := range jobs {
		runnerDuration, exists := jobRunnerMap[int(*job.ID)]
		if !exists {
			logger.Error().Stack().Msgf("JobID %d was not present in usage data", *run.WorkflowID)
			continue
		}
		duration, rounded, pricePerMinute, billable := CalculateBillablePrice(job, runnerDuration.runner)
		if *runnerDuration.duration == 0 && *job.Conclusion == "cancelled" {
			// todo: find more edge cases and generalize
			duration, rounded, billable = 0, 0, 0
		}
		jobsDetails = append(jobsDetails, JobDetails{
			Repo:                 repoDetails,
			Workflow:             wfl,
			WorkflowRun:          run,
			Job:                  job,
			JobDuration:          duration,
			RoundedUpJobDuration: rounded,
			PricePerMinuteInUSD:  pricePerMinute,
			BillableInUSD:        billable,
			Runner:               runnerDuration.runner,
		})
		totals.JobDuration += duration
		totals.RoundedUpJobDuration += rounded
		totals.BillableInUSD += billable
	}
	return jobsDetails, totals
}
