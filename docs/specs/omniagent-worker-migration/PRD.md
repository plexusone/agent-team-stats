# agent-team-stats Migration: Product Requirements Document

## Overview

**Project**: agent-team-stats omniagent-worker migration
**Type**: Migration
**Status**: Planning
**Created**: 2026-07-10

## Problem Statement

agent-team-stats currently implements custom agent patterns that should be:

1. Extracted into reusable omniagent-worker package
2. Enhanced with AgentOps observability
3. Exposed as an omniskill for OmniAgent integration
4. Maintained as the canonical reference implementation

Current state:

- 4 agents (Orchestrator, Research, Synthesis, Verification) with custom implementations
- `pkg/agent/base.go` provides shared LLM setup
- Eino and ADK orchestration options exist
- No AgentOps tracing for agent lifecycle
- Cannot be embedded as a skill

## Goals

### Primary Goals

1. **Migrate to omniagent-worker** - Use Worker/Coordinator from omniagent-worker
2. **Add full AgentOps observability** - Workflow, task, handoff tracing
3. **Add omniskill interface** - Expose as embeddable skill
4. **Maintain backward compatibility** - CLI and standalone deployment still work
5. **Establish as reference implementation** - Document patterns for future teams

### Non-Goals

1. Not changing the core statistics logic
2. Not adding new agents or capabilities (yet)
3. Not migrating to Multi-Agent Spec (that's a separate future project)

## Users

### Primary User

- PlexusOne internal development (ourselves)
- CLI users of stats-agent

### Secondary Users

- OmniAgent users who want statistics verification as a skill
- Developers using this as a reference for building agent teams

## Requirements

### Functional Requirements

| ID | Requirement | Priority |
|----|-------------|----------|
| F1 | Migrate Research worker to omniagent-worker.Worker | Must |
| F2 | Migrate Synthesis worker to omniagent-worker.Worker | Must |
| F3 | Migrate Verification worker to omniagent-worker.Worker | Must |
| F4 | Migrate Coordinator to omniagent-worker.Coordinator | Must |
| F5 | Add omniskill/ with StatsVerificationSkill | Must |
| F6 | Maintain CLI functionality | Must |
| F7 | Maintain standalone HTTP deployment | Must |
| F8 | Add AgentOps traces for all operations | Must |

### Non-Functional Requirements

| ID | Requirement | Priority |
|----|-------------|----------|
| N1 | No regression in verification accuracy | Must |
| N2 | No significant performance degradation | Must |
| N3 | Clear separation between domain and infrastructure | Must |
| N4 | Document migration patterns | Should |

## Success Criteria

1. All existing tests pass
2. CLI produces same results as before
3. AgentOps traces show complete workflow visibility
4. omniskill can be imported and used in OmniAgent
5. Standalone deployment still works

## Dependencies

### Upstream Dependencies

- `github.com/plexusone/omniagent-worker` - New dependency (MVP must be complete)
- `github.com/plexusone/omniskill` - For skill interface

### Downstream Dependents

- OmniAgent (as a skill provider)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| omniagent-worker MVP delays | Medium | High | Can proceed with spec docs while waiting |
| Breaking changes during migration | Medium | Medium | Incremental migration, one worker at a time |
| Performance regression | Low | Medium | Benchmark before/after |

## Timeline

See PLAN.md for detailed phases.

## References

- [IDEATION_CHAT.md](../../../IDEATION_CHAT.md) - Original discussion
- [omniagent-worker specs](https://github.com/plexusone/omniagent-worker/docs/specs/origin/)
- [omniskill](https://github.com/plexusone/omniskill) - Skill interface
