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

By default, the CLI looks for a configuration file at `~/.honeybadger-cli.yaml` in your home directory. You can specify a different configuration file using the `--config` flag.

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

### Deploy Command

Report a deployment to Honeybadger. **Requires**: `--api-key` or `HONEYBADGER_API_KEY` (project API key)

```bash
hb deploy --environment production --repository github.com/org/repo --revision abc123 --user johndoe
```

**Required:**

 * `-e, --environment` - Environment being deployed to (e.g., `production`)

**Optional:**

 * `-r, --repository` - Repository being deployed (e.g., `github.com/org/repo`)
 * `-v, --revision` - Revision or commit SHA being deployed
 * `-u, --user` - Local username of the person deploying

### Agent Command

Start a metrics reporting agent that collects and sends system metrics to Honeybadger Insights. **Requires**: `--api-key` or `HONEYBADGER_API_KEY` (project API key)

```bash
hb agent
```

The agent collects and reports the following metrics:
 * CPU usage and load averages
 * Memory usage (total, used, free, available)
 * Disk usage for all mounted filesystems

**Optional:**

 * `--interval` - Reporting interval in seconds (default: `60`)

### Projects Command

Manage Honeybadger projects. **Requires**: `--auth-token` or `HONEYBADGER_AUTH_TOKEN` (personal auth token)

```bash
# List all projects
hb projects list

# List projects by account ID
hb projects list --account-id 12345

# Get project details
hb projects get --id 12345

# Create a new project (using inline JSON)
hb projects create --account-id 12345 --cli-input-json '{"project": {"name": "My Project"}}'

# Create a new project (using a JSON file)
hb projects create --account-id 12345 --cli-input-json file://project.json

# Update a project (using inline JSON)
hb projects update --id 12345 --cli-input-json '{"project": {"name": "Updated Name", "resolve_errors_on_deploy": true}}'

# Update a project (using a JSON file)
hb projects update --id 12345 --cli-input-json file://updates.json

# Delete a project
hb projects delete --id 12345

# Get occurrence counts for all projects
hb projects occurrences --period day --environment production

# Get occurrence counts for a specific project
hb projects occurrences --id 12345 --period hour

# Get integrations for a project
hb projects integrations --id 12345

# Get report data for a project
hb projects reports --id 12345 --type notices_per_day --start 2024-01-01T00:00:00Z --stop 2024-01-31T23:59:59Z
```

#### list

**Optional:**

 * `--account-id` - Filter projects by account ID
 * `-o, --output` - Output format: `table` or `json` (default: `table`)

#### get

**Required:**

 * `--id` - Project ID

**Optional:**

 * `-o, --output` - Output format: `text` or `json` (default: `text`)

#### create

**Required:**

 * `--account-id` - Account ID to create the project in
 * `--cli-input-json` - JSON payload (inline string or `file://path`)

**Optional:**

 * `-o, --output` - Output format: `text` or `json` (default: `text`)

JSON payload format:
```json
{
  "project": {
    "name": "My Project",
    "resolve_errors_on_deploy": true,
    "disable_public_links": false,
    "language": "ruby",
    "user_url": "https://myapp.com/users/[user_id]",
    "source_url": "https://github.com/myorg/myrepo/blob/main/[filename]#L[line]",
    "purge_days": 90,
    "user_search_field": "user_id"
  }
}
```

#### update

**Required:**

 * `--id` - Project ID
 * `--cli-input-json` - JSON payload (inline string or `file://path`)

JSON payload format (all fields optional):
```json
{
  "project": {
    "name": "Updated Name",
    "resolve_errors_on_deploy": false,
    "purge_days": 120
  }
}
```

#### delete

**Required:**

 * `--id` - Project ID

#### occurrences

**Optional:**

 * `--id` - Project ID (if omitted, shows data for all projects)
 * `--period` - Time period: `hour`, `day`, `week`, or `month` (default: `day`)
 * `--environment` - Filter by environment name
 * `-o, --output` - Output format: `table` or `json` (default: `table`)

#### integrations

**Required:**

 * `--id` - Project ID

**Optional:**

 * `-o, --output` - Output format: `table` or `json` (default: `table`)

#### reports

**Required:**

 * `--id` - Project ID
 * `--type` - Report type: `notices_by_class`, `notices_by_location`, `notices_by_user`, or `notices_per_day`

**Optional:**

 * `--start` - Start time in RFC3339 format (e.g., `2024-01-01T00:00:00Z`)
 * `--stop` - Stop time in RFC3339 format (e.g., `2024-01-31T23:59:59Z`)
 * `--environment` - Filter by environment name
 * `-o, --output` - Output format: `table` or `json` (default: `table`)

See https://docs.honeybadger.io/api/projects/ for more information.

### Faults Command

View and manage faults (errors) in your Honeybadger projects. **Requires**: `--auth-token` or `HONEYBADGER_AUTH_TOKEN` (personal auth token)

```bash
# List faults for a project
hb faults list --project-id 12345

# List with filtering
hb faults list --project-id 12345 --query "class:RuntimeError" --order recent --limit 10

# Get fault details
hb faults get --project-id 12345 --id 67890

# List notices for a fault
hb faults notices --project-id 12345 --id 67890

# Get fault counts
hb faults counts --project-id 12345

# List users affected by a fault
hb faults affected-users --project-id 12345 --id 67890
```

**Note:** All faults commands require `--project-id` (Project ID)

#### list

**Optional:**

 * `-q, --query` - Search query string
 * `--order` - Sort order: `recent` or `frequent`
 * `--limit` - Maximum number of results (max: 25)
 * `-o, --output` - Output format: `table` or `json` (default: `table`)

#### get

**Required:**

 * `--id` - Fault ID

**Optional:**

 * `-o, --output` - Output format: `text` or `json` (default: `text`)

#### notices

**Required:**

 * `--id` - Fault ID

**Optional:**

 * `--limit` - Maximum number of results (max: 25)
 * `-o, --output` - Output format: `table` or `json` (default: `table`)

#### counts

**Optional:**

 * `-o, --output` - Output format: `text` or `json` (default: `text`)

#### affected-users

**Required:**

 * `--id` - Fault ID

**Optional:**

 * `-q, --query` - Search query to filter users
 * `-o, --output` - Output format: `table` or `json` (default: `table`)

See https://docs.honeybadger.io/api/faults/ for more information.

### Insights Command

Execute BadgerQL queries against your Honeybadger Insights data. **Requires**: `--auth-token` or `HONEYBADGER_AUTH_TOKEN` (personal auth token)

```bash
# Basic query for timestamps and previews
hb insights query --project-id 12345 --query "fields @ts, @preview | sort @ts"

# Query with timezone
hb insights query --project-id 12345 --query "fields @ts, @preview | sort @ts" --timezone "America/New_York"

# Query at a specific timestamp
hb insights query --project-id 12345 --query "fields @ts, @preview | sort @ts" --ts "PT1H"

# Output as JSON
hb insights query --project-id 12345 --query "fields @ts, @preview | sort @ts" --output json
```

**Required:**

 * `--project-id` - Project ID
 * `-q, --query` - BadgerQL query to execute

**Optional:**

 * `--ts` - Timestamp range for the query (e.g., `PT1H`)
 * `--timezone` - Timezone for the query (e.g., `America/New_York`)
 * `-o, --output` - Output format: `table` or `json` (default: `table`)

See https://docs.honeybadger.io/api/insights/#query-insights-data for more information.

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
