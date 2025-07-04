# Octoscope

The missing cost explorer for GitHub Actions. Track and analyze your GitHub Actions usage and costs.


## Install
Prerequisites: gh (GitHub CLI)

```shell
gh extension install noamtamir/gh-octoscope
```

## Quickstart

Login to github via gh, if not already logged in:
```shell
gh auth login
```


Generate full ephemeral report (available for 72 hours):
```shell
gh octoscope report
```

Generate local reports and show debug logs:
```shell
gh octoscope report --csv --debug
```

Only fetch data without generating reports, for future use:
```shell
gh octoscope fetch
```

Generate reports from previously fetched data:
```shell
gh octoscope report --fetch=false
```

Delete a report data from the server:
```shell
gh octoscope report delete <report-id>
```

### Enable shell completion

#### Bash
```shell
source <(gh octoscope completion bash)

# To load completions for each session, add to your .bashrc:
# gh octoscope completion bash > /usr/local/etc/bash_completion.d/gh-octoscope
```

#### Zsh
```shell
source <(gh octoscope completion zsh)

# To load completions for each session:
# gh octoscope completion zsh > "${fpath[1]}/_gh-octoscope"
```

#### Fish
```shell
gh octoscope completion fish | source

# To load completions for each session:
# gh octoscope completion fish > ~/.config/fish/completions/gh-octoscope.fish
```

### Available Commands

```shell
gh octoscope [flags]
gh octoscope [command]
```

#### Main Commands
- `report`: Generate reports based on GitHub Actions usage data
  - `report delete`: Delete a report from the Octoscope server
- `fetch`: Fetch GitHub Actions usage data without generating reports
- `version`: Print the version number of gh-octoscope
- `completion`: Generate shell completion scripts

#### Global Flags
- `--debug`: Sets log level to debug
- `--prod-log`: Enable production structured logging
- `--from`: Generate report from this date. Format: YYYY-MM-DD
- `--page-size`: Page size for GitHub API requests (default 30)
- `--obfuscate`: Obfuscate sensitive data in reports (usernames, emails)

#### Report Command Flags
- `--csv`: Generate CSV report
- `--fetch`: Whether to fetch new data or use existing data (default true, set to false to use previously fetched data)

## Devlop Locally
### Install

Prerequisites:
- Go 1.21+
- GitHub CLI (gh)

Clone this repository:
```shell
git clone https://github.com/noamtamir/gh-octoscope.git
```

Build the extension:
```shell
cd gh-octoscope && go build
```

Install the extension:
```shell
gh extension install .
```

Login to github via gh, if not already logged in:
```shell
gh auth login
```

Run locally:
```shell
cd /path/to/repo/ && gh octoscope report
```

## Optional:Environment Variables

When using the `report` command, the following environment variables are configurable:
- `OCTOSCOPE_API_URL`: The base URL of the Octoscope API (default: https://octoscope-server-production.up.railway.app)
- `OCTOSCOPE_APP_URL`: The base URL of the Octoscope web application (default: https://octoscope.netlify.app)

You can set them via a `.env` file:

```shell
OCTOSCOPE_API_URL=http://0.0.0.0:8888
OCTOSCOPE_APP_URL=http://localhost:3333
```