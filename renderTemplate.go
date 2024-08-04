package main

import (
	_ "embed"
	"html/template"
	"io"
	"os"
)

//go:embed report-template.html
var reportTemplate string

type InputData struct {
	Jobs   []JobDetails
	Totals TotalsString
}

func renderTemplate(w io.Writer, data InputData) {
	tmpl := template.Must(template.New("report").Parse(reportTemplate))

	tmpl.Execute(w, data)
}

func generateHtmlFile(jobsDetails []JobDetails, totals Totals) {
	htmlFile, err := os.Create("report.html")
	checkErr(err)
	defer htmlFile.Close()
	data := InputData{
		Jobs:   jobsDetails,
		Totals: totals.toTotalsString(),
	}
	renderTemplate(htmlFile, data)
	logger.Info().Msg("report.html created successfully!")
}
