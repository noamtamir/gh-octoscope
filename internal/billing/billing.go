package billing

import (
	"fmt"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
)

// RunnerType represents the type of GitHub Actions runner
type RunnerType string

// RunnerDuration represents the duration a runner was used
type RunnerDuration struct {
	Runner   string
	Duration *int64
}

const (
	RunnerUbuntu          RunnerType = "UBUNTU"
	RunnerWindows         RunnerType = "WINDOWS"
	RunnerMacOS           RunnerType = "MACOS"
	RunnerLinux4Core      RunnerType = "LINUX_4_CORE"
	RunnerLinux8Core      RunnerType = "LINUX_8_CORE"
	RunnerLinux16Core     RunnerType = "LINUX_16_CORE"
	RunnerLinux32Core     RunnerType = "LINUX_32_CORE"
	RunnerLinux64Core     RunnerType = "LINUX_64_CORE"
	RunnerLinux4CoreGPU   RunnerType = "LINUX_4_CORE_GPU"
	RunnerWindows4Core    RunnerType = "WINDOWS_4_CORE"
	RunnerWindows8Core    RunnerType = "WINDOWS_8_CORE"
	RunnerWindows16Core   RunnerType = "WINDOWS_16_CORE"
	RunnerWindows32Core   RunnerType = "WINDOWS_32_CORE"
	RunnerWindows64Core   RunnerType = "WINDOWS_64_CORE"
	RunnerWindows4CoreGPU RunnerType = "WINDOWS_4_CORE_GPU"
	RunnerMacOS12Core     RunnerType = "MACOS_12_CORE"
	RunnerMacOS6CoreM1    RunnerType = "MACOS_6_CORE_M1"
	RunnerSelfHosted      RunnerType = "SELF_HOSTED"
)

// PriceConfig holds the pricing configuration for different runner types
type PriceConfig struct {
	Prices map[RunnerType]float64
}

// DefaultPriceConfig returns the default pricing configuration
func DefaultPriceConfig() *PriceConfig {
	return &PriceConfig{
		Prices: map[RunnerType]float64{
			RunnerUbuntu:          0.008,
			RunnerWindows:         0.016,
			RunnerMacOS:           0.08,
			RunnerLinux4Core:      0.016,
			RunnerLinux8Core:      0.032,
			RunnerLinux16Core:     0.064,
			RunnerLinux32Core:     0.128,
			RunnerLinux64Core:     0.256,
			RunnerLinux4CoreGPU:   0.07,
			RunnerWindows4Core:    0.032,
			RunnerWindows8Core:    0.064,
			RunnerWindows16Core:   0.128,
			RunnerWindows32Core:   0.256,
			RunnerWindows64Core:   0.512,
			RunnerWindows4CoreGPU: 0.014,
			RunnerMacOS12Core:     0.12,
			RunnerMacOS6CoreM1:    0.16,
			RunnerSelfHosted:      0,
		},
	}
}

// Calculator handles billing calculations for GitHub Actions
type Calculator struct {
	priceConfig *PriceConfig
	logger      zerolog.Logger
}

// NewCalculator creates a new billing calculator
func NewCalculator(cfg *PriceConfig, logger zerolog.Logger) *Calculator {
	if cfg == nil {
		cfg = DefaultPriceConfig()
	}
	return &Calculator{
		priceConfig: cfg,
		logger:      logger,
	}
}

// JobCost represents the cost details for a workflow job
type JobCost struct {
	ActualDuration   time.Duration
	BillableDuration time.Duration
	PricePerMinute   float64
	TotalBillableUSD float64
}

// CalculateJobCost calculates the cost for a specific job
func (c *Calculator) CalculateJobCost(job *github.WorkflowJob, runnerType RunnerType) (*JobCost, error) {
	if job.CompletedAt == nil || job.CreatedAt == nil {
		return nil, fmt.Errorf("job timing information is incomplete")
	}

	duration := job.CompletedAt.Sub(job.CreatedAt.Time)
	rounded := c.roundUpToMinute(duration)
	pricePerMinute := c.getPricePerMinute(runnerType)
	billable := c.calculateBillablePrice(pricePerMinute, rounded)

	// Handle cancelled jobs with zero duration
	if duration == 0 && *job.Conclusion == "cancelled" {
		return &JobCost{}, nil
	}

	return &JobCost{
		ActualDuration:   duration,
		BillableDuration: rounded,
		PricePerMinute:   pricePerMinute,
		TotalBillableUSD: billable,
	}, nil
}

func (c *Calculator) roundUpToMinute(d time.Duration) time.Duration {
	const (
		minute     = 60
		halfMinute = 30
	)

	secondsPortion := int(d.Seconds()) % minute
	rounded := d.Round(time.Minute)

	// Only add an extra minute if there's a partial minute and it's less than half
	if secondsPortion > 0 && secondsPortion < halfMinute {
		rounded += time.Minute
	}
	return rounded
}

func (c *Calculator) getPricePerMinute(runner RunnerType) float64 {
	if runner == RunnerSelfHosted {
		c.logger.Debug().Msg("self-hosted runners are not supported")
		return 0
	}

	price, exists := c.priceConfig.Prices[runner]
	if !exists {
		c.logger.Debug().Str("runner", string(runner)).Msg("unable to determine runner type")
		return 0
	}

	return price
}

func (c *Calculator) calculateBillablePrice(pricePerMinute float64, duration time.Duration) float64 {
	return pricePerMinute * duration.Minutes()
}
