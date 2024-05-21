package main

import (
	"time"

	"github.com/google/go-github/v62/github"
)

const MINUTE int = 60
const HALF_MINUTE int = 30
const SELF_HOSTED = "SELF_HOSTED"

// prices in USD
// todo: verify names!
var PRICES = map[string]float64{
	"UBUNTU":             0.008, // 2-core
	"WINDOWS":            0.016, // 2-core
	"MACOS":              0.08,  // 3-core
	"LINUX_4_CORE":       0.016,
	"LINUX_8_CORE":       0.032,
	"LINUX_16_CORE":      0.064,
	"LINUX_32_CORE":      0.128,
	"LINUX_64_CORE":      0.256,
	"LINUX_4_CORE_GPU":   0.07,
	"WINDOWS_4_CORE":     0.032,
	"WINDOWS_8_CORE":     0.064,
	"WINDOWS_16_CORE":    0.128,
	"WINDOWS_32_CORE":    0.256,
	"WINDOWS_64_CORE":    0.512,
	"WINDOWS_4_CORE_GPU": 0.014,
	"MACOS_12_CORE":      0.12,
	"MACOS_6_CORE_M1":    0.16,
	SELF_HOSTED:          0, // not supported
}

func calculateJobDuration(job *github.WorkflowJob) time.Duration {
	return job.CompletedAt.Sub(job.CreatedAt.Time)
}

func roundUpToClosestMinute(d time.Duration) time.Duration {
	secondsPortion := int(d.Seconds()) % MINUTE
	rounded := d.Round(time.Minute)   // rounds to closest minute
	if secondsPortion < HALF_MINUTE { // compensates for rounding down
		rounded += time.Minute
	}
	return rounded
}

func getPricePerMinute(runner string) float64 {
	// todo: fix this...
	if runner == SELF_HOSTED {
		logger.Debug().Msg("self-hosted runners are not supported at this moment")
		return 0
	}

	price, exists := PRICES[runner]
	if !exists {
		logger.Debug().Msg("Unable to determine runner type")
		return 0
	}

	return price
}

func calculateBillablePrice(pricePerMinute float64, duration time.Duration) float64 {
	return pricePerMinute * duration.Minutes()
}

func CalculateBillablePrice(job *github.WorkflowJob, runner string) (time.Duration, time.Duration, float64, float64) {
	duration := calculateJobDuration(job)
	rounded := roundUpToClosestMinute(duration)
	pricePerMinute := getPricePerMinute(runner)
	billable := calculateBillablePrice(pricePerMinute, rounded)
	return duration, rounded, pricePerMinute, billable
}
