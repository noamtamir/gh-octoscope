# Octoscope

The missing cost explorer for GitHub Actions. Track and analyze your GitHub Actions usage and costs.

## Installation
```shell
gh extension install noamtamir/gh-octoscope
```

## Usage
- Generate only local reports:
```shell
gh octoscope -csv -html -debug
```
- Generate and get link full ephemeral report (available for 72 hours):
```shell
gh octoscope -report -debug
```
- Optionally obfuscate sensitive information like usernames and emails
```shell
gh octoscope -report -debug -obfuscate
```

Available flags:
- `-csv`: Generate CSV report
- `-html`: Generate HTML report
- `-report`: Generate and get link to full ephemeral report
- `-obfuscate`: Obfuscate any sensitive information (usernames / emails)
- `-from`: Generate report from this date (YYYY-MM-DD format)
- `-fetch`: Fetch new data (defaults to true, set to -fetch=false to use previously fetched data to generate reports)
- `-debug`: Enable debug logging
- `-prod-log`: Enable production structured logging

## Viewing the HTML Report

After generating an html report, you can view it by running a simple web server:

### Prerequisites
- Python 3

### Start a local server
```shell
cd reports && python3 -m http.server 8000

// go to: http://localhost:8000/report.html
```
Press Ctrl+C to stop the local server.

## Development

### Prerequisites
- Go 1.21+
- GitHub CLI (gh)

### Setting up locally

1. Clone the repository:
```shell
git clone https://github.com/noamtamir/gh-octoscope.git
```

2. Install the extension:
```shell
gh extension install .
```

3. Run locally:
```shell
go build && gh octoscope -csv -html -debug -from=2024-01-01
```

### Optional:Environment Variables

When using the `-report` flag, the following environment variables are configurable:
- `OCTOSCOPE_API_URL`: The base URL of the Octoscope API (default: https://octoscope-server-production.up.railway.app)
- `OCTOSCOPE_APP_URL`: The base URL of the Octoscope web application (default: https://octoscope.netlify.app)

You can set them via a `.env` file:

```shell
OCTOSCOPE_API_URL=http://0.0.0.0:8888
OCTOSCOPE_APP_URL=http://localhost:3333
```