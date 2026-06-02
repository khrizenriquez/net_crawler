# Duku Net Lab

Duku Net Lab is a local-first Wi-Fi observability laboratory for a consented home network. It is designed for macOS Apple Silicon, a Huawei HG8245W5-6T gateway, Podman Compose, and supervised AI-assisted development.

The product records compact metrics and irreversibly redacted findings. It is not a credential collector. Metrics are aggregated per minute, coverage, device MAC, remote IP, domain and protocol. Raw channel-wide PCAP files are transient staging artifacts: the helper writes `.pcap.partial`, atomically publishes `.pcap` only after capture closes, and the worker filters published files against confirmed BSSID values before deleting the raw files on every path.

## V1 boundaries

- One Mac Wi-Fi interface observes one selected channel at a time.
- On macOS `26.4.1`, Apple's built-in tools no longer expose channel switching. The v1 helper captures only when the requested channel matches the current Wi-Fi channel. Configured multi-channel rotation remains capability-blocked until a compatible adapter backend is added.
- The integrated Mac radio may expose zero frames in monitor mode. The explicit `local-host` fallback captures real associated traffic for this Mac only in one-minute, 128 MiB capped segments and labels its metrics `partial`; it does not observe phones, TVs or other clients.
- Ethernet clients are out of scope for individual inspection.
- WPA2 Personal decryption is optional, passive, and possible only when the relevant handshake was observed.
- HTTPS payloads remain opaque.
- WAN totals are available only if the optional read-only Huawei adapter discovers compatible firmware counters.
- Neighbor networks may appear incidentally at the radio boundary, but their frames must be discarded before analysis or retention.

These are capability boundaries, not dashboard caveats. See [ADR-0008](docs/adr/0008-v2-capture-sources.md) for v2 expansion paths.

## Components

| Component | Runtime | Responsibility |
| --- | --- | --- |
| `duku-api` | Go container | REST, SSE, validation, demo data and host-command queue |
| `duku-worker` | Go + local `tshark` container | Authorized-BSSID filtering, explicit local-host handling and analysis boundary |
| PostgreSQL | Podman volume | Durable operational schema for production persistence |
| Dashboard | React container | Spanish localhost-only operational UI |
| `duku-capture-helper` | Restricted root-owned host binary | `probe`, `start`, `start-local`, `stop`, `status`, `restore` only |
| Duku Net Lab menu app | SwiftUI macOS app | Start/stop Podman, open dashboard and show local state |
| Harness | Go CLI | Prepare supervised Codex tasks in isolated Git worktrees |

## First run

### 1. Install requirements

Required for the local dashboard and demo stack:

- macOS Apple Silicon.
- Podman Desktop or the Podman CLI.
- Xcode command-line tools.
- `openssl`, available by default on macOS or through Homebrew.

Optional tools:

- Node.js and npm: only for running dashboard tests directly on the host.
- A Go toolchain: optional; `scripts/go.sh` caches a local Apple Silicon toolchain when Go is absent.
- `codex-cli`: only for harness-driven development.

Verify the core tools:

```sh
xcode-select -p
podman --version
podman-compose --version
openssl version
```

### 2. Prepare the Podman machine

Check whether a machine already exists:

```sh
podman machine list
```

If no machine exists, initialize one once:

```sh
podman machine init --cpus 4 --memory 4096 --now
```

If a machine exists but is stopped:

```sh
podman machine start
```

### 3. Create private local configuration

From the repository root:

```sh
make bootstrap
make configure-local-env
```

`make bootstrap` creates `~/.duku-net-lab`, copies `.env.example` to the ignored local `.env` file and sets mode `600`. `make configure-local-env` writes that absolute private data path and generates a local PostgreSQL password and host token without printing them.

Validate the result:

```sh
scripts/check-local-secrets.sh .env
```

Do not commit `.env`. The repository safety check rejects accidental publication.

### 4. Start the demo stack

```sh
make up
make status
open http://127.0.0.1:4173
```

The first build may take several minutes while Podman downloads images. The dashboard and API publish only on `127.0.0.1`. The initial stack seeds synthetic demo data; use the dashboard's `Configuración` view to reset or restore it.

Useful daily commands:

```sh
make logs
make status
make down
```

### 5. Optional menu-bar app

Build and open the local SwiftUI app:

```sh
make app-build
open "dist/Duku Net Lab.app"
```

On first launch, select this repository from the menu app so it can run `podman-compose`. The menu app must remain open when using dashboard capture controls because it executes the restricted host-command queue.

## Enable authorized capture

The demo stack works without privileged installation. Real local capture requires these additional steps.

### 1. Confirm authorized radios

Read the BSSID values from the administration panel of the router you are authorized to observe. Add only explicitly confirmed radios to the ignored `.env` file:

```text
DUKU_AUTHORIZED_BSSID=02:11:22:33:44:55,02:11:22:33:44:56
```

Restart the stack after changing `.env`:

```sh
make down
make up
```

The current v1 repository does not yet automate router onboarding. BSSID confirmation is intentionally manual. The `DUKU_WIFI_PSK_*` variables are reserved for future passive WPA2 support and are not consumed by the current worker.

### 2. Install the restricted helper once

```sh
make helper-install
sudo /usr/local/libexec/duku-capture-helper probe
```

On current macOS versions, `probe` reports degraded channel control. Start short radio captures only when the requested channel matches the Mac's current Wi-Fi channel. Use the dashboard's `Capturar tráfico local` fallback to collect real traffic from this Mac when the integrated radio does not expose monitor-mode frames.

### 3. Use manual capture first

Keep the menu app open, open the dashboard and request a short manual capture. Confirm that stop and Wi-Fi restoration behave correctly before relying on scheduled windows.

## Capture helper reference

The one-time installer writes a narrow `/etc/sudoers.d/duku-net-lab` rule and validates it with `visudo`. Review [scripts/install-helper.sh](scripts/install-helper.sh) before executing it. Do not enable schedules until a short manual capability probe succeeds on the current macOS version.

The helper accepts only:

```text
probe
start <en0> <channel> <1..120 minutes> <staging .pcap path>
start-local <en0> <1 minute> <staging .local.pcap path>
stop
status
restore
```

During capture, the helper writes a private `.pcap.partial` staging file that the worker ignores. The file remains mode `600` and is owned by the local user who invoked the restricted helper so the Podman-mounted worker can read it. Packets are truncated to `4096` bytes to bound payload retention. Associated `local-host` captures are additionally capped at 128 MiB per segment. On natural completion, quota stop or safe stop, the helper atomically publishes the `.pcap`. Radio captures are filtered by authorized BSSID before analysis. Associated `local-host` captures are retained separately and analyzed with `partial` coverage.

## Development

### Branch workflow

Use trunk-based development:

- Keep `main` deployable and up to date.
- Create short-lived branches only when starting a task, using `codex/<task-name>`.
- Merge completed work back into `main` after verification and human approval.
- Delete temporary branches after integration.
- Do not create permanent development, staging or release branches in v1.

```sh
make test
make lint
make security-check
make build
make harness
```

The harness emits a supervised prompt for the selected task manifest. It intentionally does not auto-integrate agent output. Create a worktree with:

```sh
scripts/go.sh run ./cmd/harness \
  -task fixtures/tasks/example-dashboard.json -create-worktree
```

Agents must follow [AGENTS.md](AGENTS.md) and use synthetic fixtures only.

## Testing

`make test` is deterministic and does not require `sudo`, live radio access or a running Podman machine. It runs:

- Go unit tests for analyzer, API, worker, helper, Huawei adapter, store and harness logic.
- Dashboard Vitest unit tests plus the production TypeScript/Vite build.
- Swift XCTest coverage for the pure menu-app core.
- Shell syntax checks and the localhost-only binding guard.

The PostgreSQL persistence integration test uses an isolated temporary container and is intentionally separate:

```sh
make test-postgres-integration
```

See [docs/testing.md](docs/testing.md) for the matrix and the limits of host-level simulation.

Before publishing the repository, run:

```sh
make security-check
```

This verifies that `.env`, PCAP, private-key formats, build products and operational directories remain outside the publishable Git set. Demo radios and devices use locally administered synthetic MAC addresses.

## Storage and recovery

- Authorized PCAP: maximum 72 hours and 5 GB.
- Aggregates and redacted findings: maximum 12 months.
- PostgreSQL backups: weekly, eight compressed copies outside the container volume.
- Raw channel-wide PCAP: delete immediately after filtering or on any filtering failure.

Create a backup:

```sh
scripts/backup-postgres.sh
```

## Troubleshooting

If Podman remains in `Currently starting`, inspect it before resetting anything:

```sh
podman machine list
podman system connection list
podman machine stop
podman machine start
```

If the dashboard cannot reach the API, confirm that only loopback ports are published:

```sh
curl -fsS http://127.0.0.1:8080/api/v1/status
```

Operational data, `.env`, PCAP, exports, backups and volumes must remain outside Git and outside AI-agent access. By default, captures live under `~/.duku-net-lab`.
