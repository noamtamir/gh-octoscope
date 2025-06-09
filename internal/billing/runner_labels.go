package billing

import (
	"regexp"
	"strings"

	"github.com/google/go-github/v62/github"
	"github.com/rs/zerolog"
)

// labelPatterns contains regex patterns for identifying runner types from labels
var labelPatterns = map[*regexp.Regexp]RunnerType{
	// Standard GitHub-Hosted Runners (2-core)
	regexp.MustCompile(`^ubuntu-(latest|2[24]\.04|2[24]\.04-arm)$`): RunnerUbuntu,
	regexp.MustCompile(`^windows-(latest|202[25]|11-arm)$`):         RunnerWindows,
	regexp.MustCompile(`^macos-(latest|1[345])$`):                   RunnerMacOS,

	// Large x64 Runners
	regexp.MustCompile(`^ubuntu-.*-4-cores$`):   RunnerLinux4Core,
	regexp.MustCompile(`^ubuntu-.*-8-cores$`):   RunnerLinux8Core,
	regexp.MustCompile(`^ubuntu-.*-16-cores$`):  RunnerLinux16Core,
	regexp.MustCompile(`^ubuntu-.*-32-cores$`):  RunnerLinux32Core,
	regexp.MustCompile(`^ubuntu-.*-64-cores$`):  RunnerLinux64Core,
	regexp.MustCompile(`^ubuntu-.*-96-cores$`):  RunnerLinux96Core,
	regexp.MustCompile(`^windows-.*-4-cores$`):  RunnerWindows4Core,
	regexp.MustCompile(`^windows-.*-8-cores$`):  RunnerWindows8Core,
	regexp.MustCompile(`^windows-.*-16-cores$`): RunnerWindows16Core,
	regexp.MustCompile(`^windows-.*-32-cores$`): RunnerWindows32Core,
	regexp.MustCompile(`^windows-.*-64-cores$`): RunnerWindows64Core,
	regexp.MustCompile(`^windows-.*-96-cores$`): RunnerWindows96Core,
	regexp.MustCompile(`^macos-.*-12-cores$`):   RunnerMacOS12Core,

	// Large ARM64 Runners
	regexp.MustCompile(`^ubuntu-.*-4-cores-arm64$`):   RunnerLinux4CoreARM,
	regexp.MustCompile(`^ubuntu-.*-8-cores-arm64$`):   RunnerLinux8CoreARM,
	regexp.MustCompile(`^ubuntu-.*-16-cores-arm64$`):  RunnerLinux16CoreARM,
	regexp.MustCompile(`^ubuntu-.*-32-cores-arm64$`):  RunnerLinux32CoreARM,
	regexp.MustCompile(`^ubuntu-.*-64-cores-arm64$`):  RunnerLinux64CoreARM,
	regexp.MustCompile(`^windows-.*-4-cores-arm64$`):  RunnerWindows4CoreARM,
	regexp.MustCompile(`^windows-.*-8-cores-arm64$`):  RunnerWindows8CoreARM,
	regexp.MustCompile(`^windows-.*-16-cores-arm64$`): RunnerWindows16CoreARM,
	regexp.MustCompile(`^windows-.*-32-cores-arm64$`): RunnerWindows32CoreARM,
	regexp.MustCompile(`^windows-.*-64-cores-arm64$`): RunnerWindows64CoreARM,
	regexp.MustCompile(`^macos-.*-6-core$`):           RunnerMacOS6CoreM1,

	// GPU Runners
	regexp.MustCompile(`^ubuntu-.*-4-cores-gpu$`):  RunnerLinux4CoreGPU,
	regexp.MustCompile(`^windows-.*-4-cores-gpu$`): RunnerWindows4CoreGPU,
}

// DetermineRunnerTypeFromLabels analyzes job labels to determine the runner type
// It first checks if the job ran on a self-hosted runner
// If not, it matches labels against known patterns to identify the specific runner type
func DetermineRunnerTypeFromLabels(job *github.WorkflowJob, logger zerolog.Logger) RunnerType {
	if job == nil || job.Labels == nil {
		logger.Debug().Msg("job or labels are nil, defaulting to Ubuntu runner")
		return RunnerUbuntu // Default to basic Ubuntu runner if no labels
	}

	// First pass: check for self-hosted runner
	for _, label := range job.Labels {
		if label == "self-hosted" {
			return RunnerSelfHosted
		}
	}

	// Second pass: match against known patterns
	for _, label := range job.Labels {
		for pattern, runnerType := range labelPatterns {
			if pattern.MatchString(label) {
				return runnerType
			}
		}
	}

	// Fallback logic based on common label prefixes
	for _, label := range job.Labels {
		switch {
		case strings.HasPrefix(label, "ubuntu"):
			logger.Debug().
				Strs("labels", job.Labels).
				Str("fallback_label", label).
				Msg("fallback to ubuntu runner")
			return RunnerUbuntu
		case strings.HasPrefix(label, "windows"):
			logger.Debug().
				Strs("labels", job.Labels).
				Str("fallback_label", label).
				Msg("fallback to windows runner")
			return RunnerWindows
		case strings.HasPrefix(label, "macos"):
			logger.Debug().
				Strs("labels", job.Labels).
				Str("fallback_label", label).
				Msg("fallback to macos runner")
			return RunnerMacOS
		}
	}

	// Default to Ubuntu runner if no match found
	logger.Debug().Strs("labels", job.Labels).Msg("no runner type matched, defaulting to ubuntu runner")
	return RunnerUbuntu
}
