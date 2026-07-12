# Statistics Agent Team

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/agent-team-stats/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/agent-team-stats/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/agent-team-stats/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/agent-team-stats/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/agent-team-stats/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/agent-team-stats/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/agent-team-stats
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/agent-team-stats
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-dev%20guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.dev/agent-team-stats
 [viz-svg]: https://img.shields.io/badge/Go-visualizaton-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fagent-team-stats
 [loc-svg]: https://tokei.rs/b1/github/plexusone/agent-team-stats
 [repo-url]: https://github.com/plexusone/agent-team-stats
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/agent-team-stats/blob/main/LICENSE

A multi-agent system for finding and verifying statistics from reputable web sources using Go, built with [omniagent-worker](https://github.com/plexusone/omniagent-worker), [Google ADK (Agent Development Kit)](https://github.com/google/adk-go), and [Eino](https://github.com/cloudwego/eino).

## Overview

This project implements a sophisticated multi-agent architecture that leverages LLMs and web search to find verifiable statistics, prioritizing well-known and respected publishers. The system ensures accuracy through automated verification of sources.

## Architecture

The system implements a **4-agent architecture** with clear separation of concerns:

> **Architecture**: Built with **Google ADK** for LLM-based operations. **Two orchestration options** available: ADK-based (LLM-driven decisions) and Eino-based (deterministic graph workflow). See [4_AGENT_ARCHITECTURE.md](4_AGENT_ARCHITECTURE.md) for complete details.

```
┌─────────────────────────────────────────────────────────┐
│                   User Request                          │
│              "Find climate change statistics"           │
└───────────────────┬─────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────────────┐
│            ORCHESTRATION AGENT                          │
│              (Port 8000 - Both ADK/Eino)                │
│  • Coordinates 4-agent workflow                         │
│  • Manages retry logic                                  │
│  • Ensures quality standards                            │
└─────┬─────────────┬────────────────┬────────────────────┘
      │             │                │
      ▼             ▼                ▼
┌────────────┐ ┌──────────┐ ┌─────────────────┐
│  RESEARCH  │ │SYNTHESIS │ │  VERIFICATION   │
│   AGENT    │ │  AGENT   │ │     AGENT       │
│ Port 8001  │ │Port 8004 │ │   Port 8002     │
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

### Worker Architecture (omniagent-worker)

The system uses [omniagent-worker](https://github.com/plexusone/omniagent-worker) for worker lifecycle, coordination, and observability.

#### 1. Research Worker (`workers/research/`) - Web Search Only
- **No LLM required** - Pure search functionality
- Implements `omniagent-worker.Worker` interface
- Web search via Serper/SerpAPI integration
- Returns URLs with metadata (title, snippet, domain)
- Prioritizes reputable sources (`.gov`, `.edu`, research orgs)
- Output: List of `SearchResult` objects

#### 2. Synthesis Worker (`workers/synthesis/`) - LLM Extraction
- **LLM-heavy** extraction worker
- Implements `omniagent-worker.Worker` interface
- Fetches webpage content from URLs
- Extracts numerical statistics using LLM analysis
- Finds verbatim excerpts containing statistics
- Creates `CandidateStatistic` objects with proper metadata

#### 3. Verification Worker (`workers/verification/`) - LLM Validation
- **LLM-light** validation worker
- Implements `omniagent-worker.Worker` interface
- Re-fetches source URLs to verify content
- Checks excerpts exist verbatim in source
- Validates numerical values match exactly
- Flags hallucinations and discrepancies
- Returns verification results with pass/fail reasons

#### 4. Coordinator (`coordinator/`) - Workflow Orchestration
- Built with `omniagent-worker.Coordinator`
- **Deterministic graph-based workflow** (no LLM for orchestration)
- Coordinates: Research → Synthesis → Verification
- Implements adaptive retry logic
- AgentOps tracing via OpenTelemetry
- Workflow: ValidateInput → Research → Synthesis → Verification → QualityCheck → Format

## Features

### Core Capabilities
- ✅ **Multi-agent pipeline** - Full verification workflow (Research → Synthesis → Verification) ⭐ **RECOMMENDED**
- ✅ **Real web search** - Google search via Serper/SerpAPI (30 URLs searched by default) 🔍
- ✅ **Comprehensive extraction** - Processes 15+ pages, reads 30K chars per page for thorough coverage
- ✅ **Source verification** - Validates excerpts and values match actual web pages
- ✅ **Human-in-the-loop retry** - Prompts user when partial results found
- ✅ **Reputable source prioritization** - Government, academic, research organizations

### Alternative Modes
- ✅ **Direct LLM mode** - Fast but uses LLM memory (⚠️ not recommended for statistics)
- ✅ **Hybrid mode** - LLM discovery + web verification (⚠️ low verification rate)
- ✅ **OpenAPI documentation** - Interactive Swagger UI for Direct agent (port 8005)

### Output Formats
- ✅ **ClaimsReport format** - Export as [structured-evaluation](https://github.com/plexusone/structured-evaluation) ClaimsReport via `?format=claims`
- ✅ **Source classification** - Authoritative sources (WHO, CDC, NASA, etc.) classified as high reliability

### Technical Stack
- ✅ **Multi-LLM providers** - Gemini, Claude, OpenAI, Ollama, xAI Grok via unified interface
- ✅ **Google ADK integration** - For LLM-based agents
- ✅ **Eino framework** - Deterministic graph orchestration ⭐ Recommended
- ✅ **A2A Protocol** - Agent-to-Agent interoperability (Google standard) 🔗
- ✅ **LLM Observability** - OmniObserve integration (Opik, Langfuse, Phoenix) 👁️
- ✅ **Huma v2** - OpenAPI 3.1 docs for Direct agent
- ✅ **MCP Server** - Integration with Claude Code and other MCP clients
- ✅ **Docker deployment** - Easy containerized setup 🐳

## Output Format

The system returns verified statistics in JSON format:

```json
[
  {
    "name": "Global temperature increase since pre-industrial times",
    "value": 1.1,
    "unit": "°C",
    "source": "IPCC Sixth Assessment Report",
    "source_url": "https://www.ipcc.ch/...",
    "excerpt": "Global surface temperature has increased by approximately 1.1°C since pre-industrial times...",
    "verified": true,
    "date_found": "2025-12-13T10:30:00Z"
  }
]
```

### Field Descriptions

- **name**: Description of the statistic
- **value**: Numerical value (float32)
- **unit**: Unit of measurement (e.g., "°C", "%", "million", "billion")
- **source**: Name of source organization/publication
- **source_url**: URL to the original source
- **excerpt**: Verbatim quote containing the statistic
- **verified**: Whether the verification agent confirmed it
- **date_found**: Timestamp when statistic was found

## Installation

### Prerequisites

- Go 1.21 or higher
- LLM API key for your chosen provider:
  - **Gemini** (default): Google API key (set as `GOOGLE_API_KEY` or `GEMINI_API_KEY`)
  - **Claude**: Anthropic API key (set as `ANTHROPIC_API_KEY` or `CLAUDE_API_KEY`)
  - **OpenAI**: OpenAI API key (set as `OPENAI_API_KEY`)
  - **xAI Grok**: xAI API key (set as `XAI_API_KEY`)
  - **Ollama**: Local Ollama installation (default: `http://localhost:11434`)
- Optional: API keys for search provider (Google Search, etc.)

### Setup

1. Clone the repository:
```bash
git clone https://github.com/plexusone/agent-team-stats.git
cd agent-team-stats
```

2. Install dependencies:
```bash
make install
# or
go mod download
```

3. Configure environment variables:
```bash
# For Gemini (default)
export GOOGLE_API_KEY="your-google-api-key"

# For Claude
export LLM_PROVIDER="claude"
export ANTHROPIC_API_KEY="your-anthropic-api-key"

# For OpenAI
export LLM_PROVIDER="openai"
export OPENAI_API_KEY="your-openai-api-key"

# For xAI Grok
export LLM_PROVIDER="xai"
export XAI_API_KEY="your-xai-api-key"

# For Ollama (local)
export LLM_PROVIDER="ollama"
export OLLAMA_URL="http://localhost:11434"
export LLM_MODEL="llama3:latest"

# Optional: Create .env file
cp .env.example .env
# Edit .env with your API keys
```

4. Build the agents:
```bash
make build
```

## Usage

You can run the system either with Docker (containerized) or locally. Choose the method that best fits your needs.

| Method | Best For | Command |
|--------|----------|---------|
| **Docker** 🐳 | Production, quick start, isolated environment | `docker-compose up -d` |
| **Local** 💻 | Development, debugging, customization | `make run-all-eino` |

### Quick Start with Docker 🐳

The fastest way to get started:

```bash
# Start all agents with Docker Compose
docker-compose up -d

# Test the orchestration endpoint
curl -X POST http://localhost:8000/orchestrate \
  -H "Content-Type: application/json" \
  -d '{"topic": "climate change", "min_verified_stats": 5}'

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

See [DOCKER.md](DOCKER.md) for complete Docker deployment guide.

---

### Local Development Setup

#### Running the Agents Locally

##### Option 1: Run all agents with Eino orchestrator (Recommended)
```bash
make run-all-eino
```

##### Option 2: Run all agents with ADK orchestrator
```bash
make run-all
```

##### Option 3: Run each agent separately (in different terminals)
```bash
# Terminal 1: Research Agent (ADK)
make run-research

# Terminal 2: Verification Agent (ADK)
make run-verification

# Terminal 3: Orchestration Agent (choose one)
make run-orchestration       # ADK version (LLM-based)
make run-orchestration-eino  # Eino version (deterministic, recommended)
```

#### Using the CLI

The CLI supports three modes: **Direct LLM search** (fast, like ChatGPT), **Direct + Verification** (hybrid), and **Multi-agent verification pipeline** (thorough, verified).

##### Direct Mode (⚠️ Not Recommended for Statistics)

Direct mode uses a single LLM call to find statistics from memory - similar to ChatGPT without web search:

```bash
# Start direct agent first
make run-direct

# Then query (fast but uses LLM memory)
./bin/stats-agent search "climate change" --direct
```

**Why Not Recommended for Statistics:**
- ❌ **Uses LLM memory** - Not real-time web search (training data up to Jan 2025)
- ❌ **Outdated URLs** - LLM guesses URLs where stats came from
- ❌ **Low accuracy** - Pages may have moved, changed, or be paywalled
- ⚠️ **0% verification rate** - When combined with `--direct-verify`, most claims fail

**When to Use:**
- ✅ General knowledge questions
- ✅ Concept explanations
- ✅ Quick brainstorming (accept unverified data)

**For statistics, use Pipeline mode instead** (see below)

##### Hybrid Mode (⚠️ Also Not Recommended - Low Verification Rate)

Combines Direct mode with verification, but suffers from the same LLM memory issues:

```bash
# Start both agents
make run-direct-verify

# Then query
./bin/stats-agent search "climate change" --direct --direct-verify
```

**Why Not Recommended:**
- ❌ **Low verification rate** - Typically 0-30% of LLM claims verify
- ❌ **Same LLM memory problem** - URLs are guessed, not from real search
- ⚠️ **Slow with poor results** - Verification overhead but few verified stats

**For reliable statistics, use Pipeline mode instead**

##### Multi-Agent Pipeline Mode (Thorough Verification)

For verified, web-scraped statistics (requires agents running):

```bash
# Start agents first
make run-all-eino

# Then in another terminal:
# Basic search with verification pipeline
./bin/stats-agent search "climate change"

# Request specific number of verified statistics
./bin/stats-agent search "global warming" --min-stats 15

# Increase candidate search space
./bin/stats-agent search "AI trends" --min-stats 10 --max-candidates 100

# Only reputable sources
./bin/stats-agent search "COVID-19 statistics" --reputable-only

# JSON output only
./bin/stats-agent search "renewable energy" --output json

# Text output only
./bin/stats-agent search "climate data" --output text
```

**Advantages of Multi-Agent Mode:**
- ✅ **Verified sources** - Actually fetches and checks web pages
- 🔍 **Web search** - Finds current statistics from the web
- 🎯 **Accuracy** - Validates excerpts and values match
- 🔄 **Human-in-the-loop** - Prompts to continue if target not met

##### CLI Options

```bash
stats-agent search <topic> [options]

Options:
  -d, --direct              Use direct LLM search (fast, like ChatGPT)
      --direct-verify       Verify LLM claims with verification agent (requires --direct)
  -m, --min-stats <n>       Minimum statistics to find (default: 10)
  -c, --max-candidates <n>  Max candidates for pipeline mode (default: 50)
  -r, --reputable-only      Only use reputable sources
  -o, --output <format>     Output format: json, text, both (default: both)
      --orchestrator-url    Override orchestrator URL
  -v, --verbose             Show verbose debug information
      --version             Show version information
```

**Mode Comparison:**

| Mode | Speed | Accuracy | Agents Needed | Client Needs API Key? | Best For |
|------|-------|----------|---------------|----------------------|----------|
| `--direct` | ⚡⚡⚡ Fastest | ⚠️ LLM-claimed | Direct agent only | ❌ No | Quick research, brainstorming |
| `--direct --direct-verify` | ⚡⚡ Fast | ✅ Web-verified | Direct + Verification | ❌ No | Balanced speed + accuracy |
| Pipeline (default) | ⚡ Slower | ✅✅ Fully verified | All 4 agents | ❌ No | Maximum reliability |
```

---

### Using with Claude Code (MCP Server)

The system can be used as an MCP server with Claude Code and other MCP clients:

```bash
# Build the MCP server
make build-mcp

# Configure in Claude Code's MCP settings (see MCP_SERVER.md)
```

See [MCP_SERVER.md](MCP_SERVER.md) for detailed setup instructions.

### API Usage

You can also call the agents directly via HTTP (works with both Docker and local deployment):

```bash
# Call orchestration agent (port 8000 - supports both ADK and Eino)
curl -X POST http://localhost:8000/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "climate change",
    "min_verified_stats": 10,
    "max_candidates": 30,
    "reputable_only": true
  }'

# Get ClaimsReport format (structured-evaluation compatible)
curl -X POST "http://localhost:8000/orchestrate?format=claims" \
  -H "Content-Type: application/json" \
  -d '{"topic": "climate change", "min_verified_stats": 5}'
```

See [ClaimsReport Integration](docs/guides/claims-report.md) for detailed usage.

## Configuration

### Environment Variables

#### LLM Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_PROVIDER` | LLM provider: `gemini`, `claude`, `openai`, `xai`, `ollama` | `gemini` |
| `LLM_MODEL` | Model name (provider-specific) | See defaults below |
| `LLM_API_KEY` | Generic API key (overrides provider-specific) | - |
| `LLM_BASE_URL` | Base URL for custom endpoints (Ollama, etc.) | - |

**Provider-Specific API Keys:**
| Variable | Description | Default |
|----------|-------------|---------|
| `GOOGLE_API_KEY` / `GEMINI_API_KEY` | Google API key for Gemini | **Required for Gemini** |
| `ANTHROPIC_API_KEY` / `CLAUDE_API_KEY` | Anthropic API key for Claude | **Required for Claude** |
| `OPENAI_API_KEY` | OpenAI API key | **Required for OpenAI** |
| `XAI_API_KEY` | xAI API key for Grok | **Required for xAI** |
| `OLLAMA_URL` | Ollama server URL | `http://localhost:11434` |

**Default Models by Provider:**
- Gemini: `gemini-2.5-flash` (or `gemini-2.5-pro`)
- Claude: `claude-sonnet-4-20250514` (or `claude-opus-4-1-20250805`)
- OpenAI: `gpt-4o` (or `gpt-5`)
- xAI: `grok-4-1-fast-reasoning` (or `grok-4-1-fast-non-reasoning`)
- Ollama: `llama3:8b` (or `mistral:7b`)

See [LLM_CONFIGURATION.md](LLM_CONFIGURATION.md) for detailed LLM setup.

#### Search Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SEARCH_PROVIDER` | Search provider: `serper`, `serpapi` | `serper` |
| `SERPER_API_KEY` | Serper API key (get from serper.dev) | Required for real search |
| `SERPAPI_API_KEY` | SerpAPI key (alternative provider) | Required for SerpAPI |

**Note:** Without a search API key, the research agent will use mock data. See [SEARCH_INTEGRATION.md](SEARCH_INTEGRATION.md) for setup details.

#### Observability Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `OBSERVABILITY_ENABLED` | Enable LLM observability | `false` |
| `OBSERVABILITY_PROVIDER` | Provider: `opik`, `langfuse`, `phoenix` | `opik` |
| `OBSERVABILITY_API_KEY` | API key for the provider | - |
| `OBSERVABILITY_ENDPOINT` | Custom endpoint (optional) | Provider default |
| `OBSERVABILITY_PROJECT` | Project name for grouping traces | `stats-agent-team` |

**Supported Providers:**
- [Comet Opik](https://www.comet.com/site/products/opik/) - LLM tracing and evaluation
- [Langfuse](https://langfuse.com/) - Open-source LLM observability
- [Arize Phoenix](https://phoenix.arize.com/) - ML observability platform

#### Other Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `RESEARCH_AGENT_URL` | Research agent URL | `http://localhost:8001` |
| `SYNTHESIS_AGENT_URL` | Synthesis agent URL | `http://localhost:8004` |
| `VERIFICATION_AGENT_URL` | Verification agent URL | `http://localhost:8002` |
| `ORCHESTRATOR_URL` | Orchestrator URL (both ADK/Eino) | `http://localhost:8000` |

### Port Configuration

Each agent exposes both HTTP and A2A (Agent-to-Agent) protocol endpoints:

| Agent | HTTP Port | A2A Port | Description |
|-------|-----------|----------|-------------|
| **Orchestration (ADK/Eino)** ⭐ | **8000** | **9000** | Graph-based workflow coordination |
| Research (ADK) | 8001 | 9001 | Web search via Serper/SerpAPI |
| Verification (ADK) | 8002 | 9002 | LLM-based verification |
| Synthesis (ADK) | 8004 | 9004 | LLM-based statistics extraction |
| **Direct (Huma)** ⭐ | **8005** | - | Direct LLM search with OpenAPI docs |

**A2A Endpoints per Agent:**
- `GET /.well-known/agent-card.json` - Agent discovery
- `POST /invoke` - JSON-RPC execution

Enable A2A with: `A2A_ENABLED=true`

## Project Structure

```
agent-team-stats/
├── workers/                # omniagent-worker based workers
│   ├── research/           # Research worker (web search)
│   │   └── worker.go
│   ├── synthesis/          # Synthesis worker (LLM extraction)
│   │   └── worker.go
│   └── verification/       # Verification worker (LLM validation)
│       └── worker.go
├── coordinator/            # Workflow coordinator
│   └── coordinator.go      # omniagent-worker.Coordinator implementation
├── omniskill/              # OmniAgent skill integration
│   └── stats.go            # Skill interface for omniagent
├── cmd/
│   └── coordinator/        # Coordinator CLI entrypoint
│       └── main.go
├── pkg/
│   ├── config/            # Configuration management
│   ├── direct/            # Direct LLM search service
│   ├── llm/               # Multi-provider LLM factory (OmniLLM + OmniObserve)
│   │   └── adapters/      # OmniLLM adapter for ADK integration
│   ├── models/            # Shared data models
│   └── orchestration/     # Orchestration logic
├── main.go                # CLI entry point
├── Makefile               # Build and run commands
├── go.mod                 # Go dependencies
├── .env.example           # Environment template
└── README.md              # This file
```

## Development

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Cleaning Build Artifacts

```bash
make clean
```

## Technology Stack

- **Language**: Go 1.21+
- **Worker Framework**:
  - [omniagent-worker](https://github.com/plexusone/omniagent-worker) - Go-first multi-agent worker framework ⭐
- **Agent Frameworks**:
  - [Google ADK (Agent Development Kit)](https://github.com/google/adk-go) - LLM-based agents + A2A protocol
  - [Eino](https://github.com/cloudwego/eino) - Deterministic graph orchestration
- **LLM Integration**:
  - [OmniLLM](https://github.com/plexusone/omnillm) - Multi-provider LLM abstraction
  - Supports: Gemini, Claude, OpenAI, xAI Grok, Ollama
- **Observability**:
  - [OmniObserve](https://github.com/plexusone/omniobserve) - Unified LLM observability
  - AgentOps tracing via OpenTelemetry
  - Supports: Comet Opik, Langfuse, Arize Phoenix
- **Protocols**:
  - HTTP - Custom security, flexibility (ports 800x)
  - A2A - Agent-to-Agent interoperability (ports 900x)
- **Search**:
  - [OmniSerp](https://github.com/plexusone/omniserp) - Unified serp API abstraction
  - Supports: Serper.dev, SerpAPI 

## How It Works

1. **User Request**: User provides a topic via CLI or API
2. **Orchestration**: Orchestrator receives request and initiates workflow
3. **Research Phase**: Research agent searches web for candidate statistics
4. **Verification Phase**: Verification agent validates each candidate
5. **Quality Control**: Orchestrator checks if minimum verified stats met
6. **Retry Logic**: If needed, request more candidates and verify
7. **Response**: Return verified statistics in structured JSON format

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

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features including Perplexity integration, multi-language support, and browser extension.

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Acknowledgments

- Worker framework: [omniagent-worker](https://github.com/plexusone/omniagent-worker)
- Agent frameworks: [Google ADK](https://github.com/google/adk-go), [Eino](https://github.com/cloudwego/eino)
- Multi-LLM support: [OmniLLM](https://github.com/plexusone/omnillm)
- LLM observability: [OmniObserve](https://github.com/plexusone/omniobserve)
- A2A protocol for agent interoperability
- Inspired by multi-agent collaboration frameworks
