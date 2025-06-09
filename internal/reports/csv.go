package reports

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

// CSVGenerator generates CSV reports
type CSVGenerator struct {
	jobsPath       string
	totalsPath     string
	logger         zerolog.Logger
	ownerName      string
	repoName       string
	reportID       string
	timeFormat     bool   // whether to use timestamped filenames
	dateTimeFormat string // format for timestamps
}

// NewCSVGenerator creates a new CSV report generator
func NewCSVGenerator(jobsPath, totalsPath string, logger zerolog.Logger) *CSVGenerator {
	return &CSVGenerator{
		jobsPath:       jobsPath,
		totalsPath:     totalsPath,
		logger:         logger,
		timeFormat:     false,
		dateTimeFormat: "2006-01-02T15:04:05",
	}
}

// NewCSVGeneratorWithFormat creates a new CSV report generator with formatted filenames
func NewCSVGeneratorWithFormat(basePath string, owner, repo, reportID string, logger zerolog.Logger) *CSVGenerator {
	timestamp := time.Now().Format("2006-01-02T15:04:05")
	jobsPath := basePath + "/" + timestamp + "_" + owner + "_" + repo + "_" + reportID + "_report.csv"
	totalsPath := basePath + "/" + timestamp + "_" + owner + "_" + repo + "_" + reportID + "_totals.csv"

	return &CSVGenerator{
		jobsPath:       jobsPath,
		totalsPath:     totalsPath,
		logger:         logger,
		ownerName:      owner,
		repoName:       repo,
		reportID:       reportID,
		timeFormat:     true,
		dateTimeFormat: "2006-01-02T15:04:05",
	}
}

func (g *CSVGenerator) GetJobsPath() string {
	return g.jobsPath
}

func (g *CSVGenerator) GetTotalsPath() string {
	return g.totalsPath
}

func (g *CSVGenerator) Generate(data *ReportData) error {
	g.logger.Debug().Msg("Generating CSV report")

	if err := g.generateJobsReport(data.Jobs, data.ObfuscateData); err != nil {
		return err
	}

	if err := g.generateTotalsReport(data.Totals); err != nil {
		return err
	}

	return nil
}

func (g *CSVGenerator) generateJobsReport(jobs []JobDetails, shouldObfuscate bool) error {
	if len(jobs) == 0 {
		g.logger.Debug().Msg("No runs in the requested time frame")
		return nil
	}

	flattened := FlattenJobs(jobs, shouldObfuscate)
	data := g.prepareCSVData(flattened)

	return g.writeCSVFile(g.jobsPath, data)
}

func (g *CSVGenerator) generateTotalsReport(totals TotalCosts) error {
	// Add the requested columns: report_id, owner, repository, report_created_at
	headers := []string{"report_id", "owner", "repository", "report_created_at", "total_job_duration", "total_rounded_up_job_duration", "total_billable_in_usd"}

	// Get current timestamp for report_created_at
	createdAt := time.Now().Format(g.dateTimeFormat)

	// Use defaults for owner/repo/id if not provided (for backward compatibility)
	reportID := g.reportID
	if reportID == "" {
		reportID = "not_specified"
	}

	owner := g.ownerName
	if owner == "" {
		owner = "not_specified"
	}

	repo := g.repoName
	if repo == "" {
		repo = "not_specified"
	}

	data := [][]string{
		headers,
		{
			reportID,
			owner,
			repo,
			createdAt,
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

	g.logger.Debug().Msgf("%s created successfully!", path)
	return nil
}

func (g *CSVGenerator) prepareCSVData(flattened []FlatJobDetails) [][]string {
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

func (g *CSVGenerator) structToStringSlice(fj FlatJobDetails) []string {
	v := reflect.ValueOf(fj)
	n := v.NumField()
	values := make([]string, n)

	for i := 0; i < n; i++ {
		val := v.Field(i).Interface()

		// Convert different types to string
		switch value := val.(type) {
		case string:
			values[i] = value
		case float64:
			values[i] = strconv.FormatFloat(value, 'f', 3, 64)
		case int:
			values[i] = strconv.Itoa(value)
		case int64:
			values[i] = strconv.FormatInt(value, 10)
		default:
			// For any other type, use fmt.Sprint
			values[i] = fmt.Sprint(value)
		}
	}

	return values
}
