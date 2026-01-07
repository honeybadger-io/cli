---
name: honeybadger
description: Interact with Honeybadger error tracking and monitoring. List projects, view faults/errors, query insights data, manage uptime monitors, check-ins, teams, and more. Use this when the user wants to investigate errors, check application health, or manage their Honeybadger configuration.
allowed-tools:
  - Bash
  - Read
---

# Honeybadger CLI Skill

Use the `hb` CLI to interact with Honeybadger's APIs.

## Prerequisites

The CLI must be installed and available in the PATH as `hb`. Authentication requires a personal auth token set via:
- Environment variable: `HONEYBADGER_AUTH_TOKEN`
- Or flag: `--auth-token <token>`

## Quick Reference

### Investigating Errors

```bash
# List projects to find the project ID
hb projects list

# List recent faults for a project
hb faults list --project-id <id>

# Get details about a specific fault
hb faults get --project-id <id> --fault-id <fault_id>

# See individual error occurrences (notices)
hb faults notices --project-id <id> --fault-id <fault_id>

# Find users affected by an error
hb faults users --project-id <id> --fault-id <fault_id>
```

### Querying Insights

```bash
# Run a BadgerQL query
hb insights query --project-id <id> --query "SELECT count() FROM requests"

# Query with time range
hb insights query --project-id <id> --query "SELECT count() FROM logs" --ts week
```

### Common Operations

```bash
# List accounts
hb accounts list

# List teams for an account
hb teams list --account-id <id>

# Check uptime monitors
hb uptime sites list --project-id <id>

# View check-ins (cron monitoring)
hb checkins list --project-id <id>
```

## Output Formats

Most commands support `--output json` for machine-readable output:

```bash
hb projects list --output json
hb faults list --project-id <id> --output json
```

## Detailed Reference

For complete command documentation, see [reference.md](./reference.md).

For usage examples, see [examples.md](./examples.md).
