package main

import (
	_ "embed"
	"html/template"
	"io"
	"os"
)

//go:embed report-template.html
var reportTemplate string

func renderTemplate(w io.Writer, jobsDetails []FlattenedJob) {
	tmpl := template.Must(template.New("report").Parse(reportTemplate))
	tmpl.Execute(w, jobsDetails)
}

func generateHtmlFile(jobsDetails []JobDetails) {
	htmlFile, err := os.Create("report.html")
	checkErr(err)
	defer htmlFile.Close()
	renderTemplate(htmlFile, flattenJobs(jobsDetails))
	logger.Info().Msg("report.html created successfully!")
}
