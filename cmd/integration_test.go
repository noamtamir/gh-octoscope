// +build integration

package cmd_test

import (
	"os"
	"testing"

	"github.com/noamtamir/gh-octoscope/cmd"
	"github.com/rogpeppe/go-internal/testscript"
)

// TestMain sets up the testscript environment
func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		// Register the CLI binary for testscript
		"gh-octoscope": mainWithExitCode,
	}))
}

// mainWithExitCode wraps the cmd.Execute() to return an exit code
// instead of calling os.Exit, which is required by testscript
func mainWithExitCode() int {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

// TestScripts runs all testscript files in testdata/script/
func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(env *testscript.Env) error {
			// You can set up environment variables here
			// env.Setenv("OCTOSCOPE_API_URL", "http://test.example.com")
			return nil
		},
	})
}
