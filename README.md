# Honeybadger CLI

[![Test](https://github.com/honeybadger-io/cli/actions/workflows/test.yml/badge.svg)](https://github.com/honeybadger-io/cli/actions/workflows/test.yml)

A command-line interface for interacting with Honeybadger's Reporting API.

## Installation

```bash
go install github.com/honeybadger-io/cli@latest
```

## Configuration

The CLI can be configured using either command-line flags, environment variables, or a configuration file.

### Configuration File

By default, the CLI looks for a configuration file at `config/honeybadger.yml` in the current directory. You can specify a different configuration file using the `--config` flag.

Example configuration file:
```yaml
api_key: your-api-key-here
endpoint: https://api.honeybadger.io  # Optional, use https://eu-api.honeybadger.io for EU region
```

### Environment Variables

You can set configuration using environment variables prefixed with `HONEYBADGER_`:

```bash
export HONEYBADGER_API_KEY=your-api-key-here
export HONEYBADGER_ENDPOINT=https://eu-api.honeybadger.io  # Optional, for EU region
```

### Command-line Flags

Global flags that apply to all commands:

- `--api-key`: Your Honeybadger API key
- `--endpoint`: Honeybadger endpoint (default: https://api.honeybadger.io)
- `--config`: Path to configuration file

## Usage

### Deploy Command

Report a deployment to Honeybadger:

```bash
hb deploy --environment production --repository github.com/org/repo --revision abc123 --user johndoe
```

Required flags:
- `-e, --environment`: Environment being deployed to (e.g., production)

Optional flags:
- `-r, --repository`: Repository being deployed
- `-v, --revision`: Revision being deployed
- `-u, --user`: Local username of the person deploying

### Run Command

Report a check-in to Honeybadger using either an ID or a slug:

```bash
# Using ID
hb run --id check-123

# Using slug
hb run --slug daily-backup
```

Required flags (one of):
- `-i, --id`: Check-in ID to report
- `-s, --slug`: Check-in slug to report

## Development

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
