# Change Log

All notable changes to this project will be documented in this file. See [Keep a
CHANGELOG](http://keepachangelog.com/) for how to update this file. This project
adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Changed

- BREAKING: `deployments list`, `uptime outages`, and `uptime checks` now require human-readable values for `--created-after`/`--created-before` (`YYYY-MM-DD` or RFC3339). Unix epoch input for these flags is no longer accepted.

## [0.5.0] - 2026-02-10

### Added

- Added option to provide project_id via environment variable or config file (#23)

## [0.4.0] - 2026-01-20

### Added

- Added CLI commands for Data API endpoints (#16)
- Moved default config file to ~/.honeybadger-cli.yaml (#20)
- Added `run` and `check-in` commands to execute commands and report their status to Honeybadger check-ins (#7)

## [0.3.0] - 2026-01-06

### Added

- Release tagged versions to homebrew (#15)
- Add `projects` and `faults` commands to work with the Data API (#13)

## [0.2.1] - 2024-12-21

### Added

- Added [agent command](./README.md#agent-command)

## [0.2.0] - 2024-12-16

### Added
- Made API endpoint configurable

## [0.1.0] - 2024-12-09

### Added

- Initial release
