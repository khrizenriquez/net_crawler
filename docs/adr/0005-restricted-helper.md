# ADR-0005: Restricted privileged capture helper

## Status
Accepted

## Decision
Install one root-owned helper with a narrow sudoers rule. Permit only `probe`, `start`, `stop`, `status`, and `restore`. Restrict capture to `en0`, validated channels, a two-hour maximum, and staging paths inside the application directory.

Write active channel-wide captures as `.pcap.partial`. Keep the file mode at `600` and transfer ownership to the local user who invoked the restricted helper so the Podman-mounted worker can read it without widening filesystem permissions. An internal root-owned supervisor publishes the completed `.pcap` with an atomic rename after `tcpdump` exits naturally or after a safe `stop`. The worker ignores partial files and accepts only published `.pcap` staging artifacts.

## Consequences
The dashboard never receives arbitrary root execution. Installation requires one reviewed `sudo` step.

The internal supervisor is not added to the sudoers allowlist. A partially written raw capture cannot cross the worker analysis boundary.
