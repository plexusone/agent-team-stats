# agent-team-stats Migration: Implementation Plan

## Overview

**Project**: agent-team-stats omniagent-worker migration
**Type**: Migration
**Status**: Planning
**Created**: 2026-07-10

## Prerequisites

- [ ] omniagent-worker MVP complete (Phase 1-3 from omniagent-worker PLAN.md)

## Phases

### Phase 1: Preparation

**Goal**: Prepare the codebase for migration without breaking changes.

**Tasks**:

1. Create new directory structure
   - [ ] Create `workers/` directory
   - [ ] Create `coordinator/` directory
   - [ ] Create `omniskill/` directory

2. Add dependencies
   - [ ] Add `github.com/plexusone/omniagent-worker` to go.mod
   - [ ] Run `go mod tidy`

3. Create baseline tests
   - [ ] Add integration test that captures current CLI output
   - [ ] Document current behavior for regression testing

**Exit Criteria**: New directories exist, dependency added, baseline captured.

### Phase 2: Migrate Research Worker

**Goal**: Migrate the simplest worker first (no LLM required).

**Tasks**:

1. Implement Research worker
   - [ ] Create `workers/research/worker.go`
   - [ ] Implement `Worker` interface
   - [ ] Move search logic from `agents/research/`

2. Test in isolation
   - [ ] Unit tests for Research worker
   - [ ] Verify search functionality unchanged

**Exit Criteria**: Research worker passes all tests.

### Phase 3: Migrate Synthesis Worker

**Goal**: Migrate the LLM-based extraction worker.

**Tasks**:

1. Implement Synthesis worker
   - [ ] Create `workers/synthesis/worker.go`
   - [ ] Implement `Worker` interface with LLM
   - [ ] Move extraction logic from `agents/synthesis/`

2. Test in isolation
   - [ ] Unit tests for Synthesis worker
   - [ ] Verify extraction quality unchanged

**Exit Criteria**: Synthesis worker passes all tests.

### Phase 4: Migrate Verification Worker

**Goal**: Migrate the LLM-based verification worker.

**Tasks**:

1. Implement Verification worker
   - [ ] Create `workers/verification/worker.go`
   - [ ] Implement `Worker` interface with LLM
   - [ ] Move verification logic from `agents/verification/`

2. Test in isolation
   - [ ] Unit tests for Verification worker
   - [ ] Verify verification accuracy unchanged

**Exit Criteria**: Verification worker passes all tests.

### Phase 5: Migrate Coordinator

**Goal**: Replace orchestration with omniagent-worker Coordinator.

**Tasks**:

1. Implement Coordinator
   - [ ] Create `coordinator/coordinator.go`
   - [ ] Create `coordinator/workflow.go` (wrap Eino graph)
   - [ ] Use omniagent-worker.Coordinator with Pool

2. Add AgentOps integration
   - [ ] Configure AgentOps store
   - [ ] Verify traces created

3. Update CLI
   - [ ] Update `main.go` to use new Coordinator
   - [ ] Remove old orchestration code paths

4. Test end-to-end
   - [ ] Integration tests
   - [ ] Regression tests against baseline

**Exit Criteria**: CLI works with new Coordinator, AgentOps traces visible.

### Phase 6: Add OmniSkill Interface

**Goal**: Expose statistics verification as an omniskill.

**Tasks**:

1. Implement skill
   - [ ] Create `omniskill/skill.go`
   - [ ] Create `omniskill/doc.go`
   - [ ] Implement `skill.Skill` interface

2. Test skill
   - [ ] Unit tests for skill
   - [ ] Integration test with mock OmniAgent

3. Document usage
   - [ ] Add example in README
   - [ ] Document configuration options

**Exit Criteria**: omniskill package can be imported and used.

### Phase 7: Cleanup

**Goal**: Remove old code and finalize migration.

**Tasks**:

1. Remove old code
   - [ ] Remove `agents/` directory (keep `agents/direct/` if needed)
   - [ ] Remove `pkg/agent/` directory
   - [ ] Remove `pkg/llm/` directory (if fully replaced)
   - [ ] Remove old orchestration code

2. Update documentation
   - [ ] Update README.md
   - [ ] Update architecture diagrams
   - [ ] Document migration for reference

3. Final testing
   - [ ] Full regression test
   - [ ] Performance comparison
   - [ ] AgentOps trace verification

**Exit Criteria**: Clean codebase, all tests pass, documentation complete.

## Dependencies Between Phases

```
Prerequisites (omniagent-worker MVP)
    │
    ▼
Phase 1 (Preparation)
    │
    ▼
Phase 2 (Research Worker)
    │
    ▼
Phase 3 (Synthesis Worker)
    │
    ▼
Phase 4 (Verification Worker)
    │
    ▼
Phase 5 (Coordinator)
    │
    ▼
Phase 6 (OmniSkill)
    │
    ▼
Phase 7 (Cleanup)
```

## Timeline Estimates

| Phase | Estimated Effort |
|-------|------------------|
| Phase 1 | 0.5 day |
| Phase 2 | 1 day |
| Phase 3 | 1 day |
| Phase 4 | 1 day |
| Phase 5 | 1-2 days |
| Phase 6 | 1 day |
| Phase 7 | 0.5 day |
| **Total** | **6-8 days** |

Note: Requires omniagent-worker MVP (~5-8 days) to be complete first.

## Rollback Plan

If migration fails at any phase:

1. Keep old code in `agents/` until Phase 7
2. CLI can be reverted to use old paths
3. Git history preserves all original code

## Success Metrics

1. All existing tests pass
2. CLI produces identical output (regression test)
3. No performance degradation (< 5% slower)
4. AgentOps traces show complete workflow
5. omniskill can be imported and executed
