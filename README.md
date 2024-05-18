# Octoscope
The missing cost explorer for GitHub Actions

## Usage
```shell
gh octoscope
gh octoscope -csv -debug -from=2024-01-01
```
## Developing locally
dependencies:
- go
- gh cli: https://cli.github.com/

install locally:
```shell
gh extension install .
```
run locally:
```shell
gh octoscope
```
see changes:
```shell
go build && gh octoscope
```
publish release: push a tag to start release pipeline