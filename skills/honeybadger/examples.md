# Honeybadger CLI Examples

## Investigating a Production Error

When a user reports an error, use this workflow to investigate:

```bash
# 1. Find the project
hb projects list

# 2. Search for the error by class name or message
hb faults list --project-id 12345 --query "NoMethodError" --order recent

# 3. Get detailed fault information
hb faults get --project-id 12345 --fault-id 67890

# 4. View recent occurrences (notices) of this error
hb faults notices --project-id 12345 --fault-id 67890 --limit 10

# 5. Find which users were affected
hb faults affected-users --project-id 12345 --fault-id 67890
```

## Analyzing Error Trends

```bash
# Get fault counts to see error volume
hb faults counts --project-id 12345 --occurred-after "2024-01-01T00:00:00Z"

# Get occurrence counts grouped by day
hb projects occurrences --project-id 12345 --period day

# Get a report of errors by class
hb projects reports --project-id 12345 --report notices_by_class
```

## Querying Insights Data

BadgerQL allows SQL-like queries against your application data:

```bash
# Count requests in the last 3 hours (default)
hb insights query --project-id 12345 --query "SELECT count() FROM requests"

# Find slow requests this week
hb insights query --project-id 12345 --ts week \
  --query "SELECT path, avg(duration) FROM requests WHERE duration > 1000 GROUP BY path"

# Analyze errors by environment
hb insights query --project-id 12345 --ts today \
  --query "SELECT environment, count() FROM errors GROUP BY environment"

# Search logs
hb insights query --project-id 12345 \
  --query "SELECT message, severity FROM logs WHERE message CONTAINS 'timeout'"
```

## Checking Application Health

```bash
# Check uptime monitor status
hb uptime sites list --project-id 12345

# View recent outages
hb uptime outages list --project-id 12345 --site-id abc123 --limit 5

# Check if scheduled jobs are running (check-ins)
hb checkins list --project-id 12345
```

## Working with JSON Output

For programmatic use or piping to other tools:

```bash
# Get projects as JSON
hb projects list --output json

# Get faults as JSON and pipe to jq
hb faults list --project-id 12345 --output json | jq '.[] | {id, message: .klass}'

# Get account info as JSON
hb accounts list --output json
```

## Managing Team Access

```bash
# List your accounts
hb accounts list

# View teams in an account
hb teams list --account-id abc123

# See who's on a team
hb teams members list --team-id 456

# Check pending invitations
hb teams invitations list --team-id 456
```

## Viewing Recent Deployments

```bash
# List recent deployments
hb deployments list --project-id 12345 --limit 10

# Filter by environment
hb deployments list --project-id 12345 --environment production

# Get deployment details
hb deployments get --project-id 12345 --id 789
```

## Common Workflows

### "What errors happened after the last deploy?"

```bash
# Get last deployment time
hb deployments list --project-id 12345 --limit 1 --output json

# List faults since that time
hb faults list --project-id 12345 --occurred-after "2024-01-15T10:30:00Z" --order recent
```

### "Which errors are affecting the most users?"

```bash
# List faults ordered by frequency
hb faults list --project-id 12345 --order frequent --limit 10

# For each fault, check affected users
hb faults affected-users --project-id 12345 --fault-id <id>
```

### "Is our background job running?"

```bash
# List check-ins to find the job
hb checkins list --project-id 12345

# Get details including last check-in time
hb checkins get --project-id 12345 --id <checkin_id>
```
