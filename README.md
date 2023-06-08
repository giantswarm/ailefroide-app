# Ailefroide

> This is not Alfred this is a froide.

## What is this?

The purpose of this application is to automatically create and manage support
handles for teams in Slack.

It achieves this by pulling a list of users from multiple sources and then
cross-referencing them with their role, AFK and On Call status.

- User information is drawn from Personio
- Team membership is currently taken from Github matching on the username stored
  in Personio
- On call status is drawn from OpsGenie
- AFK status is drawn from Google AFK calendar

Other than your main team, additional role information is also drawn from the
following teams in Github

- chapter-sa for Solution Architects
- chapter-ae for Account Engineers
- product-owners for Product owners
- chapter-architecture for Platform architects
- chapter-sre for site reliability engineers

> Teams eligible for support handle rotation are Github teams that start with
> the `team-` prefix and contain either Solution Architects or Account
> Engineers. Teams which do not contain either of these two roles are skipped
> from the rotation cycle.

If you appear in both chapter-sa and chapter-ae, you are classed as an account
engineer.

## Configuration

See [config.yaml](./examples/config.yaml) for configuration options

In addition to the config options defined, the following environment variables
must also be defined

- `SLACK_TOKEN` An API key for the slack API
- `OPSGENIE_TOKEN` An API key for the OpsGenie API
- `PERSONIO_CLIENT_ID` A client ID for Personio
- `PERSONIO_CLIENT_SECRET` A client secret for Personio

### Team specific configuration

To enable Ailefroide for your team, you need to edit the `ConfigMap` for the
application deployment. For Giant Swarm this can be found in [Workload Clusters
Fleet](https://github.com/giantswarm/workload-clusters-fleet/blob/main/management-clusters/gorilla/organizations/giantswarm-production/workload-clusters/rfjh2/apps/ailefroide/configmap.yaml).

> By default, if your team is unlisted, your team will not be included in the
> rotation. You **must** enable your team to be included.

For each team, there are certain configuration options that can be included

- `skipRotation bool` - If true, disables your team from
  the support-handle rotation.
- `includePo bool` - if False, your product owner will not
  be included in the team support handle. Defaults to true
- `includeOCE bool` - If false, your On Call Engineer will not be included in
  the support handle. Defaults to true
- `includePa bool` - If true, your teams Platform architect will be included in
  the support handle. Defaults to false
- `includeSRE bool` - If true, your teams site reliability engineer will be
  included in the support handle. Defaults to false
- `extraCover string[]` A list of additional people to include for extra
  coverage of your teams support handle. Both email addresses and Github handles
  are supported here.

#### Examples

- Enable your team, accept all defaults

  ```yaml
  teams:
    team-honeybadger: {}
  ```

- Enable your team but set `skipRotation` (Has same effect as unlisted team)

  ```yaml
  teams:
    team-honeybadger:
      skipRotation: true
  ```

  - Enable your team, include the on call engineer, skip the product owner but
    include the SRE

  ```yaml
  teams:
    team-honeybadger:
      skipRotation: false
      includePo: false
      includeOCE: true
      includeSRE: true
  ```

- Enable your team, accept all defaults, add `fakeuser@giantswarm.io` as
  additional cover for your team.

  ```yaml
  teams:
    team-honeybadger:
      extraCover:
        - fakeuser@giantswarm.io
  ```

- Alternatively you can use Github handles

  ```yaml
  teams:
    team-honeybadger:
      extraCover:
        - imafakegithubuser
  ```

### Include When Not AFK

In certain instances, you may wish for additional members of your team to
appear in the support handles. These will typically be members of your team
that are not already identified by the roles above for which specific flags
exist.

To support this, `Ailefroide` allows for your own team members to be added to
the list identified as `extraCover`. When someone is inside this list, they are
always added to the support handle except in instances they are AFK.

## Build

```bash
go mod tidy
go build .
```

## Execution

> Execution of this application will affect live support handles. When testing,
> please use the `debug` flag to ensure support is not impacted by changes
> being made.

To run this application you need to specify a configuration file for it.

```bash
./ailefroide -config config.yaml
```

Possible arguments are:

- `-config string` The configuration file to use
- `-debug bool` Run this application in debug mode
- `-debugteam` Limit debugging to the single team specified

## Trivia

The name `Ailefroide` is the name of a Mountain in the [French Alps](https://en.wikipedia.org/wiki/Ailefroide).

It was chosen from a play on the app not being Alfred.

- I'm not Alfred I'm afraid
- I'm not Alfred I'm a fraud
- I'm not Alfred I'm a froide

Then remembering there are at least two climbers on the Solution Architecture
team, I chose Ailefroide as being the nearest alternative with meaning.
