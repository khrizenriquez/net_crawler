# ADR-0006: Keep operational data outside agent access

## Status
Accepted

## Decision
Exclude secrets, real PCAP, exports, backups and volumes from Git and from agent task paths. Use synthetic fixtures and a seeded demo mode for development.

Require `scripts/check-local-secrets.sh` before starting the stack. It rejects placeholder credentials, short values, URL-unsafe values and `.env` permissions broader than `600`.

Run `make security-check` before publication. It verifies that operational directories, build output, sensitive artifact formats, known router identifiers and weak Compose fallbacks remain outside the publishable Git set. Seeded demo radios and devices use locally administered synthetic MAC addresses.

## Consequences
AI-assisted development remains useful without exposing household network data.
