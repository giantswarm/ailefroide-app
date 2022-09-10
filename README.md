# Ailefroide

> this is not alfred this is a froid.

## What is this?

this is WIP

Matches slack users to github teams, works out who the solution architects and
account engineers are, who is on call and will use that information to manage
support handles

## TODO

- AFK calendar integration
- Create slack handles
- Performance improvements
- Containerisation
- Chart
- Everything else...

## Build

```bash
go mod tidy
go build .
```

## Execution

Currently only prints teams to STDOUT

```bash
export GITHUB_TOKEN='xxxxx'
export SLACK_TOKEN='xxxxx'
export OPSGENIE_TOKEN='xxxxxx'
./ailefroide
```

