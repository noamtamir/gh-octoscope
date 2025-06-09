package billing

import (
	"fmt"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
)

type RunnerType string

const (
	// Standard runners
	RunnerUbuntu  RunnerType = "UBUNTU"  // ubuntu-latest, ubuntu-22.04, ubuntu-24.04
	RunnerWindows RunnerType = "WINDOWS" // windows-latest, windows-2022, windows-2025
	RunnerMacOS   RunnerType = "MACOS"   // macos-latest, macos-13, macos-14, macos-15

	// Linux large runners
	RunnerLinux4Core  RunnerType = "LINUX_4_CORE"  // ubuntu-*-4-cores
	RunnerLinux8Core  RunnerType = "LINUX_8_CORE"  // ubuntu-*-8-cores
	RunnerLinux16Core RunnerType = "LINUX_16_CORE" // ubuntu-*-16-cores
	RunnerLinux32Core RunnerType = "LINUX_32_CORE" // ubuntu-*-32-cores
	RunnerLinux64Core RunnerType = "LINUX_64_CORE" // ubuntu-*-64-cores
	RunnerLinux96Core RunnerType = "LINUX_96_CORE" // ubuntu-*-96-cores

	// Linux ARM runners
	RunnerLinux4CoreARM  RunnerType = "LINUX_4_CORE_ARM"  // ubuntu-*-4-cores-arm64
	RunnerLinux8CoreARM  RunnerType = "LINUX_8_CORE_ARM"  // ubuntu-*-8-cores-arm64
	RunnerLinux16CoreARM RunnerType = "LINUX_16_CORE_ARM" // ubuntu-*-16-cores-arm64
	RunnerLinux32CoreARM RunnerType = "LINUX_32_CORE_ARM" // ubuntu-*-32-cores-arm64
	RunnerLinux64CoreARM RunnerType = "LINUX_64_CORE_ARM" // ubuntu-*-64-cores-arm64

	// GPU runners
	RunnerLinux4CoreGPU RunnerType = "LINUX_4_CORE_GPU" // ubuntu-*-4-cores-gpu

	// Windows runners
	RunnerWindows4Core  RunnerType = "WINDOWS_4_CORE"  // windows-*-4-cores
	RunnerWindows8Core  RunnerType = "WINDOWS_8_CORE"  // windows-*-8-cores
	RunnerWindows16Core RunnerType = "WINDOWS_16_CORE" // windows-*-16-cores
	RunnerWindows32Core RunnerType = "WINDOWS_32_CORE" // windows-*-32-cores
	RunnerWindows64Core RunnerType = "WINDOWS_64_CORE" // windows-*-64-cores
	RunnerWindows96Core RunnerType = "WINDOWS_96_CORE" // windows-*-96-cores

	// Windows ARM runners
	RunnerWindows4CoreARM  RunnerType = "WINDOWS_4_CORE_ARM"  // windows-*-4-cores-arm64
	RunnerWindows8CoreARM  RunnerType = "WINDOWS_8_CORE_ARM"  // windows-*-8-cores-arm64
	RunnerWindows16CoreARM RunnerType = "WINDOWS_16_CORE_ARM" // windows-*-16-cores-arm64
	RunnerWindows32CoreARM RunnerType = "WINDOWS_32_CORE_ARM" // windows-*-32-cores-arm64
	RunnerWindows64CoreARM RunnerType = "WINDOWS_64_CORE_ARM" // windows-*-64-cores-arm64

	// Windows GPU runners
	RunnerWindows4CoreGPU RunnerType = "WINDOWS_4_CORE_GPU" // windows-*-4-cores-gpu

	// macOS special runners
	RunnerMacOS12Core  RunnerType = "MACOS_12_CORE"   // macos-*-12-cores
	RunnerMacOS6CoreM1 RunnerType = "MACOS_6_CORE_M1" // macos-*-6-core

	// Self-hosted runners
	RunnerSelfHosted RunnerType = "SELF_HOSTED" // Any runner with "self-hosted" label
)

type PriceConfig struct {
	Prices map[RunnerType]float64
}

func DefaultPriceConfig() *PriceConfig {
	return &PriceConfig{
		Prices: map[RunnerType]float64{
			// Standard GitHub-Hosted Runners (2-core)
			RunnerUbuntu:  0.008,
			RunnerWindows: 0.016,
			RunnerMacOS:   0.080,

			// Large x64 Runners
			RunnerLinux4Core:    0.016,
			RunnerLinux8Core:    0.032,
			RunnerLinux16Core:   0.064,
			RunnerLinux32Core:   0.128,
			RunnerLinux64Core:   0.256,
			RunnerLinux96Core:   0.384,
			RunnerWindows4Core:  0.032,
			RunnerWindows8Core:  0.064,
			RunnerWindows16Core: 0.128,
			RunnerWindows32Core: 0.256,
			RunnerWindows64Core: 0.512,
			RunnerWindows96Core: 0.768,
			RunnerMacOS12Core:   0.120,

			// Large ARM64 Runners
			RunnerLinux4CoreARM:    0.010,
			RunnerLinux8CoreARM:    0.020,
			RunnerLinux16CoreARM:   0.040,
			RunnerLinux32CoreARM:   0.080,
			RunnerLinux64CoreARM:   0.160,
			RunnerWindows4CoreARM:  0.020,
			RunnerWindows8CoreARM:  0.040,
			RunnerWindows16CoreARM: 0.080,
			RunnerWindows32CoreARM: 0.160,
			RunnerWindows64CoreARM: 0.320,
			RunnerMacOS6CoreM1:     0.160,

			// GPU Runners
			RunnerLinux4CoreGPU:   0.070,
			RunnerWindows4CoreGPU: 0.140,

			// Self-hosted runners
			RunnerSelfHosted: 0,
		},
	}
}

type Calculator struct {
	priceConfig *PriceConfig
	logger      zerolog.Logger
}

func NewCalculator(cfg *PriceConfig, logger zerolog.Logger) *Calculator {
	if cfg == nil {
		cfg = DefaultPriceConfig()
	}
	return &Calculator{
		priceConfig: cfg,
		logger:      logger,
	}
}

type JobCost struct {
	ActualDuration   time.Duration
	BillableDuration time.Duration
	PricePerMinute   float64
	TotalBillableUSD float64
}

// CalculateJobCost calculates the job cost by first determining the runner type from job labels
func (c *Calculator) CalculateJobCost(job *github.WorkflowJob) (*JobCost, RunnerType, error) {
	if job.CompletedAt == nil || job.CreatedAt == nil {
		return nil, "", fmt.Errorf("job timing information is incomplete")
	}

	// Determine runner type based on job labels
	runnerType := DetermineRunnerTypeFromLabels(job, c.logger)

	duration := job.CompletedAt.Sub(job.CreatedAt.Time)
	rounded := c.roundUpToMinute(duration)
	pricePerMinute := c.getPricePerMinute(runnerType)
	billable := c.calculateBillablePrice(pricePerMinute, rounded)

	// Handle cancelled jobs with zero duration
	if duration == 0 && *job.Conclusion == "cancelled" {
		return &JobCost{}, runnerType, nil
	}

	return &JobCost{
		ActualDuration:   duration,
		BillableDuration: rounded,
		PricePerMinute:   pricePerMinute,
		TotalBillableUSD: billable,
	}, runnerType, nil
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
