# MCP Server for Statistics Agent Team

This document explains how to use the Statistics Agent Team as an MCP (Model Context Protocol) server with MCP clients like Claude Code.

## Overview

The MCP server exposes the Statistics Agent Team's functionality through the Model Context Protocol, allowing AI assistants like Claude to search for and verify statistics on any topic using a multi-agent system.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  MCP Client                             │
│              (e.g., Claude Code)                        │
└──────────────────────┬──────────────────────────────────┘
                       │ MCP Protocol (stdio)
                       │
┌──────────────────────▼──────────────────────────────────┐
│              MCP Server (stats-agent-team)              │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Embedded Eino Orchestration Agent                 │ │
│  │  (Deterministic Graph Workflow)                    │ │
│  └──────────────────────┬─────────────────────────────┘ │
└─────────────────────────┼───────────────────────────────┘
                          │ HTTP
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
┌────────────┐     ┌────────────┐     ┌────────────────┐
│ Research   │────▶│ Synthesis  │────▶│ Verification   │
│ Agent      │     │ Agent      │     │ Agent          │
│ (Web URLs) │     │ (Extract   │     │ (Validate      │
│ :8001      │     │  Stats)    │     │  Sources)      │
└────────────┘     │ :8004      │     │ :8002          │
                   └────────────┘     └────────────────┘
```

> **Note:** The MCP server **embeds** the Eino orchestration logic directly in-process. You only need to run the three worker agents (Research, Synthesis, Verification) when using the MCP server. The standalone orchestration agent (`orchestration-eino` on port 8000) is only needed for direct HTTP API access, not for MCP usage.

## Features

The MCP server provides one tool:

### `search_statistics`

Search for verified statistics on a given topic using a multi-agent system.

**Parameters:**
- `topic` (string, required): The topic to search statistics for
- `min_verified_stats` (number, optional): Minimum number of verified statistics to return (default: 10)
- `max_candidates` (number, optional): Maximum number of candidate statistics to gather (default: 30)
- `reputable_only` (boolean, optional): Only use reputable sources (default: true)

**Returns:**
- Markdown-formatted results with verified statistics
- JSON output with all statistics
- Human-readable format with sources and verification details

## Prerequisites

1. **Go 1.21+** installed
2. **API Keys** configured:
   - `GOOGLE_API_KEY` for Gemini (default LLM)
   - Or other LLM provider keys as configured
3. **All three agents** running:
   ```bash
   # Terminal 1: Research Agent (port 8001)
   ./bin/research

   # Terminal 2: Synthesis Agent (port 8004)
   ./bin/synthesis

   # Terminal 3: Verification Agent (port 8002)
   ./bin/verification
   ```
   Or use the convenience target:
   ```bash
   make run-all-eino
   ```

## Building

```bash
# Build the MCP server
go build -o bin/mcp-server ./mcp/server/main.go

# Or use make
make build-mcp
```

## Configuration

### Environment Variables

The MCP server uses the same configuration as other agents:

```bash
# LLM Configuration
export LLM_PROVIDER=gemini  # or claude, openai, ollama, xai
export GOOGLE_API_KEY=your-api-key

# Agent URLs (defaults shown)
export RESEARCH_AGENT_URL=http://localhost:8001
export SYNTHESIS_AGENT_URL=http://localhost:8004
export VERIFICATION_AGENT_URL=http://localhost:8002
```

See [LLM_CONFIGURATION.md](LLM_CONFIGURATION.md) for full configuration options.

## Using with Claude Code

### 1. Add to Claude Code Configuration

Add the MCP server to your Claude Code MCP settings file:

**Location:** `~/.config/claude-code/mcp-settings.json` (or similar)

```json
{
  "mcpServers": {
    "stats-agent-team": {
      "command": "/path/to/stats-agent-team/bin/mcp-server",
      "env": {
        "GOOGLE_API_KEY": "your-google-api-key",
        "RESEARCH_AGENT_URL": "http://localhost:8001",
        "SYNTHESIS_AGENT_URL": "http://localhost:8004",
        "VERIFICATION_AGENT_URL": "http://localhost:8002"
      }
    }
  }
}
```

### 2. Start Required Agents

Before using the MCP server, start all three agents:

```bash
# Terminal 1: Research Agent (port 8001)
cd /path/to/stats-agent-team
export GOOGLE_API_KEY=your-key
./bin/research

# Terminal 2: Synthesis Agent (port 8004)
cd /path/to/stats-agent-team
export GOOGLE_API_KEY=your-key
./bin/synthesis

# Terminal 3: Verification Agent (port 8002)
cd /path/to/stats-agent-team
export GOOGLE_API_KEY=your-key
./bin/verification
```

Or start all agents at once:

```bash
cd /path/to/stats-agent-team
export GOOGLE_API_KEY=your-key
make run-all-eino
```

### 3. Use in Claude Code

The MCP server will automatically start when Claude Code launches. You can now use it by asking Claude to search for statistics:

**Example prompts:**
- "Search for statistics about climate change"
- "Find verified statistics on AI adoption rates"
- "Get statistics about cybersecurity threats in 2024"

Claude will use the `search_statistics` tool to find and verify statistics from reputable sources.

## Optional Source Checks with Parallel Search MCP

Add [Parallel Search MCP](https://docs.parallel.ai/integrations/mcp/search-mcp) alongside the statistics server in Claude Code to discover primary sources and read the context around a statistic. It provides `web_search` for source discovery and `web_fetch` for page excerpts. Anonymous search uses Fast mode and is free for light use, with no Parallel account or API key required. Usage limits apply; see the linked guide for authentication options when you need higher limits.

From the project directory where you use Claude Code, add the companion with local scope:

```bash
claude mcp add --transport http --scope local \
  parallel-search https://search.parallel.ai/mcp \
  --header "User-Agent: agent-team-stats/parallel-mcp"

claude mcp get parallel-search
```

Local scope keeps this server private to you in the current project. Keep your existing `stats-agent-team` registration. Start a new Claude Code session in that directory and run `/mcp` to confirm that `parallel-search` is connected and exposes `web_search` and `web_fetch`. See [Claude Code's MCP guide](https://code.claude.com/docs/en/mcp) for server management.

With the statistics server and its workers configured as above, try this prompt:

```text
Research US adult cigarette smoking prevalence using primary sources.
1. Use parallel-search's web_search to find recent CDC source pages.
2. Call stats-agent-team's search_statistics for the same topic with
   min_verified_stats=3, max_candidates=20, and reputable_only=true.
3. Use parallel-search's web_fetch to read the source URLs returned by
   search_statistics, and the CDC pages found in step 1.
4. Compare each figure's quoted evidence, population, units, and data year.
   Return the source URL and exact supporting excerpt for each figure.
   Flag unavailable evidence or discrepancies instead of filling in gaps.
   Separate the tool's verification status from your additional source checks.
```

This is a client-side research workflow. The Go research pipeline still uses its configured Serper or SerpAPI provider and still needs its existing credentials. `search_statistics` accepts a topic, not a list of source URLs; Parallel results are not injected into the pipeline. Additional source checks do not change the tool's verification status. If a page cannot be fetched or the companion reaches its usage limit, report the missing evidence and retry later.

To remove only the companion from this project:

```bash
claude mcp remove --scope local parallel-search
```

## Example Usage

### Request

```json
{
  "topic": "climate change",
  "min_verified_stats": 5,
  "max_candidates": 20,
  "reputable_only": true
}
```

### Response

The tool returns formatted markdown output:

```markdown
# Statistics Search Results

**Topic:** climate change
**Verified:** 5 statistics
**Failed:** 3 statistics
**Total Candidates:** 20
**Timestamp:** 2025-12-13 10:30:00

## JSON Output

```json
[
  {
    "name": "Global temperature increase since pre-industrial times",
    "value": 1.1,
    "unit": "°C",
    "source": "IPCC Sixth Assessment Report",
    "source_url": "https://www.ipcc.ch/...",
    "excerpt": "Global surface temperature has increased by approximately 1.1°C...",
    "verified": true,
    "date_found": "2025-12-13T10:30:00Z"
  }
]
```

## Verified Statistics

### 1. Global temperature increase since pre-industrial times

- **Value:** 1.1 °C
- **Source:** IPCC Sixth Assessment Report
- **URL:** https://www.ipcc.ch/...
- **Excerpt:** "Global surface temperature has increased by approximately 1.1°C..."
- **Verified:** ✓
- **Date Found:** 2025-12-13
```

## Troubleshooting

### MCP Server Not Starting

1. **Check logs:** The MCP server logs to stderr, which Claude Code captures
2. **Verify agents are running:** All three agents (Research, Synthesis, Verification) must be running
3. **Check API keys:** Ensure `GOOGLE_API_KEY` or other LLM provider keys are set

### No Statistics Found

1. **Check research agent:** Ensure it's running on port 8001
2. **Check synthesis agent:** Ensure it's running on port 8004
3. **Check verification agent:** Ensure it's running on port 8002
4. **Review topic:** Try a more specific or different topic

### Connection Errors

1. **Check agent URLs:** Verify `RESEARCH_AGENT_URL`, `SYNTHESIS_AGENT_URL`, and `VERIFICATION_AGENT_URL` are correct
2. **Check network:** Ensure agents can communicate with each other
3. **Check ports:** Ensure ports 8001, 8004, and 8002 are not blocked

## Development

### Testing the MCP Server

You can test the MCP server manually using the MCP Inspector or by writing a simple client:

```bash
# Run the server (stdio mode)
export GOOGLE_API_KEY=your-key
./bin/mcp-server

# The server expects JSON-RPC messages on stdin
# See https://modelcontextprotocol.io for protocol details
```

### Logs

The MCP server logs to stderr:
- `[MCP] Server running on stdio transport`
- `[MCP] Searching for statistics on topic: ...`
- `[MCP] Found N verified statistics (from M candidates)`

## Architecture Details

### Communication Flow

1. **Claude Code → MCP Server:** Claude sends `tools/call` request via stdio
2. **MCP Server → Eino Orchestrator:** Orchestrator coordinates deterministic graph workflow
3. **Orchestrator → Research Agent:** HTTP POST to `/research` - finds relevant source URLs
4. **Orchestrator → Synthesis Agent:** HTTP POST to `/synthesize` - extracts statistics from sources
5. **Orchestrator → Verification Agent:** HTTP POST to `/verify` - validates statistics against sources
6. **MCP Server → Claude Code:** Returns formatted results via stdio

### Error Handling

- **Tool errors:** Returned with `isError: true` and error message in content
- **Network errors:** Logged and returned as tool errors
- **Validation errors:** Topic validation happens before orchestration

## See Also

- [README.md](README.md) - Main project documentation
- [LLM_CONFIGURATION.md](LLM_CONFIGURATION.md) - LLM provider configuration
- [MCP Protocol](https://modelcontextprotocol.io) - Official MCP documentation
- [Claude Code](https://claude.com/claude-code) - Claude Code documentation

## License

MIT - See [LICENSE](LICENSE) for details
