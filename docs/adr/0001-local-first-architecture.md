# ADR-0001: Local-first split architecture

## Status
Accepted

## Decision
Run API, dashboard, worker and PostgreSQL in Podman Compose. Run a minimal SwiftUI menu app and restricted capture helper on macOS.

## Consequences
Operational data stays local. The container stack cannot execute arbitrary host commands. The menu app is required to bridge validated capture requests.

