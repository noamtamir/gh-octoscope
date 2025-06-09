package api

import (
	"context"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
)

type Client interface {
	GetRepository(ctx context.Context) (*github.Repository, error)
	ListWorkflows(ctx context.Context) (*github.Workflows, error)
	ListRepositoryRuns(ctx context.Context, from time.Time) (*github.WorkflowRuns, error)
	ListWorkflowJobs(ctx context.Context, runID int64) (*github.Jobs, error)
	ListWorkflowJobsAttempt(ctx context.Context, runID, attempt int64) (*github.Jobs, error)
}

type client struct {
	ghClient *github.Client
	repo     repository.Repository
	logger   zerolog.Logger
	pageSize int
}

type Config struct {
	PageSize int
	Logger   zerolog.Logger
	Token    string
}

func NewClient(repo repository.Repository, cfg Config) Client {
	return &client{
		ghClient: github.NewClient(nil).WithAuthToken(cfg.Token),
		repo:     repo,
		logger:   cfg.Logger,
		pageSize: cfg.PageSize,
	}
}

func (c *client) GetRepository(ctx context.Context) (*github.Repository, error) {
	repo, resp, err := c.ghClient.Repositories.Get(ctx, c.repo.Owner, c.repo.Name)
	if err != nil {
		return nil, err
	}
	c.logResponse(resp, repo)
	return repo, nil
}

func (c *client) ListWorkflows(ctx context.Context) (*github.Workflows, error) {
	opt := &github.ListOptions{
		PerPage: c.pageSize,
	}

	allWfls := &github.Workflows{
		TotalCount: github.Int(0),
		Workflows:  []*github.Workflow{},
	}

	for {
		wfls, resp, err := c.ghClient.Actions.ListWorkflows(ctx, c.repo.Owner, c.repo.Name, opt)
		if err != nil {
			return nil, err
		}
		c.logResponse(resp, wfls)

		allWfls.Workflows = append(allWfls.Workflows, wfls.Workflows...)
		totalCount := *allWfls.TotalCount + *wfls.TotalCount
		allWfls.TotalCount = &totalCount

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allWfls, nil
}

func (c *client) ListRepositoryRuns(ctx context.Context, from time.Time) (*github.WorkflowRuns, error) {
	opt := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{
			PerPage: c.pageSize,
		},
		Created: ">=" + from.Format("2006-01-02"),
	}

	allRuns := &github.WorkflowRuns{
		TotalCount:   github.Int(0),
		WorkflowRuns: []*github.WorkflowRun{},
	}

	for {
		runs, resp, err := c.ghClient.Actions.ListRepositoryWorkflowRuns(ctx, c.repo.Owner, c.repo.Name, opt)
		if err != nil {
			return nil, err
		}
		c.logResponse(resp, runs)

		allRuns.WorkflowRuns = append(allRuns.WorkflowRuns, runs.WorkflowRuns...)
		totalCount := *allRuns.TotalCount + *runs.TotalCount
		allRuns.TotalCount = &totalCount

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRuns, nil
}

func (c *client) ListWorkflowJobs(ctx context.Context, runID int64) (*github.Jobs, error) {
	opt := &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{
			PerPage: c.pageSize,
		},
	}

	allJobs := &github.Jobs{
		TotalCount: github.Int(0),
		Jobs:       []*github.WorkflowJob{},
	}

	for {
		jobs, resp, err := c.ghClient.Actions.ListWorkflowJobs(ctx, c.repo.Owner, c.repo.Name, runID, opt)
		if err != nil {
			return nil, err
		}
		c.logResponse(resp, jobs)

		allJobs.Jobs = append(allJobs.Jobs, jobs.Jobs...)
		totalCount := *allJobs.TotalCount + *jobs.TotalCount
		allJobs.TotalCount = &totalCount

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allJobs, nil
}

func (c *client) ListWorkflowJobsAttempt(ctx context.Context, runID, attempt int64) (*github.Jobs, error) {
	opt := &github.ListOptions{
		PerPage: c.pageSize,
	}

	allJobs := &github.Jobs{
		TotalCount: github.Int(0),
		Jobs:       []*github.WorkflowJob{},
	}

	for {
		jobs, resp, err := c.ghClient.Actions.ListWorkflowJobsAttempt(ctx, c.repo.Owner, c.repo.Name, runID, attempt, opt)
		if err != nil {
			return nil, err
		}
		c.logResponse(resp, jobs)

		allJobs.Jobs = append(allJobs.Jobs, jobs.Jobs...)
		totalCount := *allJobs.TotalCount + *jobs.TotalCount
		allJobs.TotalCount = &totalCount

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allJobs, nil
}

func (c *client) logResponse(resp *github.Response, body interface{}) {
	c.logger.Debug().
		Str("method", resp.Request.Method).
		Str("url", resp.Request.URL.RequestURI()).
		Str("status", resp.Status).
		Interface("body", body).
		Msg("API response")
}
