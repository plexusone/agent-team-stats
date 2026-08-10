# agent-team-stats: Master Roadmap

This file tracks all projects and their roadmaps for agent-team-stats.

## Active Projects

| Project | Status | Description |
|---------|--------|-------------|
| [verified-statistics](verified-statistics/ROADMAP.md) | In progress | Evidence-gated claim verification (`INIT-AGENTTEAMSTATS-001`) |
| [omniagent-worker-migration](omniagent-worker-migration/ROADMAP.md) | Blocked | Migrate to omniagent-worker |

## Project Index

### verified-statistics

**Status**: In progress (`INIT-AGENTTEAMSTATS-001`)
**Goal**: Raise the trustworthiness of statistics the pipeline marks "verified" — tighten the evidence bar in structured-evaluation (quote-with-value, source role, corroboration, staleness) and harden agent-team-stats' evidence gathering. Phase 1 targets structured-evaluation (v0.13.0); Phase 2 targets agent-team-stats.

See [verified-statistics/ROADMAP.md](verified-statistics/ROADMAP.md) for phases and RMIs. RMI IDs are tracked in VisionStudio (`INIT-AGENTTEAMSTATS-001`); commits carry `Refs: RMI-AGENTTEAMSTATS-NNN`.

### omniagent-worker-migration

**Status**: Blocked (waiting on omniagent-worker MVP)
**Goal**: Migrate agent-team-stats to use omniagent-worker and add omniskill interface.

See [omniagent-worker-migration/ROADMAP.md](omniagent-worker-migration/ROADMAP.md) for detailed work items.

## Completed Projects

(None yet)
