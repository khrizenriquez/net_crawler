# ADR-0005: Restricted privileged capture helper

## Status
Accepted

## Decision
Install one root-owned helper with a narrow sudoers rule. Permit only `probe`, `start`, `stop`, `status`, and `restore`. Restrict capture to `en0`, validated channels, a two-hour maximum, and staging paths inside the application directory.

## Consequences
The dashboard never receives arbitrary root execution. Installation requires one reviewed `sudo` step.

