# aprs-beacon

1. **Fixed-point beacon (Beacon)**: re-sends a set of static coordinates on a fixed schedule — purely time-driven, no change detection.
2. **Traccar mobile reporting (Traccar)**: periodically polls the latest position of [Traccar](https://www.traccar.org/) devices and reports to APRS-IS only when the **movement distance** or the **time since the last report** crosses a threshold.

## Features

- The fixed beacon and Traccar mobile reporting share a single APRS-IS uplink (one reused connection per callsign).
- Clear trigger logic: report when movement ≥ `max_distance_meters` **or** time since last report ≥ `min_update_seconds`; the first observation is always reported (to seed the state).
- Distance uses the `aprsutils` Haversine formula; speed, course and altitude are taken directly from the Traccar API (no incorrect conversions).
- Units strictly match the protocol: Traccar `speed` is in knots, APRS is also in knots, so **no conversion** is applied; `course` is in degrees; `altitude` is converted from metres to feet.
- Structured logging (zap: console + rotating file + a separate error log), with the `aprsutils` logs routed into the same logger.
- Configuration via viper, with `SIGHUP` hot-reload; debug mode is entered automatically when `config_debug.yaml` is present.
- Scheduling via gocron, with `SingletonMode` to prevent overlapping job runs.
- Client masquerading supported: the login `software`/`version` and the per-packet `tocall` are all configurable, falling back to built-in defaults when empty.
- Graceful shutdown: `SIGINT`/`SIGTERM` → stop the scheduler, close all uplinks, flush logs.

## Project layout

```
aprs-beacon/
├── main.go                       # Entry: config → logger → cron.Init → app.Init → cron.Start, then wait for signals and shut down gracefully
├── config.yaml                   # Production config
├── config_debug.yaml             # When present, enables debug mode and overrides config (do not commit real secrets)
└── internal/
    ├── app/app.go                # Composition root: builds the manager and services, registers cron jobs
    ├── aprsis/manager.go         # One aprsutils client per callsign: lazy connect / reuse / reconnect on failure
    ├── packet/
    │   ├── packet.go             # Builds APRS position (!) and status (>) packets (aprsutils only parses, not builds)
    │   └── format.go             # Latitude/longitude/course/speed/altitude formatting and unit conversion
    ├── beacon/beacon.go          # Fixed-point periodic reporting service
    ├── traccar/
    │   ├── client.go             # Traccar /api/positions HTTP client (Basic Auth)
    │   └── service.go            # Polling + movement/timeout trigger logic, per-callsign state
    ├── cron/cron.go              # Global scheduler Init/Start/Stop
    ├── meta/                     # Application identity defaults (Name / Version / ToCall)
    └── infra/
        ├── config/              # viper config loading (model + hot-reload)
        └── logger/              # zap logging (the adapter also satisfies aprsutils.Logger)
```

## Requirements

- Go 1.26+
- Key dependencies: `github.com/APRSCN/aprsutils`, `github.com/go-co-op/gocron`, `github.com/spf13/viper`, `go.uber.org/zap`, `gopkg.in/natefinch/lumberjack.v2`

## Build and run

```sh
go build ./...        # build
go vet ./...          # static analysis
go test ./...         # run tests

go run .              # run directly (reads config.yaml from the current directory)
```

At runtime it reads `config.yaml` from the current directory; if a `config_debug.yaml` exists alongside it, debug mode is entered and that file overrides the config (log level becomes Debug).

Hot-reload: `kill -HUP <pid>` reloads the config file.
Graceful shutdown: `Ctrl-C` or `kill -TERM <pid>`.

## Configuration

See [`config.yaml`](config.yaml) in the repository for a complete example.

### log

| Key | Description |
|---|---|
| `file.all` | Path to the full log file (JSON); leave empty to skip file output |
| `file.err` | Path to the error-only log file (WARN and above); leave empty to skip |
| `max_size` | Per-file rotation size (MB) |
| `max_backups` | Number of rotated files to keep |
| `max_age` | Days to retain rotated files |
| `compress` | Whether to gzip rotated files |

### aprs

| Key | Description |
|---|---|
| `servers` | List of APRS-IS servers (`host`/`port`), tried in order until one connects |
| `software` | Software name advertised in the login line; defaults to `aprs-beacon` when empty |
| `version` | Version advertised in the login line; defaults to `1.0.0` when empty |
| `tocall` | The AX.25 destination (TOCALL) on every packet; defaults to `APBC1` when empty |
| `retry_times` | Reconnection attempts the underlying client makes after a link drop |

> `software`/`version`/`tocall` are used to masquerade as a particular client; all are optional and fall back to the built-in defaults from `internal/meta` when empty.

### beacon (fixed-point)

| Key | Description |
|---|---|
| `enabled` | Whether enabled |
| `interval` | Re-send period (seconds) |
| `stations[]` | List of fixed stations |

Each station: `callsign`, `latitude`, `longitude`, `symbol` (two-character APRS symbol; a backslash table id must be written as `"\\"`), `comment`, `info` (optional, sent as an additional status report).

### traccar (mobile reporting)

| Key | Description |
|---|---|
| `enabled` | Whether enabled |
| `interval` | Polling period (seconds) |
| `request_timeout` | Per-request HTTP timeout (seconds) |
| `expire_seconds` | Position staleness threshold (seconds); with a device's `skip_old`, drops fixes that are too old |
| `min_update_seconds` | Force a report when this much time has elapsed since the last one |
| `max_distance_meters` | Report when movement exceeds this distance |
| `devices[]` | List of tracked devices |

Each device: in addition to the station fields (`callsign`/`symbol`/`comment`/`info`), it has `url` (Traccar address), `account`/`password` (Basic Auth), `device` (device ID), and `skip_old` (whether to drop stale fixes).

> Security note: `config.yaml` contains plaintext passwords; do not commit real credentials to version control.

## APRS packet format

Each report produces 1–2 packet lines (only the position packet when there is no `Info`):

```
<callsign>><tocall>,TCPIP*:!<lat><table><lon><code>[CSE/SPD][/A=AAAAAA]<comment>
<callsign>><tocall>,TCPIP*:><info>
```

- The position is uncompressed: latitude `DDMM.mmN/S`, longitude `DDDMM.mmE/W`; the two-character `symbol` is split into table and code.
- When speed or course is present, the `CSE/SPD` extension is appended (the missing component is sent as 0); when altitude is present, the `/A=` extension is appended (feet, 6 digits).
- Speed is in knots, course is 0–360 degrees, and altitude is converted from metres to feet.

## How it works

`main.go` initializes the config, logger and scheduler in order. `internal/app` acts as the composition root: it builds the `aprsis.Manager` and the beacon/traccar services, registers their `Run` methods as periodic gocron jobs, then starts the scheduler and blocks waiting for a shutdown signal.

- **Beacon**: each cycle iterates over all fixed stations, builds position packets and sends them.
- **Traccar**: each cycle fetches the latest position for each device, decides whether to report based on the movement/timeout triggers, and updates that callsign's last position and time after reporting.
