# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.0] - 2026-08-13

### Fixed

- Ignore the per-member escalation-layer schedule copies PagerDuty now creates (`... (1)`, `... (2)`, ...), which matched the team prefix and marked every member as on call.
- Page through the PagerDuty schedule list. The escalation ladder pushed the account well past the 25 schedules a single unpaginated call returns, so most teams' primary schedules were never seen.
- Skip the on-call lookup when no schedule matches a team, instead of asking PagerDuty for every on-call in the account.
- Surface PagerDuty client and schedule-listing errors instead of discarding them.
- Point `image.repository` at `gsoci.azurecr.io/giantswarm/ailefroide`. `quay.io` was dropped from the CI registry list and has held nothing newer than `0.4.0` since October 2024.

### Security

- Resolve all 20 open Dependabot advisories:
  - `golang.org/x/crypto` (17 advisories, 8 critical) is no longer required at all. It was pulled in solely by `go-github/v47`'s use of the unmaintained `golang.org/x/crypto/openpgp` (`GO-2026-5932`, no fix available); `go-github` v74+ replaced that with a `MessageSigner` interface, so bumping to `v88` drops the module from the graph.
  - `github.com/golang-jwt/jwt/v4` `v4.4.1` -> `v4.5.2` (`GHSA-mh63-6h87-95cp`, `GHSA-29wx-vh33-7x7r`).
  - `github.com/slack-go/slack` `v0.11.2` -> `v0.23.1` (`GHSA-gxhx-2686-5h9g`).
- `govulncheck` under the CI toolchain (go1.26.4) now reports zero module vulnerabilities. The one remaining finding is `GO-2026-5856` in `crypto/tls`, fixed in go1.26.5 and therefore owned by the `architect` image rather than this repository.

### Changed

- Update the `architect` orb to `9.6.0` (was `6.8.0`).
- Update `github.com/google/go-github` `v47` -> `v88` and `github.com/bradleyfalzon/ghinstallation/v2` `v2.1.0` -> `v2.19.0`. `github.NewClient` gained an options form returning an error, so the client is now built with `github.WithHTTPClient`.
- Raise the `go` directive to `1.25.0`, required by the updated dependencies. CI builds with go1.26.4.
- Surface the discarded error from `ghinstallation.NewKeyFromFile`. It left the transport nil, so an unreadable App key surfaced later as an unrelated transport failure instead of naming the key. `NewGithub` now returns an error and `main` exits on it.
- Build the container image for `linux/amd64` and `linux/arm64`, selecting the matching binary via `TARGETARCH`.
- Update the base image to `alpine:3.24.1` (was `3.16.2`, EOL since May 2024).
- Drop the deprecated no-op `replace-chart-version-with-git` / `replace-app-version-with-git` keys from `.abs/main.yaml`; the orb now stamps versions with `--override-chart-version` / `--override-app-version`.
- Regenerate `.github/workflows/zz_generated.*.yaml` via devctl to use the centralized reusable workflow, removing the Node-20 `mindsers/changelog-reader-action` dependency.

## [0.6.0] - 2025-12-17

## [0.5.0] - 2025-11-26

## [0.4.0] - 2024-10-28

### Changed

- Update `github.com/giantswarm/personio-go` to `v0.5.0`.

## [0.3.8] - 2024-04-18

- Improve Personio time-off handling (should fix stability issue)

## [0.3.7] - 2024-03-14

- Better logging

## [0.3.6] - 2024-02-15

- Simplify calendar functions
- kill On call engineer at end of day

## [0.3.5] - 2023-11-30

## [0.3.4] - 2023-11-29

### Changed

- Make CronJob PSS compliant.

## [0.3.3] - 2023-10-16

### Changed

- Fix calendar and half day logic

## [0.3.2] - 2023-10-13

### Changed

- Fix for long description
- additional error checks for slack errors

## [0.3.1] - 2023-10-12

### Changed

- Support handles were being matched on name rather than handle which caused teams
  to be missed from rotation

## [0.3.0] - 2023-10-10

### Changed

- Now create user group function checks errors for existing handles
- Now user group description is checked before submit the usergroup to not go over the max length

## [0.2.0] - 2023-06-09

### Added

- Better functionality to support adding other team members to the team support handles

## [0.1.7] - 2023-03-28

## [0.1.6] - 2023-03-28

## [0.1.5] - 2023-03-28

## [0.1.4] - 2023-03-28

## [0.1.3] - 2023-03-28

## [0.1.2] - 2022-12-20

- Fix image settings

## [0.1.1] - 2022-12-20

- Move github to github-app

## [0.1.0] - 2022-09-27

## [0.1.0] - 2022-09-27

- Release first stable version.

[Unreleased]: https://github.com/giantswarm/ailefroide-app/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/giantswarm/ailefroide-app/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/giantswarm/ailefroide-app/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/giantswarm/ailefroide-app/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.8...v0.4.0
[0.3.8]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.7...v0.3.8
[0.3.7]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.6...v0.3.7
[0.3.6]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.1...v0.3.1
[0.3.1]: https://github.com/giantswarm/ailefroide-app/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/giantswarm/ailefroide-app/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/ailefroide-app/compare/v0.1.7...v0.2.0
[0.1.7]: https://github.com/giantswarm/ailefroide-app/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/giantswarm/ailefroide-app/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/giantswarm/ailefroide-app/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/giantswarm/ailefroide-app/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/giantswarm/ailefroide-app/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/giantswarm/ailefroide-app/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/giantswarm/ailefroide-app/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/giantswarm/ailefroide-app-app/releases/tag/v0.1.0
