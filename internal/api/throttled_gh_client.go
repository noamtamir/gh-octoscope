package api

import (
	"context"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v62/github"
	"golang.org/x/time/rate"
)

// ThrottledClient provides a rate-limited concurrent client for GitHub API
type ThrottledClient interface {
	Client
	FetchRunsWithJobs(ctx context.Context, from time.Time) ([]RunWithJobs, error)
}

// RunWithJobs contains a workflow run with its associated jobs and usage data
type RunWithJobs struct {
	Run         *github.WorkflowRun
	Jobs        []*github.WorkflowJob
	UsageData   *github.WorkflowRunUsage
	Workflow    *github.Workflow
	AttemptJobs map[int][]*github.WorkflowJob
}

// ThrottledClientConfig extends the base Config with concurrency settings
type ThrottledClientConfig struct {
	Config
	MaxConcurrentRequests int           // Maximum number of concurrent requests
	RequestsPerSecond     float64       // Rate limiter requests per second
	Burst                 int           // Maximum burst size for rate limiter
	RetryLimit            int           // Maximum number of retries for a request
	RetryBackoff          time.Duration // Base backoff duration for retries
}

type throttledClient struct {
	client
	limiter      *rate.Limiter
	maxWorkers   int
	retryLimit   int
	retryBackoff time.Duration
}

// NewThrottledClient creates a new throttled client with rate limiting
func NewThrottledClient(repo repository.Repository, cfg ThrottledClientConfig) ThrottledClient {
	// Default values if not provided
	maxWorkers := cfg.MaxConcurrentRequests
	if maxWorkers <= 0 {
		maxWorkers = 5 // Default to 5 concurrent requests
	}

	requestsPerSecond := cfg.RequestsPerSecond
	if requestsPerSecond <= 0 {
		requestsPerSecond = 5 // Default to 5 requests per second (300 per minute)
	}

	burst := cfg.Burst
	if burst <= 0 {
		burst = 10 // Default burst size
	}

	retryLimit := cfg.RetryLimit
	if retryLimit <= 0 {
		retryLimit = 3 // Default to 3 retries
	}

	retryBackoff := cfg.RetryBackoff
	if retryBackoff <= 0 {
		retryBackoff = 1 * time.Second // Default to 1 second backoff
	}

	return &throttledClient{
		client: client{
			ghClient: github.NewClient(nil).WithAuthToken(cfg.Token),
			repo:     repo,
			logger:   cfg.Logger,
			pageSize: cfg.PageSize,
		},
		limiter:      rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
		maxWorkers:   maxWorkers,
		retryLimit:   retryLimit,
		retryBackoff: retryBackoff,
	}
}

// executeWithRateLimit executes a function with rate limiting and retry logic
func (c *throttledClient) executeWithRateLimit(ctx context.Context, fn func() error) error {
	var err error
	var retryCount int

	for retryCount = 0; retryCount <= c.retryLimit; retryCount++ {
		// Wait for rate limiter
		if err = c.limiter.Wait(ctx); err != nil {
			return err
		}

		// Execute the function
		err = fn()
		if err == nil {
			return nil
		}

		// Check if it's a rate limit error
		if rateLimitErr, ok := err.(*github.RateLimitError); ok {
			c.logger.Warn().
				Int("retry", retryCount).
				Time("reset_at", rateLimitErr.Rate.Reset.Time).
				Msg("GitHub rate limit exceeded, waiting for reset")

			// Wait until rate limit resets
			waitTime := time.Until(rateLimitErr.Rate.Reset.Time)
			if waitTime > 0 {
				select {
				case <-time.After(waitTime):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			continue
		}

		// Check if it's a secondary rate limit error (usually HTTP 403)
		if abuseErr, ok := err.(*github.AbuseRateLimitError); ok {
			c.logger.Warn().
				Int("retry", retryCount).
				Str("retry_after", abuseErr.RetryAfter.String()).
				Msg("GitHub secondary rate limit exceeded")

			var waitTime time.Duration
			if abuseErr.RetryAfter.String() != "" {
				waitTime = *abuseErr.RetryAfter
			} else {
				// Use exponential backoff
				waitTime = c.retryBackoff * time.Duration(1<<uint(retryCount))
			}

			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		// For other errors, use exponential backoff
		if retryCount < c.retryLimit {
			backoff := c.retryBackoff * time.Duration(1<<uint(retryCount))
			c.logger.Warn().
				Int("retry", retryCount).
				Dur("backoff", backoff).
				Err(err).
				Msg("API error, retrying with backoff")

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return err
}

// FetchRunsWithJobs fetches workflow runs and their jobs concurrently
func (c *throttledClient) FetchRunsWithJobs(ctx context.Context, from time.Time) ([]RunWithJobs, error) {
	// First, get repository info
	// var repo *github.Repository
	// err := c.executeWithRateLimit(ctx, func() error {
	// 	var err error
	// 	repo, _, err = c.ghClient.Repositories.Get(ctx, c.repo.Owner, c.repo.Name)
	// 	return err
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// Then, get workflows info
	var workflows *github.Workflows
	err := c.executeWithRateLimit(ctx, func() error {
		var err error
		workflows, err = c.ListWorkflows(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup of workflows
	workflowMap := make(map[int64]*github.Workflow)
	for _, wfl := range workflows.Workflows {
		workflowMap[*wfl.ID] = wfl
	}

	// Now, get workflow runs
	var runs *github.WorkflowRuns
	err = c.executeWithRateLimit(ctx, func() error {
		var err error
		runs, err = c.ListRepositoryRuns(ctx, from)
		return err
	})
	if err != nil {
		return nil, err
	}

	if *runs.TotalCount == 0 {
		return []RunWithJobs{}, nil
	}

	// Process workflow runs concurrently with throttling
	results := make([]RunWithJobs, 0, len(runs.WorkflowRuns))
	resultChan := make(chan RunWithJobs)
	errorChan := make(chan error)
	done := make(chan struct{})
	wg := &sync.WaitGroup{}

	// Create a semaphore to limit the number of concurrent goroutines
	sem := make(chan struct{}, c.maxWorkers)

	// Start a goroutine to collect results
	go func() {
		for runWithJobs := range resultChan {
			results = append(results, runWithJobs)
		}
		close(done)
	}()

	// Process each run
	runCount := len(runs.WorkflowRuns)
	for i, run := range runs.WorkflowRuns {
		wg.Add(1)

		// Acquire semaphore slot
		sem <- struct{}{}

		go func(index int, run *github.WorkflowRun) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore slot

			result, err := c.processRun(ctx, run, workflowMap)
			if err != nil {
				c.logger.Error().
					Int("index", index).
					Int64("runID", *run.ID).
					Err(err).
					Msg("Error processing workflow run")
				errorChan <- err
				return
			}

			resultChan <- result
			c.logger.Debug().
				Int("processed", index+1).
				Int("total", runCount).
				Int64("runID", *run.ID).
				Msg("Processed workflow run")
		}(i, run)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Check for errors
	select {
	case err := <-errorChan:
		if err != nil {
			return nil, err
		}
	case <-done:
		// All good
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return results, nil
}

// processRun fetches all data for a single workflow run
func (c *throttledClient) processRun(ctx context.Context, run *github.WorkflowRun, workflowMap map[int64]*github.Workflow) (RunWithJobs, error) {
	result := RunWithJobs{
		Run:         run,
		AttemptJobs: make(map[int][]*github.WorkflowJob),
	}

	// Get the workflow
	wfl, exists := workflowMap[*run.WorkflowID]
	if !exists {
		c.logger.Error().Int64("workflowID", *run.WorkflowID).Msg("workflow ID not found")
		return result, nil
	}
	result.Workflow = wfl

	// Get workflow run usage
	var usage *github.WorkflowRunUsage
	err := c.executeWithRateLimit(ctx, func() error {
		var err error
		usage, _, err = c.ghClient.Actions.GetWorkflowRunUsageByID(ctx, c.repo.Owner, c.repo.Name, *run.ID)
		return err
	})
	if err != nil {
		return result, err
	}
	result.UsageData = usage

	// Get jobs for the current run attempt
	var jobs *github.Jobs
	err = c.executeWithRateLimit(ctx, func() error {
		var err error
		jobs, err = c.ListWorkflowJobs(ctx, *run.ID)
		return err
	})
	if err != nil {
		return result, err
	}
	result.Jobs = jobs.Jobs

	// Get jobs for previous run attempts if there are any
	if *run.RunAttempt > 1 {
		for i := 1; i < int(*run.RunAttempt); i++ {
			attemptNum := int64(i)
			var attemptJobs *github.Jobs

			err := c.executeWithRateLimit(ctx, func() error {
				var err error
				attemptJobs, err = c.ListWorkflowJobsAttempt(ctx, *run.ID, attemptNum)
				return err
			})
			if err != nil {
				return result, err
			}

			result.AttemptJobs[i] = attemptJobs.Jobs
		}
	}

	return result, nil
}
