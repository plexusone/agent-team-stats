# Architecture Overview

The system implements a **worker-based architecture** using [omniagent-worker](https://github.com/plexusone/omniagent-worker) with clear separation of concerns.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   User Request                          │
│              "Find climate change statistics"           │
└───────────────────┬─────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────────────┐
│               COORDINATOR                               │
│         (omniagent-worker.Coordinator)                  │
│  • Coordinates worker workflow                          │
│  • Manages retry logic                                  │
│  • AgentOps tracing via OpenTelemetry                   │
└─────┬─────────────┬────────────────┬────────────────────┘
      │             │                │
      ▼             ▼                ▼
┌────────────┐ ┌──────────┐ ┌─────────────────┐
│  RESEARCH  │ │SYNTHESIS │ │  VERIFICATION   │
│   WORKER   │ │  WORKER  │ │     WORKER      │
│            │ │          │ │                 │
│ • Search   │─│• Fetch   │─│• Re-fetch URLs  │
│   Serper   │ │  URLs    │ │• Validate text  │
│ • Filter   │ │• LLM     │ │• Check numbers  │
│   Sources  │ │  Extract │ │• Flag errors    │
└────────────┘ └──────────┘ └─────────────────┘
      │             │                │
      ▼             ▼                ▼
  URLs only     Statistics     Verified Stats
```

## Worker Architecture (omniagent-worker)

The system uses [omniagent-worker](https://github.com/plexusone/omniagent-worker) for worker lifecycle, coordination, and observability.

### 1. Research Worker (`workers/research/`) - Web Search Only

- **No LLM required** - Pure search functionality
- Implements `omniagent-worker.Worker` interface
- Web search via Serper/SerpAPI integration
- Returns URLs with metadata (title, snippet, domain)
- Prioritizes reputable sources (`.gov`, `.edu`, research orgs)
- Output: List of `SearchResult` objects

### 2. Synthesis Worker (`workers/synthesis/`) - LLM Extraction

- **LLM-heavy** extraction worker
- Implements `omniagent-worker.Worker` interface
- Fetches webpage content from URLs
- Extracts numerical statistics using LLM analysis
- Finds verbatim excerpts containing statistics
- Creates `CandidateStatistic` objects with proper metadata

### 3. Verification Worker (`workers/verification/`) - LLM Validation

- **LLM-light** validation worker
- Implements `omniagent-worker.Worker` interface
- Re-fetches source URLs to verify content
- Checks excerpts exist verbatim in source
- Validates numerical values match exactly
- Flags hallucinations and discrepancies
- Returns verification results with pass/fail reasons

### 4. Coordinator (`coordinator/`) - Workflow Orchestration

- Built with `omniagent-worker.Coordinator`
- **Deterministic graph-based workflow** (no LLM for orchestration)
- Coordinates: Research → Synthesis → Verification
- Implements adaptive retry logic
- AgentOps tracing via OpenTelemetry
- Workflow: ValidateInput → Research → Synthesis → Verification → QualityCheck → Format

## Reputable Sources

The research agent prioritizes these source types:

- **Government Agencies**: CDC, NIH, Census Bureau, EPA, etc.
- **Academic Institutions**: Universities, research journals
- **Research Organizations**: Pew Research, Gallup, McKinsey, etc.
- **International Organizations**: WHO, UN, World Bank, IMF, etc.
- **Respected Media**: With proper citations (NYT, WSJ, Economist, etc.)

## Error Handling

- **Source Unreachable**: Marked as failed with reason
- **Excerpt Not Found**: Verification fails with explanation
- **Value Mismatch**: Flagged as discrepancy
- **Insufficient Results**: Automatic retry with more candidates
- **Max Retries Exceeded**: Returns partial results with warning

## Learn More

- [4-Agent Architecture](4-agent-architecture.md) - Detailed agent implementation
- [Eino Orchestration](eino-orchestration.md) - Deterministic graph-based orchestration
