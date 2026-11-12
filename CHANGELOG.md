# Change Log

All notable changes to this project will be documented in this file. See [Keep a
CHANGELOG](http://keepachangelog.com/) for how to update this file. This project
adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.11.0] - 2026-11-12

The Data API commands now use Honeybadger's v3 Data API. This is a breaking change, released as a minor version (v0.11.0) while the CLI is below 1.0.

### Changed

- Data API commands authenticate with an API Token (`hbu_` personal, older `hbt_`, or `hba_` account), set with `--api-token`, `HONEYBADGER_API_TOKEN` or `api_token` in the config file. `--auth-token`, `HONEYBADGER_AUTH_TOKEN` and `auth_token` are removed, and v2 personal auth tokens no longer work; the CLI says so before sending a request, and also refuses an Ingestion Key (`hbp_`)
- A 401 from the Data API adds a hint to check the token hasn't been revoked and belongs to this region (US or EU); a missing scope is named in the error
- IDs are strings, as v3 returns them. Fault IDs keep their v2 numbers; every other v2 ID (projects, deploys, environments, comments, teams and the rest) is replaced by v3's, so update scripts, `HONEYBADGER_PROJECT_ID` and `project_id` in config files (`hb projects list` shows the new project IDs). `uptime` site IDs are UUIDs and are checked before sending
- `--cli-input-json` takes v3's request body directly: `{"name": "My Project"}`, not `{"project": {"name": "My Project"}}`. The old wrapped form, unknown fields (nested ones too), a null on a field that can't be null, and trailing content are refused before sending. Field names follow v3 (status page `sites` and `check_ins` are lists of objects, for example)
- Updates are partial: only the fields you send change. `dashboards update` no longer needs the whole dashboard; a sent `widgets` list replaces the widgets, and an alarm's `trigger_config` replaces the stored trigger
- `--output json` prints v3's resource JSON, so field names and shapes follow v3 (for example, a fault's `url` is now `links.web`, and an integration's `options` is now `config`). The [v3 Data API reference](https://docs.honeybadger.io/api/v3/) documents each one, and every Data API command's `--help` links to its page. `insights query --output json` prints the query's `results` and `meta`
- `insights query --ts` is a time range such as `1h` or `P7D`, as v3 defines it, not a timestamp
- `faults list`, `faults notices`, `deployments list`, `uptime outages` and `uptime checks` accept `--limit` up to 100
- `check-ins list`, `environments list`, `streams list` and `comments list` return every page instead of the first; `comments list` stays oldest first, as in v2
- `projects get` no longer shows the project's token, teams, users or sites (use `hb ingestion-keys list` for keys); `faults notices` drops the HOSTNAME column; `environments get` drops Updated; `projects occurrences` prints timestamps as dates
- The Ingestion Key is still set with `--api-key` / `HONEYBADGER_API_KEY`; `deploy`, `agent`, `run` and `check-in` are unchanged
- Requires Go 1.27 to build from source

### Added

- `integrations` command to manage a project's notification integrations (replaces `projects integrations`)
- `ingestion-keys` command to manage a project's Ingestion Keys
- `faults resolve`, `unresolve`, `ignore` and `unignore` for many faults at once, by `--id`, `--query`, or `--all` with a time filter, with `--dry-run`
- `faults merge`, `faults pause-recording`, `faults resume-recording` and `faults delete`
- `check-ins events` to list a check-in's recent events
- `whoami` to show the API Token in use, its scopes and the projects it can reach
- `deployments list --before` and `--after` to page through deployments; each page prints the cursor for the next one to stderr

### Removed

- `accounts` (all subcommands), `teams members` and `teams invitations`, and `deployments delete`: the v3 client this release uses doesn't support them yet; they return in a later release
- `projects reports`: v3 has no reports. Use `insights query`; for example notices by class over the last week: `hb insights query --project-id ID --ts P7D --query 'filter event_type::str == "notice" | stats count() as notices by klass::str as klass | sort notices desc'`
- `projects integrations` (use `integrations list`)
- `--account-id` on `projects list`, `projects create`, `teams` and `statuspages`: the account comes from the API Token
- `deployments list --created-after` and `--created-before`, and `uptime outages`/`uptime checks --created-after`: v3 has no such filters. Page back through deployments with `--before` instead

## [0.10.1] - 2026-08-14

### Fixed

- `alarms create`/`alarms update` help no longer claims `stream_ids` is required; it is optional and defaults to all of the project's streams. The help now explains that unrecognized IDs are ignored and that a list in which none are recognized is rejected (commonly a slug such as `default` passed where a stream ID belongs)

## [0.10.0] - 2026-08-13

### Added

- Add `dashboards` command for managing Insights dashboards (list, get, create, update, delete)

## [0.9.0] - 2026-07-17

### Added

- Add `streams` command for listing Insights streams
- Add `faults update` command for resolving, ignoring, assigning, and marking faults to resolve on deploy
- Add `--stream-ids` flag to `insights query` to restrict queries to specific streams

### Changed

- `projects create` no longer requires `--account-id`; when omitted, the project is created in the first account the auth token has access to

## [0.8.0] - 2026-07-06

### Added

- Add `alarms` command for managing Insights alarms (list, get, create, update, delete, history)

## [0.7.0] - 2026-04-06

### Added

- Add --tag flag to agent for custom metric metadata (#25)

## [0.6.0] - 2026-03-05

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
