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
	OwnerName                         *string  `json:"owner_name,omitempty"`
	RepoID                            *int64   `json:"repo_id,omitempty"`
	RepoName                          *string  `json:"repo_name,omitempty"`
	WorkflowID                        *int64   `json:"workflow_id,omitempty"`
	WorkflowName                      *string  `json:"workflow_name,omitempty"`
	WorkflowRunID                     *int64   `json:"workflow_run_id,omitempty"`
	WorkflowRunName                   *string  `json:"workflow_run_name,omitempty"`
	HeadBranch                        *string  `json:"head_branch,omitempty"`
	HeadSHA                           *string  `json:"head_sha,omitempty"`
	WorkflowRunRunNumber              *int     `json:"workflow_run_run_number,omitempty"`
	WorkflowRunRunAttempt             *int     `json:"workflow_run_run_attempt,omitempty"`
	WorkflowRunEvent                  *string  `json:"workflow_run_event,omitempty"`
	WorkflowRunDisplayTitle           *string  `json:"workflow_run_display_title,omitempty"`
	WorkflowRunStatus                 *string  `json:"workflow_run_status,omitempty"`
	WorkflowRunConclusion             *string  `json:"workflow_run_conclusion,omitempty"`
	WorkflowRunCreatedAt              *string  `json:"workflow_run_created_at,omitempty"`
	WorkflowRunUpdatedAt              *string  `json:"workflow_run_updated_at,omitempty"`
	WorkflowRunRunStartedAt           *string  `json:"workflow_run_run_started_at,omitempty"`
	ActorLogin                        *string  `json:"actor_login,omitempty"`
	JobID                             *int64   `json:"job_id,omitempty"`
	JobName                           *string  `json:"job_name,omitempty"`
	JobStatus                         *string  `json:"job_status,omitempty"`
	JobConclusion                     *string  `json:"job_conclusion,omitempty"`
	JobCreatedAt                      *string  `json:"job_created_at,omitempty"`
	JobStartedAt                      *string  `json:"job_started_at,omitempty"`
	JobCompletedAt                    *string  `json:"job_completed_at,omitempty"`
	JobSteps                          *string  `json:"job_steps,omitempty"`
	JobLabels                         *string  `json:"job_labels,omitempty"`
	JobRunnerID                       *int64   `json:"job_runner_id,omitempty"`
	JobRunnerName                     *string  `json:"job_runner_name,omitempty"`
	JobRunnerGroupID                  *int64   `json:"job_runner_group_id,omitempty"`
	JobRunnerGroupName                *string  `json:"job_runner_group_name,omitempty"`
	JobRunAttempt                     *int64   `json:"job_run_attempt,omitempty"`
	JobDurationSeconds                *float64 `json:"job_duration,omitempty"`
	JobDurationHumanReadable          *string  `json:"job_duration_human_readable,omitempty"`
	RoundedUpJobDurationSeconds       *float64 `json:"rounded_up_job_duration,omitempty"`
	RoundedUpJobDurationHumanReadable *string  `json:"rounded_up_job_duration_human_readable,omitempty"`
	PricePerMinuteInUSD               *float64 `json:"price_per_minute_in_usd,omitempty"`
	BillableInUSD                     *float64 `json:"billable_in_usd,omitempty"`
	Runner                            *string  `json:"runner,omitempty"`
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

	// Helper functions for safe pointer assignment
	strPtr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	int64Ptr := func(i *int64) *int64 {
		return i
	}
	intPtr := func(i *int) *int {
		return i
	}
	float64Ptr := func(f float64) *float64 {
		return &f
	}
	safeString := func(ptr *string) string {
		if ptr != nil {
			return *ptr
		}
		return ""
	}
	timestampPtr := func(ts *github.Timestamp) *string {
		if ts == nil {
			return nil
		}
		t := ts.Time
		if t.IsZero() {
			return nil
		}
		s := t.String()
		return &s
	}

	// Compose values, obfuscate if needed
	ownerName := ""
	if job.Repo != nil && job.Repo.Owner != nil && job.Repo.Owner.Login != nil {
		ownerName = *job.Repo.Owner.Login
	}
	repoName := ""
	if job.Repo != nil && job.Repo.Name != nil {
		repoName = *job.Repo.Name
	}
	actorLogin := ""
	if job.WorkflowRun != nil && job.WorkflowRun.Actor != nil && job.WorkflowRun.Actor.Login != nil {
		actorLogin = *job.WorkflowRun.Actor.Login
	}
	workflowRunName := ""
	if job.WorkflowRun != nil && job.WorkflowRun.Name != nil {
		workflowRunName = *job.WorkflowRun.Name
	}
	workflowRunDisplayTitle := ""
	if job.WorkflowRun != nil && job.WorkflowRun.DisplayTitle != nil {
		workflowRunDisplayTitle = *job.WorkflowRun.DisplayTitle
	}

	if shouldObfuscate {
		ownerName = obfuscateString(ownerName)
		repoName = obfuscateString(repoName)
		actorLogin = obfuscateString(actorLogin)
		workflowRunName = obfuscateString(workflowRunName)
		workflowRunDisplayTitle = obfuscateString(workflowRunDisplayTitle)
	}

	var jobLabels *string
	if job.Job != nil && job.Job.Labels != nil && len(job.Job.Labels) > 0 {
		joined := strings.Join(job.Job.Labels, "; ")
		jobLabels = &joined
	}

	return FlatJobDetails{
		OwnerName:                         strPtr(ownerName),
		RepoID:                            int64Ptr(job.Repo.ID),
		RepoName:                          strPtr(repoName),
		WorkflowID:                        int64Ptr(job.Workflow.ID),
		WorkflowName:                      strPtr(safeString(job.Workflow.Name)),
		WorkflowRunID:                     int64Ptr(job.WorkflowRun.ID),
		WorkflowRunName:                   strPtr(workflowRunName),
		HeadBranch:                        strPtr(safeString(job.WorkflowRun.HeadBranch)),
		HeadSHA:                           strPtr(safeString(job.WorkflowRun.HeadSHA)),
		WorkflowRunRunNumber:              intPtr(job.WorkflowRun.RunNumber),
		WorkflowRunRunAttempt:             intPtr(job.WorkflowRun.RunAttempt),
		WorkflowRunEvent:                  strPtr(safeString(job.WorkflowRun.Event)),
		WorkflowRunDisplayTitle:           strPtr(workflowRunDisplayTitle),
		WorkflowRunStatus:                 strPtr(safeString(job.WorkflowRun.Status)),
		WorkflowRunConclusion:             strPtr(safeString(job.WorkflowRun.Conclusion)),
		WorkflowRunCreatedAt:              timestampPtr(job.WorkflowRun.CreatedAt),
		WorkflowRunUpdatedAt:              timestampPtr(job.WorkflowRun.UpdatedAt),
		WorkflowRunRunStartedAt:           timestampPtr(job.WorkflowRun.RunStartedAt),
		ActorLogin:                        strPtr(actorLogin),
		JobID:                             int64Ptr(job.Job.ID),
		JobName:                           strPtr(safeString(job.Job.Name)),
		JobStatus:                         strPtr(safeString(job.Job.Status)),
		JobConclusion:                     strPtr(safeString(job.Job.Conclusion)),
		JobCreatedAt:                      timestampPtr(job.Job.CreatedAt),
		JobStartedAt:                      timestampPtr(job.Job.StartedAt),
		JobCompletedAt:                    timestampPtr(job.Job.CompletedAt),
		JobSteps:                          strPtr(steps),
		JobLabels:                         jobLabels,
		JobRunnerID:                       int64Ptr(job.Job.RunnerID),
		JobRunnerName:                     strPtr(safeString(job.Job.RunnerName)),
		JobRunnerGroupID:                  int64Ptr(job.Job.RunnerGroupID),
		JobRunnerGroupName:                strPtr(safeString(job.Job.RunnerGroupName)),
		JobRunAttempt:                     int64Ptr(job.Job.RunAttempt),
		JobDurationSeconds:                float64Ptr(job.JobDuration.Seconds()),
		JobDurationHumanReadable:          strPtr(job.JobDuration.String()),
		RoundedUpJobDurationSeconds:       float64Ptr(job.RoundedUpJobDuration.Seconds()),
		RoundedUpJobDurationHumanReadable: strPtr(job.RoundedUpJobDuration.String()),
		PricePerMinuteInUSD:               float64Ptr(job.PricePerMinuteInUSD),
		BillableInUSD:                     float64Ptr(job.BillableInUSD),
		Runner:                            strPtr(job.Runner),
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
