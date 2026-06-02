# Testing Matrix

## Deterministic suite

Run:

```sh
make test
make lint
make security-check
make schema-check
```

| Area | Coverage |
| --- | --- |
| Analyzer | Authorized-BSSID filter construction, raw-PCAP deletion on success and failure, redaction, Base64 omission, parsing and retention quota |
| Store | Schedule validation, overlap rejection, scheduled commands, lifecycle completion, snapshots, seeded demo, ingest and sorting |
| API | Local resources, malformed requests, host-token boundary, host-command lifecycle, SSE connect, CORS, ingest and CSV/JSON export |
| Worker | Environment parsing, BSSID-list normalization, partial-PCAP exclusion and authenticated report upload |
| Huawei adapter | Private HTTP-only validation and explicit unsupported-firmware status |
| Restricted helper | Interface, channel and path guards, channel parser, private state persistence, invoking-user ownership with mode `600`, stable start PID, process probe and atomic partial-PCAP publication under concurrent finalizers |
| Harness | Protected paths, manifest validation, prompts, concurrency limit and isolated Git worktrees |
| Dashboard | Formatting, explicit coverage labels, loopback API calls, POST behavior, exports and SSE construction |
| SwiftUI core | `.env` token parsing, capture-duration clamping and host-command JSON decoding |
| Shell | Syntax validation, local-secret validation, repository publication safety and automated failure if API or dashboard publishes `0.0.0.0` |

## Local integration suite

Run:

```sh
make test-postgres-integration
```

This starts a temporary PostgreSQL container with Podman, executes the opt-in persistence test and removes the fixture container.

## Manual host verification

The following checks intentionally remain manual because they require macOS privileges, local Wi-Fi state or the installed helper:

- `sudo /usr/local/libexec/duku-capture-helper probe`
- Short authorized capture on the Mac's current Wi-Fi channel
- Safe stop and Wi-Fi restoration
- Podman Desktop startup and the packaged menu-bar app

Use synthetic fixtures for automated tests. Never feed operational PCAP, `.env`, exports, backups or volumes to agents.
