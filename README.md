# Octoscope
The missing cost explorer for GitHub Actions
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