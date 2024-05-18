package main

import (
	"context"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
)

var PAGE_SIZE = 30

func logResponse(resp *github.Response) {
	logger.
		Debug().
		Str("method", resp.Request.Method).
		Str("url", resp.Request.URL.RequestURI()).
		Str("status", resp.Status).
		Msg("")
}

func getRepoDetails(repo repository.Repository, client *github.Client) *github.Repository {
	r, resp, err := client.Repositories.Get(context.Background(), repo.Owner, repo.Name)
	logResponse(resp)
	checkErr(err)
	return r
}

func getWorkflows(repo repository.Repository, client *github.Client) *github.Workflows {
	opt := &github.ListOptions{
		PerPage: PAGE_SIZE,
	}

	allWfls := &github.Workflows{
		TotalCount: github.Int(0),
		Workflows:  []*github.Workflow{},
	}
	for {
		wfls, resp, err := client.Actions.ListWorkflows(context.Background(), repo.Owner, repo.Name, opt)
		logResponse(resp)
		checkErr(err)
		allWfls.Workflows = append(allWfls.Workflows, wfls.Workflows...)
		totalCount := *allWfls.TotalCount + *wfls.TotalCount
		allWfls.TotalCount = &totalCount
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// todo: implement rate limiting - https://github.com/google/go-github/tree/master?tab=readme-ov-file#rate-limiting
	// todo: implement conditional requests - https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api?apiVersion=2022-11-28#use-conditional-requests-if-appropriate
	// todo: caching?

	return allWfls
}

// func getRuns(repo repository.Repository, client *github.Client, wflId int64, from string) *github.WorkflowRuns {
// 	fromStr := ">="
// 	if from == "" {
// 		fromTimestamp := time.Now().Add(time.Duration(-24*7) * time.Hour)
// 		fromStr = fromStr + fromTimestamp.Format("2006-01-02")
// 	} else {
// 		fromStr = fromStr + from
// 	}

// 	opt := &github.ListWorkflowRunsOptions{
// 		ListOptions: github.ListOptions{
// 			PerPage: PAGE_SIZE,
// 		},
// 		Created: fromStr,
// 	}

// 	allRuns := &github.WorkflowRuns{
// 		TotalCount:   github.Int(0),
// 		WorkflowRuns: []*github.WorkflowRun{},
// 	}
// 	for {
// 		runs, resp, err := client.Actions.ListWorkflowRunsByID(context.Background(), repo.Owner, repo.Name, wflId, opt)
// 		logResponse(resp)
// 		checkErr(err)
// 		allRuns.WorkflowRuns = append(allRuns.WorkflowRuns, runs.WorkflowRuns...)
// 		totalCount := *allRuns.TotalCount + *runs.TotalCount
// 		allRuns.TotalCount = &totalCount
// 		if resp.NextPage == 0 {
// 			break
// 		}
// 		opt.Page = resp.NextPage
// 	}

// 	return allRuns
// }

func getRepositoryRuns(repo repository.Repository, client *github.Client, from string) *github.WorkflowRuns {
	fromStr := ">="
	if from == "" {
		fromTimestamp := time.Now().Add(time.Duration(-24*7) * time.Hour)
		fromStr = fromStr + fromTimestamp.Format("2006-01-02")
	} else {
		fromStr = fromStr + from
	}

	opt := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{
			PerPage: PAGE_SIZE,
		},
		Created: fromStr,
	}

	allRuns := &github.WorkflowRuns{
		TotalCount:   github.Int(0),
		WorkflowRuns: []*github.WorkflowRun{},
	}
	for {
		runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(context.Background(), repo.Owner, repo.Name, opt)
		logResponse(resp)
		checkErr(err)
		allRuns.WorkflowRuns = append(allRuns.WorkflowRuns, runs.WorkflowRuns...)
		totalCount := *allRuns.TotalCount + *runs.TotalCount
		allRuns.TotalCount = &totalCount
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRuns
}

func getJobs(repo repository.Repository, client *github.Client, runId int64) *github.Jobs {
	opt := &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{
			PerPage: PAGE_SIZE,
		},
	}

	allJobs := &github.Jobs{
		TotalCount: github.Int(0),
		Jobs:       []*github.WorkflowJob{},
	}
	for {
		jobs, resp, err := client.Actions.ListWorkflowJobs(context.Background(), repo.Owner, repo.Name, runId, opt)
		logResponse(resp)
		checkErr(err)
		allJobs.Jobs = append(allJobs.Jobs, jobs.Jobs...)
		totalCount := *allJobs.TotalCount + *jobs.TotalCount
		allJobs.TotalCount = &totalCount
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allJobs
}

func getAttempts(repo repository.Repository, client *github.Client, runId int64, attempt int64) *github.Jobs {
	opt := &github.ListOptions{
		PerPage: PAGE_SIZE,
	}

	allJobs := &github.Jobs{
		TotalCount: github.Int(0),
		Jobs:       []*github.WorkflowJob{},
	}
	for {
		jobs, resp, err := client.Actions.ListWorkflowJobsAttempt(context.Background(), repo.Owner, repo.Name, runId, attempt, opt)
		logResponse(resp)
		checkErr(err)
		allJobs.Jobs = append(allJobs.Jobs, jobs.Jobs...)
		totalCount := *allJobs.TotalCount + *jobs.TotalCount
		allJobs.TotalCount = &totalCount
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allJobs
}

func getRunDurationInMS(repo repository.Repository, client *github.Client, runId int64) *github.WorkflowRunUsage {
	r, resp, err := client.Actions.GetWorkflowRunUsageByID(context.Background(), repo.Owner, repo.Name, runId)
	logResponse(resp)
	checkErr(err)
	return r
}
