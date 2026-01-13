# Honeybadger CLI

[![Test](https://github.com/honeybadger-io/cli/actions/workflows/test.yml/badge.svg)](https://github.com/honeybadger-io/cli/actions/workflows/test.yml)

A command-line interface for interacting with Honeybadger's Reporting API and Data API.

## Installation

```bash
go install github.com/honeybadger-io/cli/cmd/hb@latest
```

Note: The install path includes `/cmd/hb` so Go installs the `hb` binary name.

## Configuration

The CLI can be configured using either command-line flags, environment variables, or a configuration file.

### Configuration File

By default, the CLI looks for a configuration file at `config/honeybadger.yml` in the current directory. You can specify a different configuration file using the `--config` flag.

Example configuration file:
```yaml
api_key: your-project-api-key-here      # For Reporting API (deploy, agent commands)
auth_token: your-personal-auth-token    # For Data API (projects, faults, insights commands)
endpoint: https://api.honeybadger.io    # Optional, use https://eu-api.honeybadger.io for EU region
```

### Environment Variables

You can set configuration using environment variables prefixed with `HONEYBADGER_`:

```bash
export HONEYBADGER_API_KEY=your-project-api-key-here       # For Reporting API (deploy, agent)
export HONEYBADGER_AUTH_TOKEN=your-personal-auth-token     # For Data API (projects, faults, insights)
export HONEYBADGER_ENDPOINT=https://eu-api.honeybadger.io  # Optional, for EU region
```

### Command-line Flags

Global flags that apply to all commands:

 * `--api-key`: Your Honeybadger project API key (for Reporting API)
 * `--auth-token`: Your Honeybadger personal auth token (for Data API)
 * `--endpoint`: Honeybadger endpoint (default: https://api.honeybadger.io)
 * `--config`: Path to configuration file

## Usage

Use `hb <command> --help` to see detailed usage information for any command.

### Reporting API Commands

These commands use `--api-key` or `HONEYBADGER_API_KEY` (project API key):

| Command | Description |
|---------|-------------|
| `hb deploy` | Report a deployment to Honeybadger |
| `hb agent` | Start a metrics reporting agent that sends system metrics to Insights |

### Data API Commands

These commands use `--auth-token` or `HONEYBADGER_AUTH_TOKEN` (personal auth token):

| Command | Description |
|---------|-------------|
| `hb accounts` | Manage Honeybadger accounts and team members |
| `hb check-ins` | Manage check-ins for cron job and scheduled task monitoring |
| `hb comments` | Manage comments on faults |
| `hb deployments` | View and manage deployment history |
| `hb environments` | Manage project environments |
| `hb faults` | View and manage faults (errors) in your projects |
| `hb insights` | Execute BadgerQL queries against your Insights data |
| `hb projects` | Manage Honeybadger projects |
| `hb statuspages` | Manage public status pages |
| `hb teams` | Manage teams and team memberships |
| `hb uptime` | Manage uptime monitoring checks |

### Examples

```bash
# Report a deployment
hb deploy --environment production --revision abc123

# Start the metrics agent
hb agent --interval 60

# List all projects
hb projects list

# Query Insights data
hb insights query --project-id 12345 --query "fields @ts, @preview | sort @ts"

# List faults for a project
hb faults list --project-id 12345
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
