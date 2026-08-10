# Eino Orchestration Agent

A deterministic orchestration agent built with [Eino framework](https://github.com/cloudwego/eino) that provides more predictable and reliable workflow execution for finding verified statistics.

## Overview

This project now includes **two orchestration agents** with different characteristics:

### 1. ADK Orchestration (Port 8000)
- **Framework**: [Google ADK (Agent Development Kit)](https://github.com/google/adk-go)
- **Approach**: LLM-based decision making (Gemini 2.0 Flash)
- **Characteristics**:
  - Flexible, adaptive behavior
  - Uses LLM for orchestration decisions
  - More dynamic but less predictable
  - Ideal for complex decision-making workflows

### 2. Eino Orchestration (Port 8003) ⭐ RECOMMENDED
- **Framework**: [Eino](https://github.com/cloudwego/eino)
- **Approach**: Deterministic graph-based workflow
- **Characteristics**:
  - Deterministic, predictable behavior
  - Type-safe graph orchestration
  - Compile-time validation
  - More reliable and faster
  - **Recommended for production use**

## Why Eino for Orchestration?

### Deterministic Workflow
The Eino orchestrator uses a **directed graph** with explicit nodes and edges, ensuring:
- Same input → Same workflow execution path
- No LLM decision-making for orchestration logic
- Predictable resource usage and timing

### Type Safety
Eino provides compile-time type checking:
```go
Graph[*models.OrchestrationRequest, *models.OrchestrationResponse]
```
This ensures all nodes have compatible input/output types.

### Performance
- **Faster**: No LLM calls for orchestration decisions
- **Lower cost**: Only uses LLMs in research/verification agents
- **More reliable**: Graph compilation validates workflow before execution

## Two Composed Loops, Not One Linear Graph

The Eino orchestrator implements the workflow as two bounded loops — a REAL
loop for discovery and a VEAL loop for verification — rather than a single
linear pipeline. **See [REAL and VEAL
Loops](real-veal-loops.md) for the full explanation of why and how they
compose.** Summary:

```
Orchestrate()
  REAL loop (up to 5 rounds, mission: reach MinVerifiedStats):
    Read     — verified count so far, domains VEAL has rejected
    Evaluate — shortfall = target - verified; done if <= 0
    Act      — one discovery round via the Eino graph below, then hand
               the batch to the VEAL loop
    Loop     — repeat until mission complete or rounds exhausted

    ┌─ discovery graph (Eino, compiled once, invoked per round) ─┐
    │  START → [Research] → [Synthesis] → END                    │
    └───────────────────────────────────────────────────────────┘

    VEAL loop (up to 3 attempts per batch, state: verified-or-rejected):
      Validate — verification agent + local aggregator-source check
      Evaluate — GO, or NO-GO with a specific reason
      Act      — targeted fix for fixable reasons only; reject otherwise
      Loop     — re-validate until converged or attempts exhausted

  Format Response — Partial: true if still short after REAL's rounds
```

Only `Research → Synthesis` is an Eino graph (`buildDiscoveryGraph`) — the
loop-and-retry logic itself is plain bounded Go, since the fixes it applies
are branchy per-candidate decisions that don't map cleanly onto a linear
graph. Eino's type-safe steps handle what's genuinely linear within a round;
the loops decide how many rounds to run.

## Usage

### Running the Eino Orchestrator

#### Option 1: Run with Eino orchestrator
```bash
make run-all-eino
```

This starts:
- Research Agent (8001/9001)
- Verification Agent (8002/9002)
- **Eino Orchestration Agent (8003/9003)** ⭐

#### Option 2: Run Eino orchestrator separately
```bash
# Terminal 1: Research Agent
make run-research

# Terminal 2: Verification Agent
make run-verification

# Terminal 3: Eino Orchestrator
make run-orchestration-eino
```

### API Calls

#### HTTP API
```bash
curl -X POST http://localhost:8003/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "climate change",
    "min_verified_stats": 10,
    "max_candidates": 30,
    "reputable_only": true
  }'
```

#### A2A Protocol (Port 9003)
The Eino orchestrator also supports A2A protocol for agent-to-agent communication.

## Comparison: trpc-agent vs Eino

| Feature | trpc-agent Orchestrator | Eino Orchestrator |
|---------|------------------------|-------------------|
| **Port** | 8000 (HTTP), 9000 (A2A) | 8003 (HTTP), 9003 (A2A) |
| **Decision Making** | LLM-based | Deterministic |
| **Predictability** | Variable | Consistent |
| **Performance** | Slower (LLM calls) | Faster (no LLM) |
| **Cost** | Higher (LLM tokens) | Lower (no LLM) |
| **Flexibility** | High | Moderate |
| **Type Safety** | Runtime | Compile-time |
| **Workflow** | Dynamic | Static graph |
| **Best For** | Complex adaptive tasks | Predictable workflows |

## When to Use Which?

### Use Eino Orchestrator When:
- ✅ You need **deterministic, reproducible** results
- ✅ You want **faster response times**
- ✅ You need **lower costs** (no LLM for orchestration)
- ✅ Your workflow is **well-defined and stable**
- ✅ You want **compile-time type safety**

### Use ADK Orchestrator When:
- ✅ You need **adaptive decision making**
- ✅ Workflow logic **changes based on content**
- ✅ You want **LLM reasoning** for orchestration (Gemini 2.0 Flash)
- ✅ Requirements are **less well-defined**
- ✅ You need **complex decision trees** in orchestration

## Eino Graph Implementation

### Key Components

#### 1. Lambda Nodes
Each step is implemented as an `InvokableLambda`:
```go
validateInputLambda := compose.InvokableLambda(
    func(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationRequest, error) {
        // Validation logic
        return req, nil
    }
)
g.AddLambdaNode("validate_input", validateInputLambda)
```

#### 2. Type-Safe State
State is passed through typed structs within a discovery round:
- `discoveryInput` (request + domains already rejected by VEAL) → Input
- `ResearchState` → After research
- `SynthesisState` → Output of the discovery graph, handed to the VEAL loop

`OrchestrationResponse` is built directly by `Orchestrate()` after the REAL
loop completes — see [REAL and VEAL Loops](real-veal-loops.md) for how the
loop and the graph fit together.

#### 3. Graph Edges
Edges define the per-round workflow sequence:
```go
g.AddEdge(compose.START, "research")
g.AddEdge("research", "synthesis")
g.AddEdge("synthesis", compose.END)
```

#### 4. Graph Compilation
The discovery graph is compiled once per `Orchestrate()` call, then invoked
once per REAL round:
```go
compiledDiscovery, err := discoveryGraph.Compile(ctx)
synthState, err := compiledDiscovery.Invoke(ctx, &discoveryInput{Request: req, ExcludedDomains: excludedDomains})
```

## Configuration

### Environment Variables
```bash
ORCHESTRATOR_EINO_URL=http://localhost:8003
```

Add to your `.env` file to configure the Eino orchestrator URL.

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────┐
│                    USER REQUEST                          │
└──────────────────┬───────────────────────────────────────┘
                   │
                   │ Choose orchestrator:
                   ├─────────────────┬─────────────────────┐
                   │                 │                     │
                   ▼                 ▼                     ▼
         ┌─────────────────┐  ┌──────────────────┐  ┌────────────┐
         │   ORCHESTRATOR  │  │  ORCHESTRATOR    │  │   Direct   │
         │   (trpc-agent)  │  │    (Eino)        │  │   Call     │
         │   Port 8000     │  │   Port 8003      │  │            │
         │                 │  │                  │  │            │
         │  LLM-based      │  │  Deterministic   │  │            │
         │  decisions      │  │  graph workflow  │  │            │
         └────────┬────────┘  └────────┬─────────┘  └──────┬─────┘
                  │                    │                   │
                  └────────────┬───────┴───────────────────┘
                               │
                ┌──────────────┴──────────────┐
                │                             │
                ▼                             ▼
    ┌──────────────────────┐     ┌──────────────────────┐
    │  RESEARCH AGENT      │     │ VERIFICATION AGENT   │
    │  Port 8001           │     │ Port 8002            │
    └──────────────────────┘     └──────────────────────┘
```

## Benefits of Dual Orchestrators

1. **Flexibility**: Choose the right tool for your use case
2. **Comparison**: A/B test different orchestration approaches
3. **Migration**: Gradually move from LLM to deterministic workflows
4. **Learning**: Compare results between approaches

## Technology Stack

- **Eino**: CloudWeGo's LLM application framework
- **Graph Orchestration**: Directed graph with typed nodes
- **A2A Protocol**: trpc-a2a-go for agent communication
- **Type Safety**: Compile-time validation

## Logging

The Eino orchestrator logs each loop's iterations, so you can see the REAL
and VEAL loops converge (or exhaust their attempts) in real time:
```
REAL loop: starting topic=climate change target=10 max_attempts=5
REAL loop: round attempt=1 max_attempts=5 shortfall=10 excluded_domains=0
REAL/discovery: research topic=climate change excluded_domains=0
REAL/discovery: research completed sources=18 filtered_out=0
REAL/discovery: synthesis sources=18
REAL/discovery: synthesis completed candidates=15
VEAL loop: validate attempt=1 max_attempts=3 candidates=15
VEAL loop: act — searching for a primary-source replacement name="X users" rejected_domain=stats-aggregator.example
VEAL loop: rejecting candidate name="Y metric" reason=unknown attempt=1
VEAL loop: validate attempt=2 max_attempts=3 candidates=1
REAL loop: round complete attempt=1 round_candidates=15 round_verified=12 total_verified=12
REAL loop: target met verified=12 attempts_used=0
```

## Next Steps

1. **Install Eino**:
   ```bash
   go get github.com/cloudwego/eino
   ```

2. **Build**:
   ```bash
   make build
   ```

3. **Run with Eino**:
   ```bash
   make run-all-eino
   ```

4. **Test**:
   ```bash
   curl -X POST http://localhost:8003/orchestrate -H "Content-Type: application/json" -d '{"topic": "AI statistics", "min_verified_stats": 5}'
   ```

## Contributing

Contributions to improve the Eino orchestration workflow are welcome!
