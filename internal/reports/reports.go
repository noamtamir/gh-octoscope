package reports

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
)

type Generator interface {
	Generate(data *ReportData) error
}

type ReportData struct {
	Jobs          []JobDetails `json:"jobs"`
	Totals        TotalCosts   `json:"totals"`
	ObfuscateData bool         `json:"-"`
}

type JobDetails struct {
	Repo                 *github.Repository  `json:"repo,omitempty"`
	Workflow             *github.Workflow    `json:"workflow,omitempty"`
	WorkflowRun          *github.WorkflowRun `json:"workflow_run,omitempty"`
	Job                  *github.WorkflowJob `json:"job,omitempty"`
	JobDuration          time.Duration       `json:"job_duration"`
	RoundedUpJobDuration time.Duration       `json:"rounded_up_job_duration"`
	PricePerMinuteInUSD  float64             `json:"price_per_minute_in_usd"`
	BillableInUSD        float64             `json:"billable_in_usd"`
	Runner               string              `json:"runner,omitempty"`
}

type TotalCosts struct {
	JobDuration          time.Duration `json:"job_duration"`
	RoundedUpJobDuration time.Duration `json:"rounded_up_job_duration"`
	BillableInUSD        float64       `json:"billable_in_usd"`
}

type FlatJobDetails struct {
	OwnerName                         string  `json:"owner_name,omitempty"`
	RepoID                            int64   `json:"repo_id,omitempty"`
	RepoName                          string  `json:"repo_name,omitempty"`
	WorkflowID                        int64   `json:"workflow_id,omitempty"`
	WorkflowName                      string  `json:"workflow_name,omitempty"`
	WorkflowRunID                     int64   `json:"workflow_run_id,omitempty"`
	WorkflowRunName                   string  `json:"workflow_run_name,omitempty"`
	HeadBranch                        string  `json:"head_branch,omitempty"`
	HeadSHA                           string  `json:"head_sha,omitempty"`
	WorkflowRunRunNumber              int     `json:"workflow_run_run_number,omitempty"`
	WorkflowRunRunAttempt             int     `json:"workflow_run_run_attempt,omitempty"`
	WorkflowRunEvent                  string  `json:"workflow_run_event,omitempty"`
	WorkflowRunDisplayTitle           string  `json:"workflow_run_display_title,omitempty"`
	WorkflowRunStatus                 string  `json:"workflow_run_status,omitempty"`
	WorkflowRunConclusion             string  `json:"workflow_run_conclusion,omitempty"`
	WorkflowRunCreatedAt              string  `json:"workflow_run_created_at,omitempty"`
	WorkflowRunUpdatedAt              string  `json:"workflow_run_updated_at,omitempty"`
	WorkflowRunRunStartedAt           string  `json:"workflow_run_run_started_at,omitempty"`
	ActorLogin                        string  `json:"actor_login,omitempty"`
	JobID                             int64   `json:"job_id,omitempty"`
	JobName                           string  `json:"job_name,omitempty"`
	JobStatus                         string  `json:"job_status,omitempty"`
	JobConclusion                     string  `json:"job_conclusion,omitempty"`
	JobCreatedAt                      string  `json:"job_created_at,omitempty"`
	JobStartedAt                      string  `json:"job_started_at,omitempty"`
	JobCompletedAt                    string  `json:"job_completed_at,omitempty"`
	JobSteps                          string  `json:"job_steps,omitempty"`
	JobLabels                         string  `json:"job_labels,omitempty"`
	JobRunnerID                       int64   `json:"job_runner_id,omitempty"`
	JobRunnerName                     string  `json:"job_runner_name,omitempty"`
	JobRunnerGroupID                  int64   `json:"job_runner_group_id,omitempty"`
	JobRunnerGroupName                string  `json:"job_runner_group_name,omitempty"`
	JobRunAttempt                     int64   `json:"job_run_attempt,omitempty"`
	JobDurationSeconds                float64 `json:"job_duration,omitempty"`
	JobDurationHumanReadable          string  `json:"job_duration_human_readable,omitempty"`
	RoundedUpJobDurationSeconds       float64 `json:"rounded_up_job_duration,omitempty"`
	RoundedUpJobDurationHumanReadable string  `json:"rounded_up_job_duration_human_readable,omitempty"`
	PricePerMinuteInUSD               float64 `json:"price_per_minute_in_usd,omitempty"`
	BillableInUSD                     float64 `json:"billable_in_usd,omitempty"`
	Runner                            string  `json:"runner,omitempty"`
}

func FlattenJobs(jobs []JobDetails, shouldObfuscate bool) []FlatJobDetails {
	var flattened []FlatJobDetails
	for _, job := range jobs {
		flattened = append(flattened, FlattenJob(job, shouldObfuscate))
	}
	return flattened
}

func FlattenJob(job JobDetails, shouldObfuscate bool) FlatJobDetails {
	stepsBytes, _ := json.Marshal(job.Job.Steps)
	steps := string(stepsBytes)

	ownerName := *job.Repo.Owner.Login
	repoName := *job.Repo.Name
	actorLogin := *job.WorkflowRun.Actor.Login
	workflowRunName := *job.WorkflowRun.Name
	workflowRunDisplayTitle := *job.WorkflowRun.DisplayTitle

	if shouldObfuscate {
		ownerName = obfuscateString(ownerName)
		repoName = obfuscateString(repoName)
		actorLogin = obfuscateString(actorLogin)
		workflowRunName = obfuscateString(workflowRunName)
		workflowRunDisplayTitle = obfuscateString(workflowRunDisplayTitle)
	}

	return FlatJobDetails{
		OwnerName:                         ownerName,
		RepoID:                            *job.Repo.ID,
		RepoName:                          repoName,
		WorkflowID:                        *job.Workflow.ID,
		WorkflowName:                      *job.Workflow.Name,
		WorkflowRunID:                     *job.WorkflowRun.ID,
		WorkflowRunName:                   workflowRunName,
		HeadBranch:                        *job.WorkflowRun.HeadBranch,
		HeadSHA:                           *job.WorkflowRun.HeadSHA,
		WorkflowRunRunNumber:              *job.WorkflowRun.RunNumber,
		WorkflowRunRunAttempt:             *job.WorkflowRun.RunAttempt,
		WorkflowRunEvent:                  *job.WorkflowRun.Event,
		WorkflowRunDisplayTitle:           workflowRunDisplayTitle,
		WorkflowRunStatus:                 *job.WorkflowRun.Status,
		WorkflowRunConclusion:             *job.WorkflowRun.Conclusion,
		WorkflowRunCreatedAt:              job.WorkflowRun.CreatedAt.String(),
		WorkflowRunUpdatedAt:              job.WorkflowRun.UpdatedAt.String(),
		WorkflowRunRunStartedAt:           job.WorkflowRun.RunStartedAt.String(),
		ActorLogin:                        actorLogin,
		JobID:                             *job.Job.ID,
		JobName:                           *job.Job.Name,
		JobStatus:                         *job.Job.Status,
		JobConclusion:                     *job.Job.Conclusion,
		JobCreatedAt:                      job.Job.CreatedAt.String(),
		JobStartedAt:                      job.Job.StartedAt.String(),
		JobCompletedAt:                    job.Job.CompletedAt.String(),
		JobSteps:                          steps,
		JobLabels:                         strings.Join(job.Job.Labels, "; "),
		JobRunnerID:                       *job.Job.RunnerID,
		JobRunnerName:                     *job.Job.RunnerName,
		JobRunnerGroupID:                  *job.Job.RunnerGroupID,
		JobRunnerGroupName:                *job.Job.RunnerGroupName,
		JobRunAttempt:                     *job.Job.RunAttempt,
		JobDurationSeconds:                job.JobDuration.Seconds(),
		JobDurationHumanReadable:          job.JobDuration.String(),
		RoundedUpJobDurationSeconds:       job.RoundedUpJobDuration.Seconds(),
		RoundedUpJobDurationHumanReadable: job.RoundedUpJobDuration.String(),
		PricePerMinuteInUSD:               job.PricePerMinuteInUSD,
		BillableInUSD:                     job.BillableInUSD,
		Runner:                            job.Runner,
	}
}

func obfuscateString(input string) string {
	if len(input) <= 3 {
		return input
	}

	// Check if the string is an email address
	parts := strings.Split(input, "@")
	if len(parts) == 2 {
		// Handle email address
		username := parts[0]
		domain := parts[1]

		// Keep first 3 chars of username
		visiblePart := username[:3]
		maskedPart := strings.Repeat("*", len(username)-3)

		return visiblePart + maskedPart + "@" + domain
	}

	// For non-email strings
	visiblePart := input[:3]
	maskedPart := strings.Repeat("*", len(input)-3)
	return visiblePart + maskedPart
}

func (rd *ReportData) MarshalJSON() ([]byte, error) {
	type Alias ReportData
	return json.Marshal(&struct {
		Jobs []JobDetails `json:"jobs"`
		*Alias
	}{
		Jobs:  rd.Jobs,
		Alias: (*Alias)(rd),
	})
}
