# agent-team-stats Migration: Roadmap

## Overview

**Project**: agent-team-stats omniagent-worker migration
**Type**: Migration
**Status**: Planning
**Created**: 2026-07-10

## Work Items (In Order)

### Milestone 1: Migration (Current Focus)

**Blocked by**: omniagent-worker MVP

- [ ] **1.1** Create new directory structure (workers/, coordinator/, omniskill/)
- [ ] **1.2** Add omniagent-worker dependency
- [ ] **1.3** Create baseline regression tests
- [ ] **1.4** Migrate Research worker
- [ ] **1.5** Migrate Synthesis worker
- [ ] **1.6** Migrate Verification worker
- [ ] **1.7** Migrate Coordinator with Eino workflow
- [ ] **1.8** Add AgentOps integration
- [ ] **1.9** Update CLI to use new Coordinator
- [ ] **1.10** Create omniskill/skill.go
- [ ] **1.11** Remove old agents/ code
- [ ] **1.12** Update documentation

### Milestone 2: Post-Migration Enhancements

- [ ] **2.1** Add Prometheus metrics
- [ ] **2.2** Add structured logging improvements
- [ ] **2.3** Add configuration validation
- [ ] **2.4** Add worker health dashboard
- [ ] **2.5** Performance optimization based on AgentOps insights

### Milestone 3: Channel Integration

- [ ] **3.1** Add Slack channel support (via OmniAgent)
- [ ] **3.2** Add Discord channel support
- [ ] **3.3** Add webhook trigger support
- [ ] **3.4** Add scheduled research jobs

### Milestone 4: Multi-Agent Spec Comparison

**Goal**: Create second team using Multi-Agent Spec to compare with bespoke implementation.

- [ ] **4.1** Define agent-team-stats in Multi-Agent Spec format
- [ ] **4.2** Create agent-team-stats-spec/ project
- [ ] **4.3** Generate workers from spec
- [ ] **4.4** Compare performance and accuracy
- [ ] **4.5** Document differences and learnings
- [ ] **4.6** Feed improvements back to Multi-Agent Spec

### Milestone 5: Advanced Features

- [ ] **5.1** Add streaming responses for real-time progress
- [ ] **5.2** Add caching for repeated searches
- [ ] **5.3** Add source credibility scoring
- [ ] **5.4** Add multi-language support
- [ ] **5.5** Add PDF/document source support

## Reference Architecture

### Current (Pre-Migration)

```
CLI
 │
 ▼
Orchestration Agent (Eino)
 │
 ├─▶ Research Agent (HTTP)
 ├─▶ Synthesis Agent (HTTP)
 └─▶ Verification Agent (HTTP)
```

### Target (Post-Migration)

```
CLI ◀──────────────────────────────────────┐
 │                                         │
 ▼                                         │
Coordinator (omniagent-worker)             │
 │                                         │
 └─▶ Pool (in-process)                     │
      ├─▶ Research Worker                  │
      ├─▶ Synthesis Worker                 │
      └─▶ Verification Worker              │
                                           │
OmniAgent ─▶ StatsVerificationSkill ───────┘
              (omniskill/)
```

### Future (With Channels)

```
User
 │
 ├─▶ CLI
 ├─▶ Slack
 ├─▶ Discord
 └─▶ Webhook
      │
      ▼
   OmniAgent
      │
      ▼
   StatsVerificationSkill
      │
      ▼
   Coordinator
      │
      └─▶ Workers
```

## Comparison: Bespoke vs Spec-Generated

After Milestone 4, we will maintain two implementations:

| Aspect | agent-team-stats (bespoke) | agent-team-stats-spec (generated) |
|--------|---------------------------|-----------------------------------|
| Definition | Go code | Multi-Agent Spec YAML/JSON |
| Workers | Hand-crafted | Generated from spec |
| Workflow | Eino graph in code | Rendered from spec |
| Optimization | Manual | Spec-driven |
| Use case | Reference/benchmark | Spec validation |

This comparison drives improvements in both directions:

- Spec learns from bespoke best practices
- Bespoke validates spec is sufficient

## Version History

| Version | Date | Changes |
|---------|------|---------|
| Current | - | Pre-migration (custom agents) |
| 2.0.0 | TBD | Post-migration (omniagent-worker) |
| 2.1.0 | TBD | Channel integration |
| 3.0.0 | TBD | Multi-Agent Spec comparison |

## References

- [PRD.md](PRD.md) - Product requirements
- [TRD.md](TRD.md) - Technical requirements
- [PLAN.md](PLAN.md) - Implementation plan
- [omniagent-worker ROADMAP](https://github.com/plexusone/omniagent-worker/docs/specs/origin/ROADMAP.md)
