# Octoscope

The missing cost explorer for GitHub Actions. Track and analyze your GitHub Actions usage and costs.

## Features

- Track GitHub Actions usage across your repositories
- Calculate costs based on runner types and minutes used
- Generate CSV and HTML reports
- Support for all GitHub-hosted runner types
- Detailed job and workflow analysis

## Installation

```shell
gh extension install noamtamir/gh-octoscope
```

## Usage

Basic usage:
```shell
gh octoscope
```

Generate reports:
```shell
gh octoscope -csv -html -from=2024-01-01
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

### Project Structure
```
├── internal/
│   ├── api/        # GitHub API client
│   ├── billing/    # Cost calculation logic
│   └── reports/    # Report generation
├── main.go         # Application entry point
└── report-template.html
```

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
go build && gh octoscope
```

### Running Tests

```shell
go test ./...
```

### Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

MIT License - see LICENSE file for details