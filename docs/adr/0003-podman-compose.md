# ADR-0003: Prefer Podman Compose

## Status
Accepted

## Decision
Use Podman Compose as the supported runtime. Keep the Compose file compatible with Docker where practical for verification and troubleshooting.

## Consequences
The menu app must start the Podman machine before starting services. Docker-specific behavior is not a production dependency.

