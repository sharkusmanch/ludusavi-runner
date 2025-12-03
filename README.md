# ludusavi-runner

A cross-platform service that automates [Ludusavi](https://github.com/mtkennerly/ludusavi) game save backups and exports metrics to Prometheus via Pushgateway.

## Features

- **Automated backups**: Runs Ludusavi backup and cloud upload on a configurable interval
- **Prometheus metrics**: Pushes backup statistics to Pushgateway for monitoring
- **Notifications**: Sends alerts via Apprise on failures (configurable)
- **Windows service**: Runs as a proper Windows service
- **Flexible configuration**: CLI flags, environment variables, and config file support

## Installation

### From Release

Download the latest release from the [releases page](https://github.com/sharkusmanch/ludusavi-runner/releases).

### From Source

```bash
go install github.com/sharkusmanch/ludusavi-runner/cmd/ludusavi-runner@latest
```

## Quick Start

1. Create a config file at `%APPDATA%\ludusavi-runner\config.toml` (Windows) or `~/.config/ludusavi-runner/config.toml` (Linux/macOS):

```toml
interval = "20m"
backup_on_startup = true
pushgateway_url = "http://pushgateway:9091"

[apprise]
enabled = true
url = "http://apprise:8000"
key = "ludusavi"
notify = "error"
```

2. Test the configuration:

```bash
ludusavi-runner validate
```

3. Run a single backup:

```bash
ludusavi-runner run
```

4. Install as a service (Windows):

```bash
ludusavi-runner install --username ".\YourUsername"
ludusavi-runner start
```

## Usage

```
ludusavi-runner [command] [flags]

Commands:
  run           Run a single backup cycle and exit
  serve         Run the service in foreground
  install       Install as a system service
  uninstall     Remove the system service
  start         Start the installed service
  stop          Stop the installed service
  status        Show service status
  validate      Validate configuration and test connectivity
  version       Show version information

Global Flags:
  -c, --config string     Path to config file
      --dry-run           Simulate operations without running ludusavi
      --log-level string  Log level (debug, info, warn, error)
  -h, --help              Help for ludusavi-runner
```

## Configuration

Configuration is loaded from (in order of precedence):
1. CLI flags
2. Environment variables (prefix: `LUDUSAVI_RUNNER_`)
3. Config file
4. Defaults

### Config File

See [config.example.toml](config.example.toml) for all available options.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `LUDUSAVI_RUNNER_INTERVAL` | Backup interval |
| `LUDUSAVI_RUNNER_BACKUP_ON_STARTUP` | Run backup on service start |
| `LUDUSAVI_RUNNER_PUSHGATEWAY_URL` | Pushgateway URL |
| `LUDUSAVI_RUNNER_APPRISE_URL` | Apprise server URL |
| `LUDUSAVI_RUNNER_APPRISE_KEY` | Apprise notification key |
| `LUDUSAVI_RUNNER_APPRISE_NOTIFY` | Notification level (error, warning, always) |
| `LUDUSAVI_RUNNER_LOG_LEVEL` | Log level |

## Metrics

The following metrics are pushed to Pushgateway:

| Metric | Type | Description |
|--------|------|-------------|
| `ludusavi_runner_up` | gauge | Service is running (1=up) |
| `ludusavi_runner_info` | gauge | Build information |
| `ludusavi_last_run_timestamp_seconds` | gauge | Unix timestamp of last run |
| `ludusavi_last_run_success` | gauge | 1=success, 0=failure |
| `ludusavi_last_run_duration_seconds` | gauge | Duration of last run |
| `ludusavi_games_total` | gauge | Total games detected |
| `ludusavi_games_processed` | gauge | Games processed |
| `ludusavi_bytes_total` | gauge | Total bytes |
| `ludusavi_bytes_processed` | gauge | Bytes processed |
| `ludusavi_games_new` | gauge | New games backed up |
| `ludusavi_games_changed` | gauge | Games with changes |

All metrics include an `operation` label (`backup` or `cloud_upload`).

## Development

### Prerequisites

- Go 1.22+
- [golangci-lint](https://golangci-lint.run/) (for linting)

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

## License

MIT
