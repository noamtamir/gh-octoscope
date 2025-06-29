package billing

import (
	"io"
	"testing"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRoundUpToMinute(t *testing.T) {
	// Create a silent logger for tests
	logger := zerolog.New(io.Discard)

	// Create the calculator with default pricing
	calculator := NewCalculator(nil, logger)

	tests := []struct {
		name           string
		input          time.Duration
		expectedOutput time.Duration
	}{
		{
			name:           "Exactly 1 minute",
			input:          1 * time.Minute,
			expectedOutput: 1 * time.Minute,
		},
		{
			name:           "30 seconds",
			input:          30 * time.Second,
			expectedOutput: 1 * time.Minute,
		},
		{
			name:           "29 seconds",
			input:          29 * time.Second,
			expectedOutput: 1 * time.Minute,
		},
		{
			name:           "1 minute 29 seconds",
			input:          1*time.Minute + 29*time.Second,
			expectedOutput: 2 * time.Minute,
		},
		{
			name:           "1 minute 31 seconds",
			input:          1*time.Minute + 31*time.Second,
			expectedOutput: 2 * time.Minute,
		},
		{
			name:           "2 minutes 15 seconds",
			input:          2*time.Minute + 15*time.Second,
			expectedOutput: 3 * time.Minute,
		},
		{
			name:           "Zero duration",
			input:          0,
			expectedOutput: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.roundUpToMinute(tt.input)
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

func TestGetPricePerMinute(t *testing.T) {
	logger := zerolog.New(io.Discard)

	t.Run("DefaultPricing", func(t *testing.T) {
		calculator := NewCalculator(nil, logger)

		// Test a few standard prices
		assert.Equal(t, 0.008, calculator.getPricePerMinute(RunnerUbuntu))
		assert.Equal(t, 0.016, calculator.getPricePerMinute(RunnerWindows))
		assert.Equal(t, 0.080, calculator.getPricePerMinute(RunnerMacOS))
		assert.Equal(t, 0.0, calculator.getPricePerMinute(RunnerSelfHosted))
	})

	t.Run("CustomPricing", func(t *testing.T) {
		customPrices := &PriceConfig{
			Prices: map[RunnerType]float64{
				RunnerUbuntu:  0.01,
				RunnerWindows: 0.02,
				RunnerMacOS:   0.10,
			},
		}
		calculator := NewCalculator(customPrices, logger)

		// Test custom prices
		assert.Equal(t, 0.01, calculator.getPricePerMinute(RunnerUbuntu))
		assert.Equal(t, 0.02, calculator.getPricePerMinute(RunnerWindows))
		assert.Equal(t, 0.10, calculator.getPricePerMinute(RunnerMacOS))
		// Self-hosted still returns 0, even though not specified in custom map
		assert.Equal(t, 0.0, calculator.getPricePerMinute(RunnerSelfHosted))
	})
}

func TestCalculateBillablePrice(t *testing.T) {
	logger := zerolog.New(io.Discard)
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
			duration:       1 * time.Minute,
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
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCalculateJobCost_SpecialCases(t *testing.T) {
	logger := zerolog.New(io.Discard)
	calculator := NewCalculator(nil, logger)
	now := time.Now()

	tests := []struct {
		name        string
		job         *github.WorkflowJob
		expectsZero bool
	}{
		{
			name: "Skipped job",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: now},
				CompletedAt: &github.Timestamp{Time: now.Add(5 * time.Minute)},
				Conclusion:  github.String("skipped"),
				RunnerID:    github.Int64(1),
				Steps:       []*github.TaskStep{{}},
			},
			expectsZero: true,
		},
		{
			name: "Missing runner info",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: now},
				CompletedAt: &github.Timestamp{Time: now.Add(5 * time.Minute)},
				Conclusion:  github.String("success"),
				RunnerID:    nil,
				Steps:       []*github.TaskStep{{}},
			},
			expectsZero: true,
		},
		{
			name: "Empty steps",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: now},
				CompletedAt: &github.Timestamp{Time: now.Add(5 * time.Minute)},
				Conclusion:  github.String("success"),
				RunnerID:    github.Int64(1),
				Steps:       []*github.TaskStep{},
			},
			expectsZero: true,
		},
		{
			name: "Invalid timestamps (completed before created)",
			job: &github.WorkflowJob{
				CreatedAt:   &github.Timestamp{Time: now},
				CompletedAt: &github.Timestamp{Time: now.Add(-5 * time.Minute)},
				Conclusion:  github.String("success"),
				RunnerID:    github.Int64(1),
				Steps:       []*github.TaskStep{{}},
			},
			expectsZero: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cost, runner, err := calculator.CalculateJobCost(tc.job)
			assert.NoError(t, err)
			assert.Equal(t, 0*time.Duration(0), cost.ActualDuration)
			assert.Equal(t, 0*time.Duration(0), cost.BillableDuration)
			assert.Equal(t, 0.08, cost.PricePerMinute) // Default price for Ubuntu runner for skipped jobs
			assert.Equal(t, 0.0, cost.TotalBillableUSD)
			assert.Equal(t, RunnerUbuntu, runner)
		})
	}
}
