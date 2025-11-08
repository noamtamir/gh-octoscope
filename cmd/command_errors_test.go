package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommandExecutionErrors tests command error handling without building a binary
// This is the idiomatic Go way to test CLI commands
func TestCommandExecutionErrors(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Invalid date format in fetch command",
			args:          []string{"fetch", "--from=not-a-date"},
			expectError:   true,
			errorContains: "parsing time",
		},
		{
			name:          "Invalid date format in report command",
			args:          []string{"report", "--from=2024-13-45"},
			expectError:   true,
			errorContains: "parsing time",
		},
		{
			name:          "Missing data directory in report-only mode",
			args:          []string{"report", "--fetch=false"},
			expectError:   true,
			errorContains: "does not exist",
		},
		{
			name:          "Delete without report ID",
			args:          []string{"report", "delete"},
			expectError:   true,
			errorContains: "accepts 1 arg(s), received 0",
		},
		{
			name:          "Version command succeeds",
			args:          []string{"version"},
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh root command for each test
			rootCmd := NewRootCmd()

			// Capture output
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)

			// Set the args
			rootCmd.SetArgs(tc.args)

			// Execute the command
			err := rootCmd.Execute()

			if tc.expectError {
				require.Error(t, err, "Expected an error but got none")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains,
						"Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
			}
		})
	}
}
