package billing

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestCalculateJobCost(t *testing.T) {
	// Create a silent logger for tests
	logger := zerolog.New(ioutil.Discard)

	// Create the calculator with default pricing
	calculator := NewCalculator(nil, logger)

	// Common test variables
	conclusion := "success"
	createdAt := time.Date(2025, 4, 25, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		job              *github.WorkflowJob
		runnerType       RunnerType
		expectedCost     float64
		expectedError    bool
		actualDuration   time.Duration
		billableDuration time.Duration
	}{
		{
			name: "Ubuntu runner with 5-minute job",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: createdAt},
				CompletedAt: &github.Timestamp{Time: createdAt.Add(5 * time.Minute)},
				Conclusion:  &conclusion,
			},
			runnerType:       RunnerUbuntu,
			expectedCost:     0.04, // 5 minutes * $0.008/minute
			expectedError:    false,
			actualDuration:   5 * time.Minute,
			billableDuration: 5 * time.Minute,
		},
		{
			name: "Windows runner with 2.5-minute job",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: createdAt},
				CompletedAt: &github.Timestamp{Time: createdAt.Add(2*time.Minute + 30*time.Second)},
				Conclusion:  &conclusion,
			},
			runnerType:       RunnerWindows,
			expectedCost:     0.048, // 3 minutes (rounded up) * $0.016/minute
			expectedError:    false,
			actualDuration:   2*time.Minute + 30*time.Second,
			billableDuration: 3 * time.Minute,
		},
		{
			name: "MacOS runner with 1-minute job",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: createdAt},
				CompletedAt: &github.Timestamp{Time: createdAt.Add(1 * time.Minute)},
				Conclusion:  &conclusion,
			},
			runnerType:       RunnerMacOS,
			expectedCost:     0.08,
			expectedError:    false,
			actualDuration:   1 * time.Minute,
			billableDuration: 1 * time.Minute,
		},
		{
			name: "Self-hosted runner with 10-minute job",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: createdAt},
				CompletedAt: &github.Timestamp{Time: createdAt.Add(10 * time.Minute)},
				Conclusion:  &conclusion,
			},
			runnerType:       RunnerSelfHosted,
			expectedCost:     0, // Self-hosted runners have no cost
			expectedError:    false,
			actualDuration:   10 * time.Minute,
			billableDuration: 10 * time.Minute,
		},
		{
			name: "Linux64Core runner with 30-second job",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: createdAt},
				CompletedAt: &github.Timestamp{Time: createdAt.Add(30 * time.Second)},
				Conclusion:  &conclusion,
			},
			runnerType:       RunnerLinux64Core,
			expectedCost:     0.256, // 1 minute (rounded up) * $0.256/minute
			expectedError:    false,
			actualDuration:   30 * time.Second,
			billableDuration: 1 * time.Minute,
		},
		{
			name: "Invalid runner type",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: createdAt},
				CompletedAt: &github.Timestamp{Time: createdAt.Add(5 * time.Minute)},
				Conclusion:  &conclusion,
			},
			runnerType:       RunnerType("INVALID_RUNNER"),
			expectedCost:     0, // Invalid runner types return 0 cost
			expectedError:    false,
			actualDuration:   5 * time.Minute,
			billableDuration: 5 * time.Minute,
		},
		{
			name: "Missing completion time",
			job: &github.WorkflowJob{
				CreatedAt:  &github.Timestamp{Time: createdAt},
				Conclusion: &conclusion,
			},
			runnerType:    RunnerUbuntu,
			expectedError: true,
		},
		{
			name: "Missing creation time",
			job: &github.WorkflowJob{
				CompletedAt: &github.Timestamp{Time: createdAt.Add(5 * time.Minute)},
				Conclusion:  &conclusion,
			},
			runnerType:    RunnerUbuntu,
			expectedError: true,
		},
		{
			name: "Cancelled job with zero duration",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: createdAt},
				CompletedAt: &github.Timestamp{Time: createdAt}, // Same time = 0 duration
				Conclusion:  github.String("cancelled"),
			},
			runnerType:    RunnerUbuntu,
			expectedCost:  0,
			expectedError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cost, err := calculator.CalculateJobCost(tc.job, tc.runnerType)

			if tc.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.InDelta(t, tc.expectedCost, cost.TotalBillableUSD, 0.001)

			if tc.actualDuration > 0 {
				assert.Equal(t, tc.actualDuration, cost.ActualDuration)
				assert.Equal(t, tc.billableDuration, cost.BillableDuration)
			}
		})
	}
}

func TestRoundUpToMinute(t *testing.T) {
	logger := zerolog.New(ioutil.Discard)
	calculator := NewCalculator(nil, logger)

	tests := []struct {
		name     string
		duration time.Duration
		expected time.Duration
	}{
		{
			name:     "Exactly 1 minute",
			duration: time.Minute,
			expected: time.Minute,
		},
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			expected: time.Minute,
		},
		{
			name:     "29 seconds",
			duration: 29 * time.Second,
			expected: time.Minute,
		},
		{
			name:     "1 minute 29 seconds",
			duration: time.Minute + 29*time.Second,
			expected: 2 * time.Minute,
		},
		{
			name:     "1 minute 31 seconds",
			duration: time.Minute + 31*time.Second,
			expected: 2 * time.Minute,
		},
		{
			name:     "2 minutes 15 seconds",
			duration: 2*time.Minute + 15*time.Second,
			expected: 3 * time.Minute,
		},
		{
			name:     "Zero duration",
			duration: 0,
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculator.roundUpToMinute(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetPricePerMinute(t *testing.T) {
	logger := zerolog.New(ioutil.Discard)

	// Test with default pricing
	t.Run("DefaultPricing", func(t *testing.T) {
		calculator := NewCalculator(nil, logger)

		// Check a few runner types
		assert.Equal(t, 0.008, calculator.getPricePerMinute(RunnerUbuntu))
		assert.Equal(t, 0.016, calculator.getPricePerMinute(RunnerWindows))
		assert.Equal(t, 0.08, calculator.getPricePerMinute(RunnerMacOS))
		assert.Equal(t, 0.0, calculator.getPricePerMinute(RunnerSelfHosted))
		assert.Equal(t, 0.0, calculator.getPricePerMinute(RunnerType("UNKNOWN")))
	})

	// Test with custom pricing
	t.Run("CustomPricing", func(t *testing.T) {
		customPricing := &PriceConfig{
			Prices: map[RunnerType]float64{
				RunnerUbuntu:  0.01, // Custom price
				RunnerWindows: 0.02, // Custom price
				RunnerMacOS:   0.1,  // Custom price
			},
		}

		calculator := NewCalculator(customPricing, logger)

		assert.Equal(t, 0.01, calculator.getPricePerMinute(RunnerUbuntu))
		assert.Equal(t, 0.02, calculator.getPricePerMinute(RunnerWindows))
		assert.Equal(t, 0.1, calculator.getPricePerMinute(RunnerMacOS))

		// These weren't in our custom config, so they should return 0
		assert.Equal(t, 0.0, calculator.getPricePerMinute(RunnerLinux4Core))
		assert.Equal(t, 0.0, calculator.getPricePerMinute(RunnerSelfHosted))
	})
}

func TestCalculateBillablePrice(t *testing.T) {
	logger := zerolog.New(ioutil.Discard)
	calculator := NewCalculator(nil, logger)

	tests := []struct {
		name           string
		pricePerMinute float64
		duration       time.Duration
		expected       float64
	}{
		{
			name:           "1 minute at $0.008",
			pricePerMinute: 0.008,
			duration:       time.Minute,
			expected:       0.008,
		},
		{
			name:           "3 minutes at $0.016",
			pricePerMinute: 0.016,
			duration:       3 * time.Minute,
			expected:       0.048,
		},
		{
			name:           "5 minutes at $0.08",
			pricePerMinute: 0.08,
			duration:       5 * time.Minute,
			expected:       0.4,
		},
		{
			name:           "Zero price",
			pricePerMinute: 0,
			duration:       10 * time.Minute,
			expected:       0,
		},
		{
			name:           "Zero duration",
			pricePerMinute: 0.008,
			duration:       0,
			expected:       0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculator.calculateBillablePrice(tc.pricePerMinute, tc.duration)
			assert.InDelta(t, tc.expected, result, 0.0001)
		})
	}
}
