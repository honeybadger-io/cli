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
```

### Environment Variables

You can set configuration using environment variables prefixed with `HONEYBADGER_`:

```bash
export HONEYBADGER_API_KEY=your-api-key-here
```

### Command-line Flags

Global flags that apply to all commands:

- `--api-key`: Your Honeybadger API key
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

## Development

When adding new functionality, please also add corresponding tests.

To run tests locally:
```bash
go test ./...
```

### Releasing

To publish a new release:

1. Create and push a new tag with the version number:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. The release workflow will automatically:
   - Build binaries for multiple platforms (Linux, macOS, Windows)
   - Create a GitHub release with the binaries
   - Generate a changelog from commit messages

   The binaries will be available for download from the GitHub releases page.

Note: Commits with messages containing `[skip ci]` will skip the test workflow, but the release workflow will still run when a tag is pushed.

## License

MIT License
