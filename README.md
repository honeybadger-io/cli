# Honeybadger CLI

[![Test](https://github.com/honeybadger-io/cli/actions/workflows/test.yml/badge.svg)](https://github.com/honeybadger-io/cli/actions/workflows/test.yml)

A command-line interface for interacting with Honeybadger's Reporting API and Data API.

## Installation

### Homebrew

```bash
brew install honeybadger-io/tap/honeybadger
```

### Go

```bash
go install github.com/honeybadger-io/cli/cmd/hb@latest
```

Note: The install path includes `/cmd/hb` so Go installs the `hb` binary name.

### Download from GitHub

Pre-built binaries for Linux, macOS, and Windows are available on the [GitHub releases page](https://github.com/honeybadger-io/cli/releases).

### Install Script

On Linux and macOS, you can use the install script to download the binary:

```bash
curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | bash
```

No `sudo` required — the binary installs to `~/.local/bin` by default. When run as root, it installs to `/usr/local/bin`.

To also set up a systemd service for the [metrics agent](#reporting-api-commands):

```bash
curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | sudo bash -s -- --service --api-key YOUR_INGESTION_KEY
```

Run with `--help` for all options including `--version`, `--interval`, and `--install-dir`.

## Configuration

The CLI can be configured using either command-line flags, environment variables, or a configuration file.

### Configuration File

By default, the CLI looks for a configuration file at `~/.honeybadger-cli.yaml` in your home directory. You can specify a different configuration file using the `--config` flag.

Example configuration file:
```yaml
api_key: hbp_your-ingestion-key        # Ingestion Key, for the Reporting API (deploy, agent, run, check-in)
api_token: hbu_your-api-token           # API Token, for the Data API (projects, faults, insights, ...)
project_id: Xk9mZp                      # Optional, default project ID for Data API commands
endpoint: https://api.honeybadger.io    # Optional, use https://eu-api.honeybadger.io for EU region

# Optional tags for the metrics agent (added to every event)
agent:
  tags:
    environment: production
    role: web-1
```

### Environment Variables

You can set configuration using environment variables prefixed with `HONEYBADGER_`:

```bash
export HONEYBADGER_API_KEY=hbp_your-ingestion-key         # Ingestion Key, for the Reporting API
export HONEYBADGER_API_TOKEN=hbu_your-api-token            # API Token, for the Data API
export HONEYBADGER_PROJECT_ID=Xk9mZp                       # Optional, default project ID for Data API commands
export HONEYBADGER_ENDPOINT=https://eu-api.honeybadger.io  # Optional, for EU region
```

### API Tokens

Data API commands authenticate with an API Token: a personal one (`hbu_`, or an older `hbt_`) or an account one (`hba_`). Create one in your user settings at https://app.honeybadger.io/users/edit#authentication (EU: https://eu-app.honeybadger.io/users/edit#authentication). A token works only in the region it was created in.

The CLI refuses anything else before sending a request: an Ingestion Key (`hbp_`) only sends errors and events, and the v2 personal auth token, which `auth_token` / `HONEYBADGER_AUTH_TOKEN` / `--auth-token` used to hold, isn't accepted by the v3 Data API.

An API Token can be limited to some scopes and projects. When a command needs a scope the token doesn't have, the error names it (for example `requires faults:read`). `hb whoami` shows what the current token can reach.

Project IDs are the IDs the v3 Data API returns (for example `Xk9mZp`). Run `hb projects list` to find yours; replace older numeric project IDs in your flags, environment variables, and configuration.

### Command-line Flags

Global flags that apply to all commands:

 * `--api-key`: Your project's Ingestion Key (for the Reporting API)
 * `--api-token`: Your API Token (for the Data API)
 * `--endpoint`: Honeybadger endpoint (default: https://api.honeybadger.io)
 * `--config`: Path to configuration file

## Usage

Use `hb <command> --help` to see detailed usage information for any command.

### Reporting API Commands

These commands use an Ingestion Key, set with `--api-key` or `HONEYBADGER_API_KEY`:

| Command | Description |
|---------|-------------|
| `hb deploy` | Report a deployment to Honeybadger |
| `hb agent` | Start a metrics reporting agent that sends system metrics to Insights |
| `hb run` | Run a command and report its status to a check-in |
| `hb check-in` | Report a check-in without running a command |

### Data API Commands

These commands use the v3 Data API and an API Token, set with `--api-token` or `HONEYBADGER_API_TOKEN`:

| Command | Description |
|---------|-------------|
| `hb alarms` | Manage Insights alarms for your projects |
| `hb check-ins` | Manage check-ins for cron job and scheduled task monitoring |
| `hb comments` | Manage comments on faults |
| `hb dashboards` | Manage Insights dashboards for your projects |
| `hb deployments` | View deployment history |
| `hb environments` | Manage project environments |
| `hb faults` | View and manage faults (errors) in your projects, including bulk resolve and ignore |
| `hb ingestion-keys` | Manage a project's Ingestion Keys |
| `hb insights` | Execute BadgerQL queries against your Insights data |
| `hb integrations` | Manage a project's notification integrations |
| `hb projects` | Manage Honeybadger projects |
| `hb statuspages` | Manage public status pages |
| `hb streams` | View Insights data streams for your projects |
| `hb teams` | Manage teams |
| `hb uptime` | Manage uptime monitoring checks |
| `hb whoami` | Show the API Token in use and what it can reach |

The JSON these commands take with `--cli-input-json` and print with `--output json` is the v3 Data API's, documented in the [v3 Data API reference](https://docs.honeybadger.io/api/v3/). Each command's `--help` links to its page.

### Examples

```bash
# Report a deployment
hb deploy --environment production --revision abc123

# Start the metrics agent
hb agent --interval 60

# Start the metrics agent with custom tags
hb agent --tag environment=production --tag role=web-1

# Override the hostname reported by the agent
hb agent --tag host=web-1.prod --tag environment=production

# Run a command and report to a check-in
hb run --id XyZZy -- /usr/local/bin/backup.sh

# Report a check-in without running a command
hb check-in --slug daily-backup

# List all projects
hb projects list

# Query Insights data
hb insights query --project-id Xk9mZp --query "fields @ts, @preview | sort @ts" --ts P7D

# List faults for a project
hb faults list --project-id Xk9mZp

# Resolve a fault
hb faults update --project-id Xk9mZp --id Fk7pQ2 --resolved

# See which faults not seen since July would be resolved, then resolve them
hb faults resolve --project-id Xk9mZp --all --occurred-before 2026-07-01 --dry-run
hb faults resolve --project-id Xk9mZp --all --occurred-before 2026-07-01

# List Insights streams for a project
hb streams list --project-id Xk9mZp

# List Insights dashboards for a project
hb dashboards list --project-id Xk9mZp

# Create a dashboard from a JSON file
hb dashboards create --project-id Xk9mZp --cli-input-json file://dashboard.json

# Rename a dashboard (updates change only the fields you send)
hb dashboards update --project-id Xk9mZp --id abc123 --cli-input-json '{"title":"Checkout"}'

# Replace a dashboard's widgets: send the widgets list from `get`, edited
hb dashboards get --project-id Xk9mZp --id abc123 --output json | jq '{widgets}' > widgets.json
hb dashboards update --project-id Xk9mZp --id abc123 --cli-input-json file://widgets.json
```

See the [Honeybadger CLI documentation](https://docs.honeybadger.io/resources/cli/) for more information.

## Development

This project uses the [`api-go`](https://github.com/honeybadger-io/api-go) library for API interactions. For making changes to commands that use the Data API, you'll need to set up a Go workspace to work with both repositories simultaneously.

From the parent directory containing both `cli` and `api-go`:

```bash
# Initialize the workspace (if not already done)
go work init
go work use ./cli
go work use ./api-go

# The go.work file is gitignored and won't be committed
```

Now you can work on both repositories and changes to `api-go` will be immediately reflected when working on the CLI.

### Working with Dependencies

When using the workspace, Go uses the local `api-go` directory instead of fetching from GitHub. However, `go.sum` must still contain checksums for the published `api-go` module to support:
- CI/CD builds (which don't have the workspace)
- Developers who clone only this repository
- Docker builds

**When to use `GOWORK=off`:**

```bash
# Update dependencies and go.sum with published module checksums
GOWORK=off go mod tidy

# Install a specific version of a dependency
GOWORK=off go get github.com/some/package@v1.2.3

# Test the build as if no workspace exists (simulates CI/end-user builds)
GOWORK=off go build ./cmd/hb
GOWORK=off go test ./...
```

The `GOWORK=off` flag temporarily disables the workspace, ensuring that `go.sum` contains the correct checksums for the published modules.

## Contributing

Pull requests are welcome. If you're adding a new feature, please [submit an issue](https://github.com/honeybadger-io/cli/issues/new) as a preliminary step; that way you can be (moderately) sure that your pull request will be accepted.

When adding or changing functionality, please also add or update corresponding tests.

To run tests locally:

```bash
go test ./...
```

To build and test local binaries:

```bash
go build -o ./hb
./hb run --id check-123 -- /usr/local/bin/backup.sh
```

### To contribute your code:

1. Fork it.
1. Create a topic branch `git checkout -b my_branch`
1. Make your changes and add an entry to the [CHANGELOG](CHANGELOG.md).
1. Commit your changes `git commit -am "Boom"`
1. Push to your branch `git push origin my_branch`
1. Send a [pull request](https://github.com/honeybadger-io/cli/pulls)

### Releasing

To publish a new release:

1. Create and push a new tag with the version number:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

1. The release workflow will automatically:
   - Build binaries for multiple platforms (Linux, macOS, Windows)
   - Create a GitHub release with the binaries
   - Generate a changelog from commit messages

   The binaries will be available for download from the GitHub releases page.

Note: Commits with messages containing `[skip ci]` will skip the test workflow, but the release workflow will still run when a tag is pushed.

## License

MIT License. See the [LICENSE](https://raw.githubusercontent.com/honeybadger-io/cli/master/LICENSE) file in this repository for details.
