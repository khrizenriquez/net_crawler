# ADR-0008: V2 capture-source roadmap

## Status
Accepted

## Decision
Treat v1 radio capture as a replaceable source adapter. Plan v2 adapters for authorized port mirroring, gateway capture, multiple Wi-Fi interfaces and Linux hosts. macOS `26.4.1` removed the historical `airport` channel-control utility, so a compatible external adapter backend is also required for automatic multi-channel rotation.

## Consequences
V1 documents partial coverage honestly. Ethernet visibility and simultaneous-channel coverage require new capabilities, not a dashboard-only change.
