package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/jsonpretty"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getWorkflows(repo repository.Repository, client *api.RESTClient) github.Workflows {
	wflRoute := fmt.Sprintf("repos/%s/%s/actions/workflows", repo.Owner, repo.Name)
	var wfls github.Workflows
	err := client.Get(wflRoute, &wfls)
	checkErr(err)
	return wfls
}

func getRuns(repo repository.Repository, client *api.RESTClient, wflId int64) github.WorkflowRuns {
	runsRoute := fmt.Sprintf("repos/%s/%s/actions/workflows/%d/runs", repo.Owner, repo.Name, wflId)
	var runs github.WorkflowRuns
	err := client.Get(runsRoute, &runs)
	checkErr(err)
	return runs
}

func getJobs(repo repository.Repository, client *api.RESTClient, runId int64) github.Jobs {
	jobsRoute := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs", repo.Owner, repo.Name, runId)
	var jobs github.Jobs
	err := client.Get(jobsRoute, &jobs)
	checkErr(err)
	return jobs
}

func getAttempts(repo repository.Repository, client *api.RESTClient, runId int64, attempt int) github.Jobs {
	attemptsRoute := fmt.Sprintf("repos/%s/%s/actions/runs/%d/attempts/%d/jobs", repo.Owner, repo.Name, runId, attempt)
	var attemptJobs github.Jobs
	err := client.Get(attemptsRoute, &attemptJobs)
	if err != nil {
		log.Fatal(err)
	}

	return attemptJobs
}

type WriterColorized struct {
	w io.Writer
	c bool
}

func print(wcs []WriterColorized, obj interface{}) {
	jsonObj, err := json.Marshal(obj)
	checkErr(err)

	for _, wc := range wcs {
		r := bytes.NewReader(jsonObj)
		err = jsonpretty.Format(wc.w, r, "  ", wc.c) // colorized true doesn't play nice when writing to file
		checkErr(err)
	}
}

func main() {
	// configure writers
	var wcs []WriterColorized
	shouldWriteToStdout := true
	shouldWriteToFile := true

	if shouldWriteToStdout {
		wcs = append(wcs, WriterColorized{w: os.Stdout, c: true})
	}
	if shouldWriteToFile {
		f, err := os.Create("report.log")
		checkErr(err)
		wcs = append(wcs, WriterColorized{w: f, c: false})
		defer f.Close()
	}

	// setup
	client, err := api.DefaultRESTClient()
	checkErr(err)

	repo, err := repository.Current()
	checkErr(err)

	// get data
	wfls := getWorkflows(repo, client)
	print(wcs, wfls)

	if *wfls.TotalCount > 0 {
		for _, wfl := range wfls.Workflows {
			runs := getRuns(repo, client, *wfl.ID)
			print(wcs, runs)

			if *runs.TotalCount > 0 {
				for _, run := range runs.WorkflowRuns {
					jobs := getJobs(repo, client, *run.ID)
					print(wcs, jobs)

					if *run.RunAttempt > 1 {
						// TODO: revisit attempt logic...
						for i := 1; i < int(*run.RunAttempt)-1; i++ {
							attemptJobs := getAttempts(repo, client, *run.ID, i)
							print(wcs, attemptJobs)
						}
					}
				}
			}
		}
	}
}
