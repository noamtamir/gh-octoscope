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
	WorkflowRun          *github.WorkflowRun `json:"workflowRun,omitempty"`
	Job                  *github.WorkflowJob `json:"job,omitempty"`
	JobDuration          time.Duration       `json:"jobDuration"`
	RoundedUpJobDuration time.Duration       `json:"roundedUpJobDuration"`
	PricePerMinuteInUSD  float64             `json:"pricePerMinuteInUsd"`
	BillableInUSD        float64             `json:"billableInUsd"`
	Runner               string              `json:"runner,omitempty"`
}

// TotalCosts represents the total costs across all jobs
type TotalCosts struct {
	JobDuration          time.Duration `json:"jobDuration"`
	RoundedUpJobDuration time.Duration `json:"roundedUpJobDuration"`
	BillableInUSD        float64       `json:"billableInUsd"`
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
	OwnerName                   string `json:"ownerName,omitempty"`
	RepoID                      string `json:"repoId,omitempty"`
	RepoName                    string `json:"repoName,omitempty"`
	WorkflowID                  string `json:"workflowId,omitempty"`
	WorkflowName                string `json:"workflowName,omitempty"`
	WorkflowRunID               string `json:"workflowRunId,omitempty"`
	WorkflowRunName             string `json:"workflowRunName,omitempty"`
	HeadBranch                  string `json:"headBranch,omitempty"`
	HeadSHA                     string `json:"headSha,omitempty"`
	WorkflowRunRunNumber        string `json:"workflowRunRunNumber,omitempty"`
	WorkflowRunRunAttempt       string `json:"workflowRunRunAttempt,omitempty"`
	WorkflowRunEvent            string `json:"workflowRunEvent,omitempty"`
	WorkflowRunDisplayTitle     string `json:"workflowRunDisplayTitle,omitempty"`
	WorkflowRunStatus           string `json:"workflowRunStatus,omitempty"`
	WorkflowRunConclusion       string `json:"workflowRunConclusion,omitempty"`
	WorkflowRunCreatedAt        string `json:"workflowRunCreatedAt,omitempty"`
	WorkflowRunUpdatedAt        string `json:"workflowRunUpdatedAt,omitempty"`
	WorkflowRunRunStartedAt     string `json:"workflowRunRunStartedAt,omitempty"`
	ActorLogin                  string `json:"actorLogin,omitempty"`
	JobID                       string `json:"jobId,omitempty"`
	JobName                     string `json:"jobName,omitempty"`
	JobStatus                   string `json:"jobStatus,omitempty"`
	JobConclusion               string `json:"jobConclusion,omitempty"`
	JobCreatedAt                string `json:"jobCreatedAt,omitempty"`
	JobStartedAt                string `json:"jobStartedAt,omitempty"`
	JobCompletedAt              string `json:"jobCompletedAt,omitempty"`
	JobSteps                    string `json:"jobSteps,omitempty"`
	JobLabels                   string `json:"jobLabels,omitempty"`
	JobRunnerID                 string `json:"jobRunnerId,omitempty"`
	JobRunnerName               string `json:"jobRunnerName,omitempty"`
	JobRunnerGroupID            string `json:"jobRunnerGroupId,omitempty"`
	JobRunnerGroupName          string `json:"jobRunnerGroupName,omitempty"`
	JobRunAttempt               string `json:"jobRunAttempt,omitempty"`
	JobDurationSeconds          string `json:"jobDuration,omitempty"`
	RoundedUpJobDurationSeconds string `json:"roundedUpJobDuration,omitempty"`
	PricePerMinuteInUSD         string `json:"pricePerMinuteInUsd,omitempty"`
	BillableInUSD               string `json:"billableInUsd,omitempty"`
	Runner                      string `json:"runner,omitempty"`
}
