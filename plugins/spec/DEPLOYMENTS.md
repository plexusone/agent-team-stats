# Deployment Targets

This document describes the deployment targets for the stats-agent-team multi-agent system.

## Priority 1 (P1) - Primary Targets

### Local Claude Code Sub-Agents

- **Target**: `local-claude`
- **Output**: `.claude/agents/`
- **Generator**: `aiassistkit/agents/claude`
- **Use Case**: Local development with Claude Code CLI

The Claude adapter generates Markdown files with YAML front matter in the format expected by Claude Code's agent system. Each agent becomes a spawnable sub-agent via the Task tool.

**Generation Command**:

```bash
genagents -spec=plugins/spec/agents -output=.claude/agents -format=claude
```

**Features**:

- Native integration with Claude Code Task tool
- Model names: `sonnet`, `haiku`, `opus`
- Tool names: `WebSearch`, `WebFetch`, `Read`, `Write`, `Glob`, `Grep`

### Local Kiro Sub-Agents

- **Target**: `local-kiro`
- **Output**: `plugins/kiro/agents/`
- **Generator**: `aiassistkit/agents/kiro`
- **Use Case**: Local development with Kiro CLI

The Kiro adapter generates JSON files with snake_case tool names and Kiro-specific model identifiers.

**Generation Command**:

```bash
genagents -spec=plugins/spec/agents -output=plugins/kiro/agents -format=kiro
```

**Features**:

- JSON format for Kiro agent definitions
- Model names: `claude-sonnet-4`, `claude-haiku-35`, `claude-opus-4`
- Tool names: `web_search`, `web_fetch`, `read`, `write`

### AWS Bedrock AgentCore

- **Target**: `aws-agentcore`
- **Output**: `cdk/`
- **Generator**: `agentkit-aws-cdk` or `agentkit-aws-pulumi`
- **Use Case**: Production deployment on AWS

Deploys the agent team as AWS Bedrock agents with Lambda functions for tool implementations.

**Components**:

- Bedrock Agent definitions with foundation model configuration
- Lambda functions for WebSearch, WebFetch tools
- IAM roles and policies for agent execution
- API Gateway for agent invocation

**Status**: Implementation planned

## Priority 2 (P2) - Secondary Targets

### Kubernetes / Helm

- **Target**: `kubernetes`
- **Output**: `helm/`
- **Generator**: Helm chart templates
- **Use Case**: Non-AWS cloud deployments (Azure, GCP, on-prem)

Generates Helm charts for deploying the agent team as containerized services.

**Components**:

- Deployment manifests for each agent
- Service definitions for inter-agent communication
- ConfigMaps for agent instructions
- Secrets for API keys

**Platforms Supported**:

- Azure Kubernetes Service (AKS)
- Google Kubernetes Engine (GKE)
- On-premise Kubernetes clusters
- Minikube for local testing

**Status**: Implementation planned

### AgentKit Local MCP Server

- **Target**: `agentkit-local`
- **Output**: `plugins/agentkit/`
- **Generator**: `agentkit/platforms/local`
- **Use Case**: CLI assistants lacking native sub-agent support

Provides an MCP (Model Context Protocol) server that exposes agents as tools for CLI assistants like Gemini CLI or OpenAI Codex that don't have native sub-agent capabilities.

**Components**:

- MCP server configuration
- Agent definitions in agentkit format
- Tool implementations using agentkit runner

**Status**: Implementation in progress

## Multi-Target Generation

Generate for multiple targets in a single command:

```bash
genagents -spec=plugins/spec/agents \
  -targets="claude:.claude/agents,kiro:plugins/kiro/agents" \
  -verbose
```

## Architecture Overview

```
plugins/spec/agents/           # Canonical agent definitions (Markdown)
    |
    +-- genagents CLI
    |       |
    +-------+-------+-------+-------+
    |       |       |       |       |
    v       v       v       v       v
.claude/  kiro/    cdk/    helm/   agentkit/
agents/   agents/
```

The canonical Markdown specs serve as the single source of truth. Platform-specific generators produce deployment artifacts for each target environment.
