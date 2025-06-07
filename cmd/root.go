package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// Config holds application configuration
type Config struct {
	Debug      bool
	ProdLogger bool
	FullReport bool
	CSVReport  bool
	HTMLReport bool
	FromDate   string
	PageSize   int
	Fetch      bool
	Obfuscate  bool
}

// GitHubCLIConfig holds GitHub CLI configuration
type GitHubCLIConfig struct {
	Token string
	Repo  repository.Repository
}

var (
	// Config that will be used throughout the application
	cfg = Config{
		PageSize: 30,
		Fetch:    true,
	}

	// Version information
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// NewRootCmd creates and returns the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gh-octoscope",
		Short: "Calculate GitHub Actions usage and costs",
		Long: `gh-octoscope analyzes GitHub Actions workflows and generates usage and cost reports.
It fetches workflow run data from the GitHub API and calculates the cost based
on the runner types used.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load environment variables from .env file
			if err := godotenv.Load(); err != nil {
				// This is expected in production, so just log in debug mode
				logger := setupLogger()
				logger.Debug().Msg(".env file not found, expected when not running in development")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get GitHub CLI configuration
			host, _ := auth.DefaultHost()
			token, _ := auth.TokenForHost(host)
			repo, err := repository.Current()
			if err != nil {
				return fmt.Errorf("failed to get current repository: %w", err)
			}

			ghCLIConfig := GitHubCLIConfig{
				Token: token,
				Repo:  repo,
			}

			// Run the application
			return Run(cfg, ghCLIConfig)
		},
	}

	// Add persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().BoolVar(&cfg.Debug, "debug", false, "Sets log level to debug")
	rootCmd.PersistentFlags().BoolVar(&cfg.ProdLogger, "prod-log", false, "Production structured log")
	rootCmd.PersistentFlags().BoolVar(&cfg.Fetch, "fetch", true, "Fetch new data (set to false to use existing data)")
	rootCmd.PersistentFlags().StringVar(&cfg.FromDate, "from", "", "Generate report from this date. Format: YYYY-MM-DD")
	rootCmd.PersistentFlags().IntVar(&cfg.PageSize, "page-size", 30, "Page size for GitHub API requests")

	// For backward compatibility, keep a few flags in root command
	rootCmd.Flags().BoolVar(&cfg.FullReport, "report", false, "Generate full report (same as using the 'report' command)")
	rootCmd.Flags().BoolVar(&cfg.CSVReport, "csv", false, "Generate csv report (same as using 'report --csv')")
	rootCmd.Flags().BoolVar(&cfg.HTMLReport, "html", false, "Generate html report (same as using 'report --html')")
	rootCmd.Flags().BoolVar(&cfg.Obfuscate, "obfuscate", false, "Obfuscate sensitive data in the report")

	// Set version template
	rootCmd.SetVersionTemplate(`Version: {{.Version}}
Commit: ` + commit + `
Built: ` + date + `
`)

	// Add subcommands
	rootCmd.AddCommand(
		newCompletionCmd(),
		newVersionCmd(),
		newReportCmd(),
		newFetchCmd(),
	)

	return rootCmd
}

// setupLogger creates and configures a logger based on the application configuration
func setupLogger() zerolog.Logger {
	var writer io.Writer
	if cfg.ProdLogger {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		writer = os.Stdout
	} else {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	logger := zerolog.New(writer).With().Timestamp().Logger()
	if cfg.Debug {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	return logger
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once.
func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
