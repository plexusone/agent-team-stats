# Worker Architecture Implementation

## Overview

The Statistics Agent Team uses a **worker-based architecture** built on [omniagent-worker](https://github.com/plexusone/omniagent-worker) with clear separation of concerns:

```
┌─────────────────────────────────────────────────────┐
│              Coordinator                            │
│       (omniagent-worker.Coordinator)                │
└────────┬──────────────┬──────────────┬──────────────┘
         │              │              │
         ▼              ▼              ▼
┌────────────────┐ ┌──────────────┐ ┌────────────────┐
│   Research     │ │  Synthesis   │ │ Verification   │
│    Worker      │ │    Worker    │ │    Worker      │
│                │ │              │ │                │
│ - Web Search   │ │ - Fetch URLs │ │ - Verify URLs  │
│ - Find Sources │ │ - LLM Extract│ │ - Check Facts  │
│ - Filter       │ │ - Parse Stats│ │ - Validate     │
│   Reputable    │ │              │ │   Excerpts     │
└────────────────┘ └──────────────┘ └────────────────┘
        │                  │                  │
        ▼                  ▼                  ▼
   Serper/SerpAPI    Webpage Content    Source Validation
```

## Worker Responsibilities

### 1. Research Worker
**Role**: Source Discovery
**Technology**: Web Search (Serper/SerpAPI)
**No LLM Required**

**Tasks**:

- Perform web searches via `pkg/search` service
- Return URLs with metadata (title, snippet, domain)
- Filter for reputable sources (.gov, .edu, research orgs)
- **Output**: List of SearchResult objects

**Files**:

- `workers/research/worker.go` - Implements omniagent-worker.Worker interface
- `pkg/search/service.go` - OmniSerp integration

**API**:
```json
POST /research
{
  "topic": "climate change",
  "max_statistics": 20,
  "reputable_only": true
}

Response:
{
  "topic": "climate change",
  "candidates": [/* URLs as placeholder candidates */],
  "timestamp": "2025-12-13T10:30:00Z"
}
```

### 2. Synthesis Worker
**Role**: Statistics Extraction
**Technology**: LLM (Gemini/Claude/OpenAI/Ollama)
**LLM-Heavy**

**Tasks**:

- Fetch webpage content from URLs
- Use LLM to intelligently analyze text and extract statistics
- Extract numerical values, units, and context using structured prompts
- Find verbatim excerpts containing statistics
- Create candidate statistics with proper metadata
- **Output**: List of CandidateStatistic objects

**Files**:

- `workers/synthesis/worker.go` - Implements omniagent-worker.Worker interface
- Uses OmniLLM for multi-provider LLM support
- Full LLM integration with JSON output parsing

**API**:
```json
POST /synthesize
{
  "topic": "climate change",
  "search_results": [
    {
      "url": "https://www.iea.org/...",
      "title": "Renewable Energy Report",
      "snippet": "...",
      "domain": "iea.org"
    }
  ],
  "min_statistics": 5,
  "max_statistics": 20
}

Response:
{
  "topic": "climate change",
  "candidates": [
    {
      "name": "Renewable energy growth",
      "value": 83,
      "unit": "%",
      "source": "iea.org",
      "source_url": "https://www.iea.org/...",
      "excerpt": "Renewable capacity grew by 83% in 2023..."
    }
  ],
  "sources_analyzed": 5,
  "timestamp": "2025-12-13T10:30:15Z"
}
```

### 3. Verification Worker
**Role**: Fact Checking
**Technology**: LLM (light usage)
**LLM-Light**

**Tasks**:

- Re-fetch source URLs
- Verify excerpts exist verbatim in source
- Check numerical values match
- Flag hallucinations or mismatches
- **Output**: VerificationResult objects with pass/fail

**Files**:

- `workers/verification/worker.go` - Implements omniagent-worker.Worker interface

### 4. Coordinator
**Role**: Workflow Coordination
**Technology**: omniagent-worker.Coordinator
**No LLM** (deterministic workflow)

**Workflow**:

1. Call Research Worker → get URLs
2. Call Synthesis Worker → extract statistics from URLs
3. Call Verification Worker → validate statistics
4. Retry logic if needed
5. Return verified statistics

**Files**:

- `coordinator/coordinator.go` - omniagent-worker.Coordinator implementation
- `cmd/coordinator/main.go` - CLI entrypoint with AgentOps support

## Data Flow

```
User Request
     │
     ▼
┌─────────────────┐
│   Coordinator   │
└────────┬────────┘
         │
         │ 1. Search for sources
         ▼
┌─────────────────┐
│ Research Worker │ ──► Returns: [{url, title, snippet, domain}, ...]
└────────┬────────┘
         │
         │ 2. Extract statistics from URLs
         ▼
┌─────────────────┐
│Synthesis Worker │ ──► Returns: [{name, value, unit, source_url, excerpt}, ...]
└────────┬────────┘
         │
         │ 3. Verify statistics
         ▼
┌─────────────────┐
│Verification     │ ──► Returns: [{statistic, verified: true/false, reason}, ...]
│     Worker      │
└────────┬────────┘
         │
         ▼
    Verified Statistics
```

## Models (pkg/models/statistic.go)

### New Models Added:

```go
// SearchResult - Output from Research Agent
type SearchResult struct {
    URL      string
    Title    string
    Snippet  string
    Domain   string
    Position int
}

// SynthesisRequest - Input to Synthesis Agent
type SynthesisRequest struct {
    Topic         string
    SearchResults []SearchResult
    MinStatistics int
    MaxStatistics int
}

// SynthesisResponse - Output from Synthesis Agent
type SynthesisResponse struct {
    Topic           string
    Candidates      []CandidateStatistic
    SourcesAnalyzed int
    Timestamp       time.Time
}
```

## Implementation Status

### ✅ Completed (v0.9.0)

1. **omniagent-worker Migration**
   - All workers implement `omniagent-worker.Worker` interface
   - Coordinator uses `omniagent-worker.Coordinator`
   - AgentOps tracing via OpenTelemetry

2. **Workers Created**
   - `workers/research/worker.go` - Web search worker
   - `workers/synthesis/worker.go` - LLM extraction worker
   - `workers/verification/worker.go` - Validation worker

3. **Coordinator Created**
   - `coordinator/coordinator.go` - Workflow orchestration
   - `cmd/coordinator/main.go` - CLI entrypoint

4. **OmniSkill Integration**
   - `omniskill/stats.go` - OmniAgent skill interface

## Benefits of Worker Architecture

- **Unified Interface** - All workers implement omniagent-worker.Worker
- **Separation of Concerns** - Each worker has one job
- **Better Caching** - Research results can be reused
- **Parallel Processing** - Synthesize multiple URLs concurrently
- **Cost Optimization** - Only use LLM where needed (synthesis)
- **Easier Testing** - Mock each worker independently
- **AgentOps Tracing** - Full observability via OpenTelemetry
- **Flexibility** - Run in-process via Pool or as HTTP services
