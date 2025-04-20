package reports

import (
	"encoding/csv"
	"os"
	"reflect"
	"strconv"

	"github.com/rs/zerolog"
)

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

func (g *CSVGenerator) generateJobsReport(jobs []JobDetails) error {
	if len(jobs) == 0 {
		g.logger.Info().Msg("No runs in the requested time frame")
		return nil
	}

	flattened := FlattenJobs(jobs)
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
		values[i] = val.(string)
	}

	return values
}
