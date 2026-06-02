# ADR-0002: Use local tshark for packet dissection

## Status
Accepted

## Decision
Package `tshark` inside the local worker container. Use it for BSSID filtering and protocol dissection rather than implementing packet parsers from scratch.

## Consequences
The project inherits a mature protocol engine without sending captures off-device. Worker images are larger and must be updated regularly.

