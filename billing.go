package main

import (
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
)

// OS names
const UBUNTU string = "ubuntu"
const MACOS string = "macos"
const WINDOWS string = "windows"

// Price per minute in USD
const UBUNTU_PRICE_PER_MINUTE float64 = 0.008  // Ubuntu 2-core
const WINDOWS_PRICE_PER_MINUTE float64 = 0.016 // Windows 2-core
const MACOS_PRICE_PER_MINUTE float64 = 0.08    // macOS 3-core
const MINUTE int = 60
const HALF_MINUTE int = 30

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

func getPricePerMinute(labels []string) float64 {
	// todo: make this better... perhaps use fuzzy search library?
	// support large runners, support multiple labels...
	// only supporting standard runners for now, assuming 1 label
	first := labels[0]
	switch {
	case strings.HasPrefix(first, UBUNTU):
		return UBUNTU_PRICE_PER_MINUTE
	case strings.HasPrefix(first, MACOS):
		return MACOS_PRICE_PER_MINUTE
	case strings.HasPrefix(first, WINDOWS):
		return WINDOWS_PRICE_PER_MINUTE
	case strings.HasPrefix(first, "self-hosted"):
		return 0 // does not support self hosted runners!
	default:
		return UBUNTU_PRICE_PER_MINUTE
	}
}

func calculateBillablePrice(pricePerMinute float64, duration time.Duration) float64 {
	return pricePerMinute * duration.Minutes()
}

func CalculateBillablePrice(job *github.WorkflowJob) (time.Duration, time.Duration, float64, float64) {
	duration := calculateJobDuration(job)
	rounded := roundUpToClosestMinute(duration)
	pricePerMinute := getPricePerMinute(job.Labels)
	billable := calculateBillablePrice(pricePerMinute, rounded)
	return duration, rounded, pricePerMinute, billable
}
