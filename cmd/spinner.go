package cmd

import (
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// createSpinner creates a new spinner with the specified message
func createSpinner(message string) *spinner.Spinner {
	// Create a new spinner with character set 14 and speed of 100ms
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("cyan") // Use cyan color for the spinner
	return s
}

// createSuccessMessage returns a colored success message
func createSuccessMessage(message string) string {
	return color.GreenString("✓ ") + message
}

// createInfoMessage returns a colored info message
func createInfoMessage(message string) string {
	return color.CyanString("ℹ ") + message
}
