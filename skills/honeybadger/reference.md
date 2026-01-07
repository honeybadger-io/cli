# Honeybadger CLI Reference

## Authentication

All Data API commands require authentication via personal auth token:

```bash
# Via environment variable (recommended)
export HONEYBADGER_AUTH_TOKEN=your_token

# Or via flag
hb projects list --auth-token your_token
```

## Projects

### List Projects
```bash
hb projects list [--output json|table]
```

### Get Project Details
```bash
hb projects get --id <project_id> [--output json|text]
```

### Get Project Occurrence Counts
```bash
hb projects occurrences --project-id <id> [--period hour|day|week|month] [--environment <env>]
```

### Get Project Reports
```bash
hb projects reports --project-id <id> --report <type> [--start <iso8601>] [--stop <iso8601>]
```
Report types: `notices_by_class`, `notices_by_location`, `notices_by_user`, `notices_per_day`

### Get Project Integrations
```bash
hb projects integrations --project-id <id>
```

## Faults (Errors)

### List Faults
```bash
hb faults list --project-id <id> [options]
```
Options:
- `--query, -q`: Search/filter string
- `--created-after`: ISO 8601 timestamp
- `--occurred-after`: ISO 8601 timestamp
- `--occurred-before`: ISO 8601 timestamp
- `--limit`: Max results (1-25)
- `--order`: Sort order (recent, frequent)
- `--page`: Page number
- `--output`: Output format (json, table)

### Get Fault Details
```bash
hb faults get --project-id <id> --fault-id <fault_id> [--output json|text]
```

### Get Fault Counts
```bash
hb faults counts --project-id <id> [--query <q>] [--created-after <ts>]
```

### List Fault Notices
```bash
hb faults notices --project-id <id> --fault-id <fault_id> [--limit <n>] [--created-after <ts>]
```

### List Affected Users
```bash
hb faults affected-users --project-id <id> --fault-id <fault_id> [--query <q>]
```

## Insights

### Query Insights Data
```bash
hb insights query --project-id <id> --query "<badgerql>" [--ts <timerange>] [--timezone <tz>]
```
Time range shortcuts: `today`, `week`, or ISO 8601 duration (e.g., `PT3H` for 3 hours)

## Accounts

### List Accounts
```bash
hb accounts list [--output json|table]
```

### Get Account Details
```bash
hb accounts get --id <account_id> [--output json|text]
```

### List Account Users
```bash
hb accounts users list --account-id <id>
```

### List Account Invitations
```bash
hb accounts invitations list --account-id <id>
```

## Teams

### List Teams
```bash
hb teams list --account-id <id> [--output json|table]
```

### Get Team Details
```bash
hb teams get --id <team_id> [--output json|text]
```

### List Team Members
```bash
hb teams members list --team-id <id>
```

### List Team Invitations
```bash
hb teams invitations list --team-id <id>
```

## Uptime Monitoring

### List Uptime Sites
```bash
hb uptime sites list --project-id <id> [--output json|table]
```

### Get Uptime Site Details
```bash
hb uptime sites get --project-id <id> --site-id <site_id>
```

### List Outages
```bash
hb uptime outages list --project-id <id> --site-id <site_id> [--limit <n>]
```

### List Uptime Checks
```bash
hb uptime checks list --project-id <id> --site-id <site_id> [--limit <n>]
```

## Check-ins (Cron Monitoring)

### List Check-ins
```bash
hb checkins list --project-id <id> [--output json|table]
```

### Get Check-in Details
```bash
hb checkins get --project-id <id> --id <checkin_id>
```

## Comments

### List Fault Comments
```bash
hb comments list --project-id <id> --fault-id <fault_id>
```

### Get Comment Details
```bash
hb comments get --project-id <id> --fault-id <fault_id> --id <comment_id>
```

## Deployments

### List Deployments
```bash
hb deployments list --project-id <id> [--environment <env>] [--limit <n>]
```

### Get Deployment Details
```bash
hb deployments get --project-id <id> --id <deployment_id>
```

## Environments

### List Environments
```bash
hb environments list --project-id <id>
```

## Status Pages

### List Status Pages
```bash
hb statuspages list --account-id <id>
```

### Get Status Page Details
```bash
hb statuspages get --id <statuspage_id>
```
