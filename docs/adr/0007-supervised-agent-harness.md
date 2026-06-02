# ADR-0007: Supervised Codex CLI harness

## Status
Accepted

## Decision
Use a Go harness with Codex CLI as the first backend. Describe tasks with JSON manifests, create one worktree per task, allow at most three agents, and require human approval before integration.

## Consequences
The repository is agent-ready while preserving review and data boundaries. Additional model backends can implement the same adapter contract later.

