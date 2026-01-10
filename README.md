# tracktime CLI

Lightweight CLI for receiving heartbeats from editor extensions and syncing them to the tracktime.sh backend.

## Installation

### From Source

```bash
cd cli
go build -o tracktime ./cmd/tracktime
```

### Pre-built Binaries

Download from releases for your platform:
- `tracktime-darwin-amd64` (macOS Intel)
- `tracktime-darwin-arm64` (macOS Apple Silicon)
- `tracktime-linux-amd64`
- `tracktime-linux-arm64`
- `tracktime-windows-amd64.exe`

## Authentication

```bash
tracktime login   # Opens browser for OAuth
tracktime logout  # Remove stored credentials
```

## Commands

### login

Authenticate with tracktime.sh via browser-based OAuth flow. Opens your default browser to complete authentication.

```bash
tracktime login
```

**Exit codes:**
- `0` - Successfully authenticated
- `1` - Authentication failed or cancelled

### logout

Remove stored credentials.

```bash
tracktime logout
```

**Exit codes:**
- `0` - Successfully logged out
- `1` - Error removing credentials

### whoami

Show the currently logged in user.

```bash
tracktime whoami
```

```
Logged in as: user@example.com
User ID:      usr_abc123
```

**Exit codes:**
- `0` - Successfully retrieved user info
- `1` - Not logged in or API key invalid

### status

Show current configuration and queue status.

```bash
tracktime status
```

```
Configured: yes
API URL:    https://tracktime.sh/api/v1
Machine ID: a1b2c3d4...
Queue size: 5 heartbeats
Last sync:  2 minutes ago
```

**Exit codes:**
- `0` - Configured and ready
- `2` - Not configured

### heartbeat

Receive heartbeat data from editor extensions. Queues locally and opportunistically syncs to backend.

```bash
# From JSON string
tracktime heartbeat --json '{"timestamp":"2025-01-10T12:00:00Z","editor":"vscode"}'

# From stdin
echo '{"timestamp":"...","editor":"vscode"}' | tracktime heartbeat

# From file
tracktime heartbeat --file /path/to/heartbeat.json
```

**Required fields:**
- `timestamp` - ISO 8601 format
- `editor` - Editor name (e.g., `vscode`, `neovim`, `jetbrains`)

**Optional fields:**
- `project` - Project name
- `language` - Programming language
- `file_path` - Current file path
- `is_write` - Whether this was a save event

**Output:**
```json
{"status":"queued","queue_size":5,"synced":false}
```

**Exit codes:**
- `0` - Success (queued and optionally synced)
- `1` - Invalid input
- `2` - Not configured (no API key)
- `3` - Queue error

### flush

Force sync all queued heartbeats to the backend.

```bash
tracktime flush
```

**Output:**
```json
{"synced":42,"remaining":0,"errors":0}
```

**Exit codes:**
- `0` - All synced successfully
- `1` - Partial sync (some errors)
- `2` - Not configured
- `3` - Network error

## Editor Extension Integration

Extensions should call the CLI via subprocess. The `status` and `whoami` commands support a `--json` flag for structured output:

```bash
tracktime status --json
tracktime whoami --json
```

```json
{"configured":true,"api_url":"https://tracktime.sh/api/v1","machine_id":"...","queue_size":5,"last_sync":"2 minutes ago","last_sync_raw":"2025-01-10T12:00:00Z"}
```

```json
{"email":"user@example.com","user_id":"usr_abc123"}
```

The CLI handles local queueing in SQLite, opportunistic syncing, machine ID tracking, and retry logic. Extensions should send heartbeats on file open, file save (`is_write: true`), and periodic activity while typing.

## Data Storage

- Config: `~/.tracktime/config.json`
- Queue: `~/.tracktime/queue.db` (SQLite)

## Building for Release

```bash
GOOS=darwin GOARCH=amd64 go build -o dist/tracktime-darwin-amd64 ./cmd/tracktime
GOOS=darwin GOARCH=arm64 go build -o dist/tracktime-darwin-arm64 ./cmd/tracktime
GOOS=linux GOARCH=amd64 go build -o dist/tracktime-linux-amd64 ./cmd/tracktime
GOOS=linux GOARCH=arm64 go build -o dist/tracktime-linux-arm64 ./cmd/tracktime
GOOS=windows GOARCH=amd64 go build -o dist/tracktime-windows-amd64.exe ./cmd/tracktime
```

## License

MIT
