# Statistics Agent Team

A multi-agent system for finding and verifying statistics from reputable web sources using Go, built with [Google ADK (Agent Development Kit)](https://github.com/google/adk-go) and [Eino](https://github.com/cloudwego/eino).

## Overview

This project implements a sophisticated multi-agent architecture that leverages LLMs and web search to find verifiable statistics, prioritizing well-known and respected publishers. The system ensures accuracy through automated verification of sources.

## Features

### Core Capabilities

- **Multi-agent pipeline** - Full verification workflow (Research → Synthesis → Verification)
- **Real web search** - Google search via Serper/SerpAPI (30 URLs searched by default)
- **Comprehensive extraction** - Processes 15+ pages, reads 30K chars per page for thorough coverage
- **Source verification** - Validates excerpts and values match actual web pages
- **Human-in-the-loop retry** - Prompts user when partial results found
- **Reputable source prioritization** - Government, academic, research organizations

### Alternative Modes

- **Direct LLM mode** - Fast but uses LLM memory (not recommended for statistics)
- **Hybrid mode** - LLM discovery + web verification
- **OpenAPI documentation** - Interactive Swagger UI for Direct agent (port 8005)

### Technical Stack

- **Multi-LLM providers** - Gemini, Claude, OpenAI, Ollama, xAI Grok via unified interface
- **Google ADK integration** - For LLM-based agents
- **Eino framework** - Deterministic graph orchestration (Recommended)
- **A2A Protocol** - Agent-to-Agent interoperability (Google standard)
- **LLM Observability** - OmniObserve integration (Opik, Langfuse, Phoenix)
- **Huma v2** - OpenAPI 3.1 docs for Direct agent
- **MCP Server** - Integration with Claude Code and other MCP clients
- **Docker deployment** - Easy containerized setup

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

| Field | Description |
|-------|-------------|
| `name` | Description of the statistic |
| `value` | Numerical value (float32) |
| `unit` | Unit of measurement (e.g., "°C", "%", "million", "billion") |
| `source` | Name of source organization/publication |
| `source_url` | URL to the original source |
| `excerpt` | Verbatim quote containing the statistic |
| `verified` | Whether the verification agent confirmed it |
| `date_found` | Timestamp when statistic was found |

## Technology Stack

- **Language**: Go 1.21+
- **Agent Frameworks**:
    - [Google ADK (Agent Development Kit)](https://github.com/google/adk-go) - LLM-based agents + A2A protocol
    - [Eino](https://github.com/cloudwego/eino) - Deterministic graph orchestration
- **LLM Integration**:
    - [OmniLLM](https://github.com/agentplexus/omnillm) - Multi-provider LLM abstraction
    - Supports: Gemini, Claude, OpenAI, xAI Grok, Ollama
- **Observability**:
    - [OmniObserve](https://github.com/agentplexus/omniobserve) - Unified LLM observability
    - Supports: Comet Opik, Langfuse, Arize Phoenix
- **Protocols**:
    - HTTP - Custom security, flexibility (ports 800x)
    - A2A - Agent-to-Agent interoperability (ports 900x)
- **Search**:
    - [OmniSerp](https://github.com/agentplexus/omniserp) - Unified serp API abstraction
    - Supports: Serper.dev, SerpAPI

## How It Works

1. **User Request**: User provides a topic via CLI or API
2. **Orchestration**: Orchestrator receives request and initiates workflow
3. **Research Phase**: Research agent searches web for candidate statistics
4. **Verification Phase**: Verification agent validates each candidate
5. **Quality Control**: Orchestrator checks if minimum verified stats met
6. **Retry Logic**: If needed, request more candidates and verify
7. **Response**: Return verified statistics in structured JSON format

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Acknowledgments

- Built with [Google ADK (Agent Development Kit)](https://github.com/google/adk-go)
- Uses [Eino](https://github.com/cloudwego/eino) for deterministic orchestration
- Multi-LLM support via [OmniLLM](https://github.com/agentplexus/omnillm)
- LLM observability via [OmniObserve](https://github.com/agentplexus/omniobserve)
- A2A protocol for agent interoperability
- Inspired by multi-agent collaboration frameworks
