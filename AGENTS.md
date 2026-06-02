# Duku Net Lab Agent Rules

## Mission

Build and maintain a local-first Wi-Fi observability lab without expanding the authorization boundary. Prefer synthetic fixtures and deterministic verification.

## Hard boundaries

- Never read or modify `.env`, `.duku-data`, `data`, `pcap`, `exports`, `backups`, or `volumes`.
- Never inspect real PCAP files. Use synthetic fixtures only.
- Never add active Wi-Fi attacks, forced disconnects, credential storage, or neighbor-network analysis.
- Never persist complete plaintext secrets, reconstructed files, or Base64 payloads.
- Preserve localhost-only publication for API and dashboard.
- Keep the root-owned helper restricted to known actions and validated paths.

## Required flow

Use the harness pipeline:

```text
plan > implement > review > verify > human approval
```

Each task receives a Git branch and worktree. Agents may edit only `allowedPaths` from their task manifest. Integration is always a human decision.

Use trunk-based development. Keep `main` deployable. Create short-lived `codex/<task-name>` branches only for active tasks, merge verified work into `main` after human approval, then delete the temporary branch. Do not introduce permanent development branches.

## Verification

Run:

```sh
make test
make lint
make security-check
make schema-check
make test-postgres-integration
```

The PostgreSQL command is an integration check that requires a local container runtime. All other commands are deterministic and avoid operational data. If a test fails, investigate the root cause before changing code. Document any platform capability that cannot be verified without `sudo` or live radio access.
