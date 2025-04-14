# Octoscope

The missing cost explorer for GitHub Actions. Track and analyze your GitHub Actions usage and costs.

## Installation
```shell
gh extension install noamtamir/gh-octoscope
```

## Usage
```shell
gh octoscope -csv -html -debug -from=2024-01-01
```

Available flags:
- `-csv`: Generate CSV report
- `-html`: Generate HTML report
- `-debug`: Enable debug logging
- `-from`: Generate report from this date (YYYY-MM-DD format)
- `-prod-log`: Enable production structured logging

## Development

### Prerequisites
- Go 1.21+
- GitHub CLI (gh)

### Setting up locally

1. Clone the repository:
```shell
git clone https://github.com/noamtamir/gh-octoscope.git
```

2. Install as a local extension:
```shell
gh extension install .
```

3. Run locally:
```shell
go build && gh octoscope -csv -html -debug -from=2024-01-01
```