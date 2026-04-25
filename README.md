# Exorcist

A Signal group chat ghost reaper - automatically removes users who haven't participated in group chats for N days.

## Features

- **Automatic initialization**: On first run, seeds all existing members with current timestamp
- Monitors Signal group chats for member activity
- **Maximum privacy**: Only stores Signal user IDs (required for API) and timestamps - no phone numbers, no names, no message content
- Tracks last message timestamp for each member
- Auto-adds new members when they send their first message
- Automatically removes inactive members after configurable days
- Announces removals in the group chat for transparency (Signal shows who was removed)
- Configurable inactivity threshold per group
- Runs on a cron schedule (default: sync hourly, reap daily)

## Prerequisites

- Go 1.25 or higher
- `signal-cli` installed and configured
- A Signal account registered with `signal-cli`

## Installation

### 1. Install signal-cli

```bash
# macOS (via Homebrew)
brew install signal-cli

# Linux
# Follow instructions at https://github.com/AsamK/signal-cli
```

### 2. Register Signal Account

```bash
# Link your Signal account (recommended for existing accounts)
signal-cli link -n exorcist

# OR register a new number
signal-cli -a +1234567890 register
signal-cli -a +1234567890 verify CODE
```

### 3. Install Exorcist

```bash
git clone https://github.com/joel-g/exorcist.git
cd exorcist
go build -o exorcist ./cmd/exorcist
```

## Configuration

Create a `.env` file or set environment variables:

```bash
# Required: Your Signal account phone number (must match signal-cli account)
SIGNAL_ACCOUNT=+1234567890

# Optional: Database path (default: ./exorcist.db)
DB_PATH=/path/to/exorcist.db

# Optional: Cron schedules (cron format)
SYNC_SCHEDULE="0 * * * *"    # Every hour (default)
REAP_SCHEDULE="0 2 * * *"    # Daily at 2am (default)
```

### Cron Schedule Format

Format: `minute hour day month weekday`

Examples:
- `"0 * * * *"` - Every hour
- `"0 2 * * *"` - Daily at 2am
- `"*/30 * * * *"` - Every 30 minutes
- `"0 */6 * * *"` - Every 6 hours

## Quick Start

```bash
# 1. Get your Signal groups
signal-cli -a +1234567890 listGroups --detailed

# 2. Add groups to monitor
sqlite3 exorcist.db
INSERT INTO groups (signal_group_id, name, inactivity_days)
VALUES ('your-group-id-here', 'Group Name', 30);
.exit

# 3. Run Exorcist
export SIGNAL_ACCOUNT="+1234567890"
./exorcist
```

On first run, Exorcist will automatically initialize all existing group members with the current timestamp, and the inactivity countdown begins.

## Detailed Usage

### 1. Set Up Groups to Monitor

Before running Exorcist, add the groups you want to monitor to the database.

**Step 1: Get your group IDs from signal-cli:**

```bash
# List all your Signal groups
signal-cli -a +1234567890 listGroups --detailed
```

This will output something like:
```
Id: AbCdEf123456==  Name: My Group Chat  Active: true  ...
Id: XyZ789AbCdEf==  Name: Another Group  Active: true  ...
```

**Step 2: Add groups to the database:**

```bash
# Create/open the database
sqlite3 exorcist.db

# Add each group you want to monitor
INSERT INTO groups (signal_group_id, name, inactivity_days)
VALUES ('AbCdEf123456==', 'My Group Chat', 30);

INSERT INTO groups (signal_group_id, name, inactivity_days)
VALUES ('XyZ789AbCdEf==', 'Another Group', 45);

# Exit sqlite
.exit
```

**Note:** `inactivity_days` is configurable per group. Set it to however many days of silence before removal.

### 2. Run Exorcist

**Option 1: Using the helper script (easiest for local usage)**

```bash
export SIGNAL_ACCOUNT="+1234567890"
./run.sh
```

This will automatically start Exorcist in a tmux session so it keeps running in the background.

**Option 2: Run directly**

```bash
export SIGNAL_ACCOUNT="+1234567890"
./exorcist
```

**Option 3: Using tmux manually**

```bash
tmux new -s exorcist
export SIGNAL_ACCOUNT="+1234567890"
./exorcist

# Detach: Ctrl+B, then D
# Reattach: tmux attach -t exorcist
```

### 3. First Run - Automatic Initialization

On first run, Exorcist automatically:
1. Detects any groups with no members in the database
2. Fetches all current members from those Signal groups
3. Adds them to the database with `last_message_at = NOW()`
4. Starts the inactivity countdown from that moment

This means:
- **Everyone gets a fair chance**: All existing members start with the same timestamp
- **The countdown begins now**: Anyone who doesn't message in the next N days will be removed
- **No manual setup needed**: Just add groups to the database and run Exorcist

After initialization, new members who join the group will be automatically tracked when they send their first message.

## How It Works

1. **Initialization Job** (runs once on first startup):
   - Fetches all members from monitored Signal groups
   - Adds them to the database with current timestamp
   - Logs the initialization for transparency

2. **Sync Job** (runs hourly by default):
   - Fetches new messages from Signal using `signal-cli receive`
   - Extracts only timestamps (never stores message content)
   - Auto-adds new members when they send their first message
   - Updates `last_message_at` for each user who sent a message
   - Only processes messages from monitored groups

3. **Reap Job** (runs daily at 2am by default):
   - Queries database for users who haven't messaged in N days
   - Sends an announcement message to the group explaining why the removal is happening
   - Removes them from the Signal group using `signal-cli` (Signal shows who was removed)
   - Removes them from the database

**Example announcement message:**
```
🤖 Automated removal - last activity March 15, 2026 - inactive for 30+ days.
```

Signal will automatically show "[User] left" or similar, so the announcement just explains *why* the removal happened.

## Deployment

### Railway

1. Create a new Railway project
2. Add environment variables:
   - `SIGNAL_ACCOUNT`
   - `DB_PATH=/app/data/exorcist.db` (Railway provides persistent `/app/data`)
3. Deploy:

```bash
railway up
```

### Docker

```bash
# Build
docker build -t exorcist .

# Run
docker run -d \
  -e SIGNAL_ACCOUNT="+1234567890" \
  -v $(pwd)/data:/data \
  -v ~/.local/share/signal-cli:/root/.local/share/signal-cli \
  exorcist
```

## Database Schema

### groups
- `id`: Primary key
- `signal_group_id`: Signal's internal group ID
- `name`: Human-readable group name
- `inactivity_days`: Days of inactivity before removal
- `created_at`: When group was added to monitoring

### group_members
- `id`: Primary key
- `group_id`: Foreign key to groups
- `signal_user_id`: Signal user UUID or phone number (required for Signal API calls)
- `last_message_at`: Timestamp of last message (NULL if never messaged)
- `joined_at`: When user was added to tracking
- `created_at`: When record was created

**Privacy**: Only the minimum data needed for functionality is stored. No phone numbers, no names, no message content.

## Limitations

- Signal doesn't have an official bot API - uses `signal-cli` which may break
- Your Signal account must be an admin in all monitored groups
- Cannot access message history (disappearing messages)
- Requires continuous operation to track activity in real-time

## License

MIT
