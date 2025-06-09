package billing

import (
	"io"
	"testing"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestDetermineRunnerTypeFromLabels(t *testing.T) {
	// Create a silent logger for tests
	logger := zerolog.New(io.Discard)

	tests := []struct {
		name           string
		labels         []string
		expectedRunner RunnerType
	}{
		// Self-hosted runner tests
		{
			name:           "Self-hosted runner",
			labels:         []string{"self-hosted", "linux", "ubuntu-22.04"},
			expectedRunner: RunnerSelfHosted,
		},
		{
			name:           "Self-hosted runner with custom labels",
			labels:         []string{"self-hosted", "windows", "gpu"},
			expectedRunner: RunnerSelfHosted,
		},

		// Standard runners
		{
			name:           "Ubuntu latest",
			labels:         []string{"ubuntu-latest"},
			expectedRunner: RunnerUbuntu,
		},
		{
			name:           "Ubuntu 22.04",
			labels:         []string{"ubuntu-22.04"},
			expectedRunner: RunnerUbuntu,
		},
		{
			name:           "Ubuntu 24.04",
			labels:         []string{"ubuntu-24.04"},
			expectedRunner: RunnerUbuntu,
		},
		{
			name:           "Windows latest",
			labels:         []string{"windows-latest"},
			expectedRunner: RunnerWindows,
		},
		{
			name:           "macOS latest",
			labels:         []string{"macos-latest"},
			expectedRunner: RunnerMacOS,
		},

		// Large runners
		{
			name:           "Ubuntu 4 cores",
			labels:         []string{"ubuntu-latest-4-cores"},
			expectedRunner: RunnerLinux4Core,
		},
		{
			name:           "Ubuntu 8 cores",
			labels:         []string{"ubuntu-20.04-8-cores"},
			expectedRunner: RunnerLinux8Core,
		},
		{
			name:           "Windows 16 cores",
			labels:         []string{"windows-2022-16-cores"},
			expectedRunner: RunnerWindows16Core,
		},

		// ARM64 runners
		{
			name:           "Ubuntu ARM 4 cores",
			labels:         []string{"ubuntu-latest-4-cores-arm64"},
			expectedRunner: RunnerLinux4CoreARM,
		},
		{
			name:           "Windows ARM 8 cores",
			labels:         []string{"windows-2022-8-cores-arm64"},
			expectedRunner: RunnerWindows8CoreARM,
		},
		{
			name:           "macOS M1",
			labels:         []string{"macos-latest-6-core"},
			expectedRunner: RunnerMacOS6CoreM1,
		},

		// GPU runners
		{
			name:           "Ubuntu with GPU",
			labels:         []string{"ubuntu-latest-4-cores-gpu"},
			expectedRunner: RunnerLinux4CoreGPU,
		},
		{
			name:           "Windows with GPU",
			labels:         []string{"windows-2022-4-cores-gpu"},
			expectedRunner: RunnerWindows4CoreGPU,
		},

		// Fallback cases
		{
			name:           "Unknown Ubuntu variant",
			labels:         []string{"ubuntu-custom"},
			expectedRunner: RunnerUbuntu,
		},
		{
			name:           "Unknown Windows variant",
			labels:         []string{"windows-special"},
			expectedRunner: RunnerWindows,
		},
		{
			name:           "Unknown macOS variant",
			labels:         []string{"macos-custom"},
			expectedRunner: RunnerMacOS,
		},
		{
			name:           "Empty labels",
			labels:         []string{},
			expectedRunner: RunnerUbuntu,
		},
		{
			name:           "Nil labels",
			labels:         nil,
			expectedRunner: RunnerUbuntu,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test job with labels
			job := &github.WorkflowJob{
				Labels: tt.labels,
			}

			// Determine runner type
			runnerType := DetermineRunnerTypeFromLabels(job, logger)

			// Assert expected runner type
			assert.Equal(t, tt.expectedRunner, runnerType)
		})
	}
}

func TestCalculateJobCost(t *testing.T) {
	// Create a silent logger for tests
	logger := zerolog.New(io.Discard)

	// Create the calculator with default pricing
	calculator := NewCalculator(nil, logger)

	// Test job
	conclusion := "success"
	createdAt := time.Date(2025, 4, 25, 10, 0, 0, 0, time.UTC)
	completedAt := time.Date(2025, 4, 25, 10, 5, 0, 0, time.UTC) // 5 minutes later

	job := &github.WorkflowJob{
		Conclusion:  &conclusion,
		CreatedAt:   &github.Timestamp{Time: createdAt},
		CompletedAt: &github.Timestamp{Time: completedAt},
		Labels:      []string{"ubuntu-latest"},
	}

	// Calculate cost
	cost, runnerType, err := calculator.CalculateJobCost(job)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, RunnerUbuntu, runnerType)
	assert.Equal(t, 5*time.Minute, cost.ActualDuration)
	assert.Equal(t, 5*time.Minute, cost.BillableDuration)
	assert.Equal(t, 0.008, cost.PricePerMinute)
	assert.Equal(t, 0.04, cost.TotalBillableUSD) // 0.008 * 5
}
