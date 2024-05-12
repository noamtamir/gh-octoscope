package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/jsonpretty"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
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

type FlattenedJob struct {
	repo        *github.Repository
	workflow    *github.Workflow
	workflowRun *github.WorkflowRun
	job         *github.WorkflowJob
}

func main() {
	// cli
	consoleLog := flag.Bool("console", true, "Log responses to console")
	reportLog := flag.Bool("report", false, "Log responses to file")
	// csvFile := flag.Bool("csv", false, "Generate csv report")
	// jsonFile := flag.Bool("json", false, "Generate json report")
	flag.Parse()

	// configure writers
	var wcs []WriterColorized

	if *consoleLog {
		wcs = append(wcs, WriterColorized{w: os.Stdout, c: true})
	}
	if *reportLog {
		f, err := os.Create("report.log")
		checkErr(err)
		wcs = append(wcs, WriterColorized{w: f, c: false})
		defer f.Close()
	}

	// setup http client
	host, _ := auth.DefaultHost()
	token, _ := auth.TokenForHost(host)
	client := github.NewClient(nil).WithAuthToken(token)
	repo, err := repository.Current()
	checkErr(err)

	repoDetails := getRepoDetails(repo, client)

	// get data
	var flattenedJobs []FlattenedJob
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
					for _, job := range jobs.Jobs {
						flattenedJobs = append(flattenedJobs, FlattenedJob{
							repo:        repoDetails,
							workflow:    wfl,
							workflowRun: run,
							job:         job,
						})
					}

					if *run.RunAttempt > 1 {
						// TODO: revisit attempt logic...
						for i := 1; i < int(*run.RunAttempt)-1; i++ {
							attemptJobs := getAttempts(repo, client, *run.ID, int64(i))
							print(wcs, attemptJobs)
							for _, job := range attemptJobs.Jobs {
								flattenedJobs = append(flattenedJobs, FlattenedJob{
									repo:        repoDetails,
									workflow:    wfl,
									workflowRun: run,
									job:         job,
								})
							}
						}
					}
				}
			}
		}
	}
	print(wcs, flattenedJobs)
}
