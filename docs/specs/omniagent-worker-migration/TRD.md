# agent-team-stats Migration: Technical Requirements Document

## Overview

**Project**: agent-team-stats omniagent-worker migration
**Type**: Migration
**Status**: Planning
**Created**: 2026-07-10

## Current Architecture

```
agent-team-stats/
├── agents/
│   ├── orchestration/       # ADK-based orchestrator
│   ├── orchestration-eino/  # Eino-based orchestrator
│   ├── research/            # Web search agent
│   ├── synthesis/           # LLM extraction agent
│   ├── verification/        # LLM verification agent
│   └── direct/              # Direct LLM search
├── pkg/
│   ├── agent/               # BaseAgent for LLM setup
│   ├── config/              # Configuration
│   ├── llm/                 # Multi-LLM factory
│   ├── models/              # Shared data structures
│   ├── orchestration/       # Eino orchestration logic
│   └── search/              # OmniSerp integration
├── main.go                  # CLI entry point
└── cmd/                     # Binary entry points
```

## Target Architecture

```
agent-team-stats/
├── omniskill/               # NEW: omniskill interface
│   ├── skill.go             # StatsVerificationSkill
│   └── doc.go
├── workers/                 # MOVED: from agents/
│   ├── research/
│   │   └── worker.go        # implements omniagent-worker.Worker
│   ├── synthesis/
│   │   └── worker.go        # implements omniagent-worker.Worker
│   └── verification/
│       └── worker.go        # implements omniagent-worker.Worker
├── coordinator/             # MOVED: from pkg/orchestration/
│   ├── coordinator.go       # Uses omniagent-worker.Coordinator
│   └── workflow.go          # Eino workflow definition
├── pkg/
│   ├── models/              # UNCHANGED: Shared data structures
│   └── search/              # UNCHANGED: OmniSerp integration
├── cmd/
│   ├── stats-agent/         # CLI
│   └── stats-server/        # Standalone HTTP server
└── main.go                  # CLI entry point
```

## Migration Mapping

### Workers

| Current | Target | Changes |
|---------|--------|---------|
| `agents/research/main.go` | `workers/research/worker.go` | Implement Worker interface |
| `agents/synthesis/main.go` | `workers/synthesis/worker.go` | Implement Worker interface |
| `agents/verification/main.go` | `workers/verification/worker.go` | Implement Worker interface |

### Coordinator

| Current | Target | Changes |
|---------|--------|---------|
| `agents/orchestration-eino/main.go` | `coordinator/coordinator.go` | Use omniagent-worker.Coordinator |
| `pkg/orchestration/` | `coordinator/workflow.go` | Keep Eino workflow, wrap as WorkflowExecutor |

### Removed

| Current | Reason |
|---------|--------|
| `agents/orchestration/` (ADK) | Consolidate to Eino only |
| `agents/direct/` | Keep for now, but separate concern |
| `pkg/agent/base.go` | Replaced by omniagent-worker.BaseWorker |
| `pkg/llm/factory.go` | Simplified via omniagent-worker config |

## Worker Implementation

### Research Worker

```go
// workers/research/worker.go
package research

import (
    "context"

    worker "github.com/plexusone/omniagent-worker"
    "github.com/plexusone/agent-team-stats/pkg/search"
)

type ResearchWorker struct {
    worker.BaseWorker
    searchClient *search.Client
}

func New(cfg worker.WorkerConfig) *ResearchWorker {
    return &ResearchWorker{
        BaseWorker: worker.NewBaseWorker(cfg),
    }
}

func (w *ResearchWorker) ID() string      { return "research" }
func (w *ResearchWorker) Type() string    { return "research" }
func (w *ResearchWorker) Version() string { return "1.0.0" }

func (w *ResearchWorker) Init(ctx context.Context) error {
    w.searchClient = search.NewClient(/* config */)
    return nil
}

func (w *ResearchWorker) Execute(ctx context.Context, req *worker.Request) (*worker.Response, error) {
    topic := req.Input["topic"].(string)
    maxResults := req.Input["max_results"].(int)

    results, err := w.searchClient.Search(ctx, topic, maxResults)
    if err != nil {
        return nil, err
    }

    return &worker.Response{
        RequestID: req.ID,
        Output: map[string]any{
            "results": results,
        },
    }, nil
}

func (w *ResearchWorker) Shutdown(ctx context.Context) error {
    return nil
}

func (w *ResearchWorker) Health(ctx context.Context) worker.HealthStatus {
    return worker.HealthStatus{Status: "healthy"}
}
```

### Synthesis Worker

```go
// workers/synthesis/worker.go
package synthesis

import (
    "context"

    worker "github.com/plexusone/omniagent-worker"
    "github.com/plexusone/omnillm"
)

type SynthesisWorker struct {
    worker.BaseWorker
    llmClient *omnillm.ChatClient
}

func New(cfg worker.WorkerConfig) *SynthesisWorker {
    return &SynthesisWorker{
        BaseWorker: worker.NewBaseWorker(cfg),
    }
}

func (w *SynthesisWorker) ID() string      { return "synthesis" }
func (w *SynthesisWorker) Type() string    { return "synthesis" }
func (w *SynthesisWorker) Version() string { return "1.0.0" }

func (w *SynthesisWorker) Init(ctx context.Context) error {
    // LLM client initialized from config via BaseWorker
    w.llmClient = w.LLMClient()
    return nil
}

func (w *SynthesisWorker) Execute(ctx context.Context, req *worker.Request) (*worker.Response, error) {
    urls := req.Input["urls"].([]string)

    // Fetch and extract statistics from URLs
    statistics, err := w.synthesize(ctx, urls)
    if err != nil {
        return nil, err
    }

    return &worker.Response{
        RequestID: req.ID,
        Output: map[string]any{
            "statistics": statistics,
        },
    }, nil
}
```

### Verification Worker

```go
// workers/verification/worker.go
package verification

// Similar pattern to Synthesis worker
// Uses LLM to verify statistics against source URLs
```

## Coordinator Implementation

```go
// coordinator/coordinator.go
package coordinator

import (
    "context"

    worker "github.com/plexusone/omniagent-worker"
    "github.com/plexusone/omniagent-worker/eino"
    "github.com/plexusone/agent-team-stats/workers/research"
    "github.com/plexusone/agent-team-stats/workers/synthesis"
    "github.com/plexusone/agent-team-stats/workers/verification"
)

func New(cfg Config) *worker.Coordinator {
    coord := worker.NewCoordinator(worker.CoordinatorConfig{
        ID:       "stats-coordinator",
        Workflow: NewWorkflow(),
        AgentOps: cfg.AgentOps,
    })

    // Register in-process workers
    coord.Pool().Register(research.New(cfg.Research))
    coord.Pool().Register(synthesis.New(cfg.Synthesis))
    coord.Pool().Register(verification.New(cfg.Verification))

    return coord
}
```

### Eino Workflow

```go
// coordinator/workflow.go
package coordinator

import (
    worker "github.com/plexusone/omniagent-worker"
    "github.com/plexusone/omniagent-worker/eino"
)

func NewWorkflow() worker.WorkflowExecutor {
    // Wrap existing Eino graph
    return eino.NewStatefulWorkflow(buildGraph())
}

func buildGraph() *compose.Graph[Input, Output] {
    // Existing Eino workflow logic from pkg/orchestration/
    // Research → Synthesis → Verification → QualityCheck → (Retry?) → Format
}
```

## OmniSkill Implementation

```go
// omniskill/skill.go
package omniskill

import (
    "context"

    "github.com/plexusone/omniskill/skill"
    "github.com/plexusone/agent-team-stats/coordinator"
)

type StatsVerificationSkill struct {
    skill.BaseSkill
    coord *worker.Coordinator
}

func New(cfg Config) *StatsVerificationSkill {
    return &StatsVerificationSkill{
        BaseSkill: skill.BaseSkill{
            SkillName:        "stats-verification",
            SkillDescription: "Research and verify statistics from authoritative sources",
        },
    }
}

func (s *StatsVerificationSkill) Init(ctx context.Context) error {
    s.coord = coordinator.New(coordinator.Config{
        // Configuration
    })
    return s.coord.Pool().Init(ctx)
}

func (s *StatsVerificationSkill) Tools() []skill.Tool {
    return []skill.Tool{
        s.verifyStatisticsTool(),
    }
}

func (s *StatsVerificationSkill) verifyStatisticsTool() skill.Tool {
    return skill.NewTool(
        "verify_statistics",
        "Research and verify statistics on a topic",
        s.verifyStatistics,
        skill.WithParameter("topic", skill.Parameter{
            Type:        "string",
            Description: "The topic to research statistics for",
            Required:    true,
        }),
        skill.WithParameter("min_verified", skill.Parameter{
            Type:        "integer",
            Description: "Minimum number of verified statistics",
            Default:     10,
        }),
    )
}

func (s *StatsVerificationSkill) verifyStatistics(ctx context.Context, params map[string]any) (any, error) {
    resp, err := s.coord.Execute(ctx, &worker.CoordinatorRequest{
        WorkflowID: uuid.NewString(),
        Input: map[string]any{
            "topic":        params["topic"],
            "min_verified": params["min_verified"],
        },
    })
    if err != nil {
        return nil, err
    }
    return resp.Output, nil
}

func (s *StatsVerificationSkill) Close() error {
    return s.coord.Pool().Shutdown(context.Background())
}
```

## CLI Changes

```go
// main.go - minimal changes

func main() {
    // Create coordinator with in-process workers
    coord := coordinator.New(cfg)

    // CLI uses coordinator directly
    resp, err := coord.Execute(ctx, req)

    // Output results
}
```

## Configuration

### Environment Variables (Unchanged)

```bash
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=...
SEARCH_PROVIDER=serper
SERPER_API_KEY=...
AGENTOPS_ENABLED=true
AGENTOPS_DSN=postgres://...
```

### Programmatic

```go
cfg := coordinator.Config{
    AgentOps: &worker.AgentOpsConfig{
        Enabled: true,
        Store:   agentOpsStore,
    },
    Research: worker.WorkerConfig{
        ID:   "research",
        Type: "research",
    },
    Synthesis: worker.WorkerConfig{
        ID:   "synthesis",
        Type: "synthesis",
        LLM: &worker.LLMConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-20250514",
        },
    },
    Verification: worker.WorkerConfig{
        ID:   "verification",
        Type: "verification",
        LLM: &worker.LLMConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-20250514",
        },
    },
}
```

## Testing Strategy

### Unit Tests

- Test each worker independently
- Mock external dependencies (LLM, search)
- Verify Worker interface compliance

### Integration Tests

- Test coordinator with real workers
- Verify AgentOps traces created
- End-to-end CLI test

### Regression Tests

- Compare output with pre-migration baseline
- Verify accuracy not degraded

## Backward Compatibility

| Feature | Compatibility |
|---------|---------------|
| CLI interface | Unchanged |
| HTTP endpoints | Unchanged |
| Configuration | Unchanged |
| Output format | Unchanged |

## References

- [omniagent-worker TRD](https://github.com/plexusone/omniagent-worker/docs/specs/origin/TRD.md)
- [omniskill](https://github.com/plexusone/omniskill)
- [Current agent-team-stats](https://github.com/plexusone/agent-team-stats)
