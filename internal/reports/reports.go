package reports

import (
	"encoding/json"
	"html/template"
	"os"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
)

// Generator defines the interface for report generation
type Generator interface {
	Generate(data *ReportData) error
}

// ReportData contains all the data needed for report generation
type ReportData struct {
	Jobs   []JobDetails `json:"jobs"`
	Totals TotalCosts   `json:"totals"`
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

// CSVGenerator generates CSV reports
type CSVGenerator struct {
	jobsPath   string
	totalsPath string
	logger     zerolog.Logger
}

// NewCSVGenerator creates a new CSV report generator
func NewCSVGenerator(jobsPath, totalsPath string, logger zerolog.Logger) *CSVGenerator {
	return &CSVGenerator{
		jobsPath:   jobsPath,
		totalsPath: totalsPath,
		logger:     logger,
	}
}

// Generate implements the Generator interface for CSV reports
func (g *CSVGenerator) Generate(data *ReportData) error {
	if err := g.generateJobsReport(data.Jobs); err != nil {
		return err
	}
	return g.generateTotalsReport(data.Totals)
}

// HTMLGenerator generates HTML reports
type HTMLGenerator struct {
	outputPath string
	template   string
	logger     zerolog.Logger
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

// NewHTMLGenerator creates a new HTML report generator
func NewHTMLGenerator(outputPath, templatePath string, logger zerolog.Logger) (*HTMLGenerator, error) {
	tmpl, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}

	return &HTMLGenerator{
		outputPath: outputPath,
		template:   string(tmpl),
		logger:     logger,
	}, nil
}

// Generate implements the Generator interface for HTML reports
func (g *HTMLGenerator) Generate(data *ReportData) error {
	file, err := os.Create(g.outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl := template.Must(template.New("report").Parse(g.template))
	if err := tmpl.Execute(file, data); err != nil {
		return err
	}

	g.logger.Info().Msgf("%s created successfully!", g.outputPath)
	return nil
}

type flatJobDetails struct {
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
