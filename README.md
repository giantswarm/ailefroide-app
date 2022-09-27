# Ailefroide

> this is not alfred this is a froid.

## What is this?

> This is work in progress - if you find bugs with this application, please
> raise them as issues

The purpose of this application is to automnatically create and manage support
handles for teams in Slack.

It achieves this by pulling a list of users from multiple sources and then
cross-referencing them with their role and status using the following flow:

1. Retrieve a list of email addresses and github handles from personio.
2. Retrieve a list of user IDs and email addresses from Slack
3. Match email addresses from personio and slack to ensure that each person has
   a slack ID
4. Retrieve a list of github teams matching the `team-*` prefix from the
   organisation in github
5. Match slack users to github teams based on the github handle retrieved from personio
6. Determine the Account engineers and solution architects based on the github
   teams defined in `cfg.AccountEngineers` and `cfg.SolutionArchitects` respectively
7. Determine the oncall engineer by matching the user email address to opsgenie
8. Determine AFK entries by matching against the AFK Google Calandar
9. Create `support-<TEAM>` handles containing nthe non-afk solution
   architects/account-engineers and the oncall engineer
10. Create cross team topics containing the same as `9.` using the list of topics
    pulled from the teams `Topic` field in slack.

## TODO

- Containerisation
- Chart
- Deployment

## Build

```bash
go mod tidy
go build .
```

## Execution

> Execution of this application will affect live support handles. When testing,
> please use the  `debug` flag to ensure support is not impacted by changes being
> made.

To run this application you need to specify a configuration file for it.

```bash
./ailefroide -config config.yaml
```

Possible arguments are:

- `-config string` The configuration file to use
- `-debug bool` Run this application in debug mode
- `-debugteam` Limit debugging to the single team specified

## Configuration

See [config.yaml](./examples/config.yaml) for configuration options

In addition to the config options defined, the following environment variables
must also be defined

- `GITHUB_TOKEN` A personal access token for a github account to use for
  scanning users and teams
- `SLACK_TOKEN` An API key for the slack api
- `OPSGENIE_TOKEN` An api key for the opsgenie api
- `PERSONIO_CLIENT_ID` A client ID for personio
- `PERSONIO_CLIENT_SECRET` A client secret for personio
