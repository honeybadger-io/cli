# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the Honeybadger CLI (`hb`), a command-line tool for interacting with Honeybadger's APIs. It includes both traditional CLI commands and a terminal UI (TUI) for browsing Honeybadger data.

## Commands

```bash
# Build the CLI
go build -o hb .

# Run tests
go test ./...

# Run a single test
go test -run TestFunctionName ./...

# Lint (required before completing tasks)
golangci-lint run ./...
```

## Architecture

### Two Packages

- **cmd/**: Cobra-based CLI commands using spf13/cobra and spf13/viper
- **tui/**: Terminal UI using rivo/tview and gdamore/tcell

### CLI Commands (cmd/)

Commands are organized by API type:
- **Reporting API** (`--api-key`): `deploy`, `agent` - sends data to Honeybadger
- **Data API** (`--auth-token`): `projects`, `faults`, `insights`, `checkins`, `uptime`, `teams`, `accounts`, `statuspages` - reads/manages Honeybadger data

Each command file follows the Cobra pattern with `init()` registering subcommands and flags. Configuration precedence: CLI flags > environment variables (`HONEYBADGER_*`) > config file.

The `convertEndpointForDataAPI()` helper in root.go converts `api.honeybadger.io` to `app.honeybadger.io` for Data API calls.

### TUI (tui/)

Stack-based navigation system built on tview:

- **app.go**: Main `App` struct manages navigation stack, header/footer, and global input handling
- **View interface**: All views implement `Name()`, `Render()`, `Refresh()`, `HandleInput()`
- **helpers.go**: Shared utilities including `setupReadOnlyTable()` (dim highlight for leaf views without drill-down)

Navigation flow: Accounts → Account Menu → Projects/Teams/Users → Project Menu → Faults/Deployments/Uptime/Check-ins

Views with drill-down capability use `SetSelectedFunc()` and cyan highlighting. Read-only leaf views (notices, members, invitations, etc.) use dimmer highlighting to indicate no further navigation.

### API Client Dependency

Uses `github.com/honeybadger-io/api-go` for API interactions. For local development with both repos, use a Go workspace:

```bash
# From parent directory containing both cli and api-go
go work init
go work use ./cli ./api-go
```

When updating dependencies for CI (without workspace): `GOWORK=off go mod tidy`

## Before Completing Tasks

- Always run the linter (`golangci-lint run ./...`) before marking a task as done
- Fix any linter issues that are reported
