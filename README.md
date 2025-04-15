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

## Viewing the HTML Report

After generating the report, you can view it by running a simple web server:

### Prerequisites
- Python 3 (if you don't have Python installed):
  - Windows: Download from [python.org](https://www.python.org/downloads/)
  - macOS: `brew install python3` (requires Homebrew) or download from [python.org](https://www.python.org/downloads/)
  - Linux: `sudo apt install python3` (Ubuntu/Debian) or `sudo dnf install python3` (Fedora)

### Starting the Server
1. Navigate to the reports directory:
```shell
cd reports
```

2. Start the Python server:
```shell
python3 -m http.server 8000
```

3. Open your web browser and visit:
```
http://localhost:8000/report.html
```

The server will continue running until you press Ctrl+C to stop it.

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