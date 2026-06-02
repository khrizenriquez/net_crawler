# ADR-0004: PostgreSQL and bounded retention

## Status
Accepted

## Decision
Persist compact aggregates and redacted findings in PostgreSQL for 12 months. Retain authorized PCAP for at most 72 hours and 5 GB. Keep eight weekly compressed database backups.

## Consequences
Long-term trends remain queryable without retaining an indefinite packet archive.

