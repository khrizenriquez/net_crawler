# ADR-0008: V2 capture-source roadmap

## Status
Accepted

## Decision
Treat v1 radio capture and its explicit `partial/local-host` fallback as replaceable source adapters. Plan v2 adapters for authorized port mirroring, gateway capture, multiple Wi-Fi interfaces and Linux hosts. macOS `26.4.1` removed the historical `airport` channel-control utility, and the integrated radio may expose zero monitor-mode frames, so a compatible external adapter backend is also required for reliable radio-wide capture and automatic multi-channel rotation.

## Consequences
V1 documents partial coverage honestly. Ethernet visibility and simultaneous-channel coverage require new capabilities, not a dashboard-only change.
