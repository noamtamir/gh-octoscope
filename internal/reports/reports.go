package reports

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
)

// Generator defines the interface for report generation
type Generator interface {
	Generate(data *ReportData) error
}

// ReportData contains all the data needed for report generation
type ReportData struct {
	Jobs          []JobDetails `json:"jobs"`
	Totals        TotalCosts   `json:"totals"`
	ObfuscateData bool         `json:"-"` // Controls whether sensitive data should be obfuscated
}

// JobDetails contains the details of a workflow job
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

// TotalCosts represents the total costs across all jobs
type TotalCosts struct {
	JobDuration          time.Duration `json:"job_duration"`
	RoundedUpJobDuration time.Duration `json:"rounded_up_job_duration"`
	BillableInUSD        float64       `json:"billable_in_usd"`
}

// FlatJobDetails is a flattened representation of JobDetails with all fields as strings
type FlatJobDetails struct {
	OwnerName                   string `json:"owner_name,omitempty"`
	RepoID                      string `json:"repo_id,omitempty"`
	RepoName                    string `json:"repo_name,omitempty"`
	WorkflowID                  string `json:"workflow_id,omitempty"`
	WorkflowName                string `json:"workflow_name,omitempty"`
	WorkflowRunID               string `json:"workflow_run_id,omitempty"`
	WorkflowRunName             string `json:"workflow_run_name,omitempty"`
	HeadBranch                  string `json:"head_branch,omitempty"`
	HeadSHA                     string `json:"head_sha,omitempty"`
	WorkflowRunRunNumber        string `json:"workflow_run_run_number,omitempty"`
	WorkflowRunRunAttempt       string `json:"workflow_run_run_attempt,omitempty"`
	WorkflowRunEvent            string `json:"workflow_run_event,omitempty"`
	WorkflowRunDisplayTitle     string `json:"workflow_run_display_title,omitempty"`
	WorkflowRunStatus           string `json:"workflow_run_status,omitempty"`
	WorkflowRunConclusion       string `json:"workflow_run_conclusion,omitempty"`
	WorkflowRunCreatedAt        string `json:"workflow_run_created_at,omitempty"`
	WorkflowRunUpdatedAt        string `json:"workflow_run_updated_at,omitempty"`
	WorkflowRunRunStartedAt     string `json:"workflow_run_run_started_at,omitempty"`
	ActorLogin                  string `json:"actor_login,omitempty"`
	JobID                       string `json:"job_id,omitempty"`
	JobName                     string `json:"job_name,omitempty"`
	JobStatus                   string `json:"job_status,omitempty"`
	JobConclusion               string `json:"job_conclusion,omitempty"`
	JobCreatedAt                string `json:"job_created_at,omitempty"`
	JobStartedAt                string `json:"job_started_at,omitempty"`
	JobCompletedAt              string `json:"job_completed_at,omitempty"`
	JobSteps                    string `json:"job_steps,omitempty"`
	JobLabels                   string `json:"job_labels,omitempty"`
	JobRunnerID                 string `json:"job_runner_id,omitempty"`
	JobRunnerName               string `json:"job_runner_name,omitempty"`
	JobRunnerGroupID            string `json:"job_runner_group_id,omitempty"`
	JobRunnerGroupName          string `json:"job_runner_group_name,omitempty"`
	JobRunAttempt               string `json:"job_run_attempt,omitempty"`
	JobDurationSeconds          string `json:"job_duration,omitempty"`
	RoundedUpJobDurationSeconds string `json:"rounded_up_job_duration,omitempty"`
	PricePerMinuteInUSD         string `json:"price_per_minute_in_usd,omitempty"`
	BillableInUSD               string `json:"billable_in_usd,omitempty"`
	Runner                      string `json:"runner,omitempty"`
}

// FlattenJobs converts JobDetails slice to FlatJobDetails slice
func FlattenJobs(jobs []JobDetails, shouldObfuscate bool) []FlatJobDetails {
	var flattened []FlatJobDetails
	for _, job := range jobs {
		flattened = append(flattened, FlattenJob(job, shouldObfuscate))
	}
	return flattened
}

// FlattenJob converts a single JobDetails to FlatJobDetails
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
		OwnerName:                   ownerName,
		RepoID:                      strconv.FormatInt(*job.Repo.ID, 10),
		RepoName:                    repoName,
		WorkflowID:                  strconv.FormatInt(*job.Workflow.ID, 10),
		WorkflowName:                *job.Workflow.Name,
		WorkflowRunID:               strconv.FormatInt(*job.WorkflowRun.ID, 10),
		WorkflowRunName:             workflowRunName,
		HeadBranch:                  *job.WorkflowRun.HeadBranch,
		HeadSHA:                     *job.WorkflowRun.HeadSHA,
		WorkflowRunRunNumber:        strconv.Itoa(*job.WorkflowRun.RunNumber),
		WorkflowRunRunAttempt:       strconv.Itoa(*job.WorkflowRun.RunAttempt),
		WorkflowRunEvent:            *job.WorkflowRun.Event,
		WorkflowRunDisplayTitle:     workflowRunDisplayTitle,
		WorkflowRunStatus:           *job.WorkflowRun.Status,
		WorkflowRunConclusion:       *job.WorkflowRun.Conclusion,
		WorkflowRunCreatedAt:        job.WorkflowRun.CreatedAt.String(),
		WorkflowRunUpdatedAt:        job.WorkflowRun.UpdatedAt.String(),
		WorkflowRunRunStartedAt:     job.WorkflowRun.RunStartedAt.String(),
		ActorLogin:                  actorLogin,
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

// obfuscateString masks a string by keeping the first three characters and domain extension (if exists) visible
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
