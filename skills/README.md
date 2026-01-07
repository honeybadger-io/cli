# Honeybadger CLI Skill for Claude Code

This directory contains a Claude Code skill that enables Claude to interact with Honeybadger using the `hb` CLI tool. This is an alternative to the [Honeybadger MCP Server](https://github.com/honeybadger-io/honeybadger-mcp-server) for users who prefer a skill-based approach.

## Prerequisites

1. **Install the Honeybadger CLI**

   Download from the [releases page](https://github.com/honeybadger-io/cli/releases) or install via Homebrew:

   ```bash
   brew install honeybadger-io/tap/honeybadger
   ```

2. **Get a Personal Auth Token**

   Get your personal token from [app.honeybadger.io/users/edit](https://app.honeybadger.io/users/edit)

3. **Set the environment variable**

   ```bash
   export HONEYBADGER_AUTH_TOKEN=your_token_here
   ```

   Or add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.)

## Installation

### Option 1: Personal Skill (just for you)

Copy the skill to your personal Claude skills directory:

```bash
mkdir -p ~/.claude/skills
cp -r honeybadger ~/.claude/skills/
```

### Option 2: Project Skill (for your team)

Copy to your project's `.claude/skills` directory:

```bash
mkdir -p /path/to/your/project/.claude/skills
cp -r honeybadger /path/to/your/project/.claude/skills/
```

## Usage

Once installed, Claude will automatically use this skill when you ask about Honeybadger errors, monitoring, or application health. Examples:

- "What errors have occurred in production today?"
- "Show me the most frequent errors in project 12345"
- "Check if our uptime monitors are healthy"
- "Query insights for slow requests this week"
- "Who was affected by that NoMethodError?"

## Capabilities

The skill provides access to:

- **Projects** - List, view, and get reports/statistics
- **Faults** - Search errors, view occurrences, find affected users
- **Insights** - Run BadgerQL queries against your application data
- **Uptime** - Monitor sites, view outages and checks
- **Check-ins** - Monitor scheduled jobs/cron tasks
- **Teams & Accounts** - View organization structure
- **Deployments** - View deployment history
- **And more** - Comments, environments, status pages

## Comparison with MCP Server

| Feature | CLI Skill | MCP Server |
|---------|-----------|------------|
| Installation | Copy files | Configure MCP |
| Authentication | Environment variable | Environment variable |
| Write operations | Full support | Opt-in (`read-only=false`) |
| Offline docs | Yes (in skill files) | No |
| Customization | Edit markdown files | Code changes |

Choose the CLI skill if you prefer a simpler setup or want to customize the instructions. Choose the MCP server if you want tighter integration with the MCP protocol.

## Files

- `honeybadger/SKILL.md` - Main skill definition with quick reference
- `honeybadger/reference.md` - Complete command documentation
- `honeybadger/examples.md` - Practical usage examples
