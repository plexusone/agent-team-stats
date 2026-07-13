# Configuration

## Environment Variables

### LLM Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_PROVIDER` | LLM provider: `gemini`, `claude`, `openai`, `xai`, `ollama` | `gemini` |
| `LLM_MODEL` | Model name (provider-specific) | See defaults below |
| `LLM_API_KEY` | Generic API key (overrides provider-specific) | - |
| `LLM_BASE_URL` | Base URL for custom endpoints (Ollama, etc.) | - |

### Provider-Specific API Keys

| Variable | Description | Default |
|----------|-------------|---------|
| `GOOGLE_API_KEY` / `GEMINI_API_KEY` | Google API key for Gemini | **Required for Gemini** |
| `ANTHROPIC_API_KEY` / `CLAUDE_API_KEY` | Anthropic API key for Claude | **Required for Claude** |
| `OPENAI_API_KEY` | OpenAI API key | **Required for OpenAI** |
| `XAI_API_KEY` | xAI API key for Grok | **Required for xAI** |
| `OLLAMA_URL` | Ollama server URL | `http://localhost:11434` |

### Default Models by Provider

| Provider | Default Model | Alternative |
|----------|---------------|-------------|
| Gemini | `gemini-2.5-flash` | `gemini-2.5-pro` |
| Claude | `claude-sonnet-4-20250514` | `claude-opus-4-1-20250805` |
| OpenAI | `gpt-4o` | `gpt-5` |
| xAI | `grok-4-1-fast-reasoning` | `grok-4-1-fast-non-reasoning` |
| Ollama | `llama3:8b` | `mistral:7b` |

See [LLM Configuration](../guides/llm-configuration.md) for detailed LLM setup.

### Search Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SEARCH_PROVIDER` | Search provider: `serper`, `serpapi` | `serper` |
| `SERPER_API_KEY` | Serper API key (get from serper.dev) | Required for real search |
| `SERPAPI_API_KEY` | SerpAPI key (alternative provider) | Required for SerpAPI |

!!! note
    Without a search API key, the research agent will use mock data. See [Search Integration](../guides/search-integration.md) for setup details.

### Observability Configuration

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

### Coordinator Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `COORDINATOR_PORT` | Coordinator HTTP port | `8080` |
| `AGENTOPS_DSN` | AgentOps tracing DSN (postgres) | - |

## Deployment Modes

Workers can run in two modes:

| Mode | Description | Use Case |
|------|-------------|----------|
| **In-Process** (default) | Workers run within the Coordinator via Pool | Development, single-process deployment |
| **HTTP Services** | Workers run as separate HTTP services | Microservices, scaling individual workers |

**Default Port (Coordinator):** `8080`

### A2A Protocol Support

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

## Development Commands

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
