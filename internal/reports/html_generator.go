package reports

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

//go:embed template/report.html
var templateHTML string

// HTMLGenerator generates HTML reports
type HTMLGenerator struct {
	outputPath string
	template   string
	logger     zerolog.Logger
}

// NewHTMLGenerator creates a new HTML report generator
func NewHTMLGenerator(outputPath string, logger zerolog.Logger) (*HTMLGenerator, error) {
	return &HTMLGenerator{
		outputPath: outputPath,
		template:   templateHTML,
		logger:     logger,
	}, nil
}

// Generate implements the Generator interface for HTML reports
func (g *HTMLGenerator) Generate(data *ReportData) error {
	// Create data directory
	dir := filepath.Dir(g.outputPath)
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Write summary.json
	summaryPath := filepath.Join(dataDir, "summary.json")
	summaryData := struct {
		Totals TotalCosts `json:"totals"`
	}{
		Totals: data.Totals,
	}
	if err := g.writeJSON(summaryPath, summaryData); err != nil {
		return err
	}

	// Write all jobs to a single file
	jobsPath := filepath.Join(dataDir, "jobs.json")
	if err := g.writeJSON(jobsPath, data.Jobs); err != nil {
		return err
	}

	// Generate the HTML file
	return g.generateHTML(filepath.Base(g.outputPath))
}

func (g *HTMLGenerator) writeJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(data)
}

func (g *HTMLGenerator) generateHTML(dataPath string) error {
	file, err := os.Create(g.outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl := template.Must(template.New("report").Parse(g.template))
	if err := tmpl.Execute(file, nil); err != nil {
		return err
	}

	g.logger.Info().Msgf("%s and data files created successfully!", g.outputPath)
	return nil
}
