package main

import (
	"encoding/csv"
	"os"
	"reflect"
)

func generateCsvFile(jobsDetails []JobDetails) {
	if len(jobsDetails) == 0 {
		logger.Info().Msg("No runs in the requested time frame")
		return
	}

	flattened := flattenJobs(jobsDetails)
	var data [][]string

	// headers
	t := reflect.TypeOf(flattened[0])
	names := make([]string, t.NumField())
	for i := range names {
		names[i] = t.Field(i).Name
	}
	data = append(data, names)

	// data
	for _, fj := range flattened {
		data = append(data, fj.toCsv())
	}

	csvFile, err := os.Create("report.csv")
	checkErr(err)
	defer csvFile.Close()
	w := csv.NewWriter(csvFile)

	for _, record := range data {
		if err := w.Write(record); err != nil {
			logger.Fatal().Stack().Err(err).Msg("Error writing record to csv")
		}
	}

	w.Flush()
	checkErr(w.Error())
	logger.Info().Msg("report.csv created successfully!")
}

func generateTotalsCsvFile(totals Totals) {
	var data [][]string

	// headers
	headers := []string{"total_job_duration", "total_rounded_up_job_duration", "total_billable_in_usd"}
	data = append(data, headers)

	// data
	totalsString := totals.toTotalsString()
	data = append(data, totalsString.toCsv())

	csvFile, err := os.Create("totals.csv")
	checkErr(err)
	defer csvFile.Close()
	w := csv.NewWriter(csvFile)

	for _, record := range data {
		if err := w.Write(record); err != nil {
			logger.Fatal().Stack().Err(err).Msg("Error writing record to csv")
		}
	}

	w.Flush()
	checkErr(w.Error())
	logger.Info().Msg("totals.csv created successfully!")
}
