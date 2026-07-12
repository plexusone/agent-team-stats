# LLM Provider Configuration

This document describes the LLM provider configuration system for the Statistics Agent Team.

## Overview

The system now supports configurable LLM providers, allowing you to choose between:
- **Gemini** (Google) - Default
- **Claude** (Anthropic) - Planned
- **OpenAI** - Planned
- **Ollama** (Local) - Planned

## Current Status

✅ **Gemini**: Fully supported and tested
⏳ **Claude**: Configuration ready, ADK integration pending
⏳ **OpenAI**: Configuration ready, ADK integration pending
⏳ **Ollama**: Configuration ready, ADK integration pending

## Configuration

### Environment Variables

#### LLM Provider Selection

```bash
# Choose your LLM provider
export LLM_PROVIDER=gemini  # Options: gemini, claude, openai, ollama
```

#### Provider-Specific API Keys

```bash
# For Gemini (default)
export GOOGLE_API_KEY=your-google-api-key
# or
export GEMINI_API_KEY=your-google-api-key

# For Claude (when supported)
export ANTHROPIC_API_KEY=your-anthropic-api-key
# or
export CLAUDE_API_KEY=your-anthropic-api-key

# For OpenAI (when supported)
export OPENAI_API_KEY=your-openai-api-key

# For Ollama (when supported)
export OLLAMA_URL=http://localhost:11434
```

#### Model Selection

```bash
# Override the default model for your chosen provider
export LLM_MODEL=gemini-2.0-flash-exp  # For Gemini
# export LLM_MODEL=claude-3-5-sonnet-20241022  # For Claude
# export LLM_MODEL=gpt-4  # For OpenAI
# export LLM_MODEL=llama3.2  # For Ollama
```

#### Generic Options

```bash
# Use generic API key (overrides provider-specific keys)
export LLM_API_KEY=your-api-key

# Custom base URL (for Ollama or custom endpoints)
export LLM_BASE_URL=http://localhost:11434
```

## Default Models by Provider

| Provider | Default Model |
|----------|---------------|
| Gemini | `gemini-2.0-flash-exp` |
| Claude | `claude-3-5-sonnet-20241022` |
| OpenAI | `gpt-4` |
| Ollama | `llama3.2` |

## Architecture

### Model Factory

The `pkg/llm/factory.go` file provides a centralized `ModelFactory` that:
1. Reads configuration from environment variables
2. Creates the appropriate LLM model based on `LLM_PROVIDER`
3. Returns a `model.LLM` interface compatible with Google ADK

### Integration Points

All workers use the model factory via OmniLLM:

- **Synthesis Worker** (`workers/synthesis/worker.go`) - LLM-heavy extraction
- **Verification Worker** (`workers/verification/worker.go`) - LLM-light validation

Example usage:
```go
// Create model using factory
modelFactory := llm.NewModelFactory(cfg)
model, err := modelFactory.CreateModel(ctx)
if err != nil {
    return nil, fmt.Errorf("failed to create model: %w", err)
}

log.Printf("Agent: Using %s", modelFactory.GetProviderInfo())
```

## Examples

### Using Gemini (Default)

```bash
export GOOGLE_API_KEY=your-google-api-key
make run-all-eino
```

### Using Claude (When Supported)

```bash
export LLM_PROVIDER=claude
export ANTHROPIC_API_KEY=your-anthropic-api-key
export LLM_MODEL=claude-3-5-sonnet-20241022
make run-all-eino
```

### Using OpenAI (When Supported)

```bash
export LLM_PROVIDER=openai
export OPENAI_API_KEY=your-openai-api-key
export LLM_MODEL=gpt-4-turbo
make run-all-eino
```

### Using Ollama (When Supported)

```bash
export LLM_PROVIDER=ollama
export OLLAMA_URL=http://localhost:11434
export LLM_MODEL=llama3.2
make run-all-eino
```

## Future Development

### Claude Support

Claude support requires:
1. ADK integration or custom HTTP client for Anthropic API
2. Adapter to convert Claude responses to ADK's `model.LLM` interface
3. Testing with Claude-specific prompt engineering

### OpenAI Support

OpenAI support requires:
1. ADK integration or custom HTTP client for OpenAI API
2. Adapter to convert OpenAI responses to ADK's `model.LLM` interface
3. Testing with OpenAI-specific parameters

### Ollama Support

Ollama support requires:
1. Custom HTTP client for Ollama API
2. Adapter to convert Ollama responses to ADK's `model.LLM` interface
3. Support for different local models (llama, mistral, etc.)
4. Streaming support for better performance

## Configuration Precedence

The system uses the following precedence for API keys:

1. `LLM_API_KEY` (if set, overrides all provider-specific keys)
2. Provider-specific key (`GEMINI_API_KEY`, `CLAUDE_API_KEY`, etc.)
3. Alternative provider key (`GOOGLE_API_KEY` for Gemini, `ANTHROPIC_API_KEY` for Claude)

## Error Handling

If you select an unsupported provider, the system will return a clear error message:

```
Error: claude support via ADK is not yet implemented - use gemini for now
Error: openai support via ADK is not yet implemented - use gemini for now
Error: ollama support via ADK is not yet implemented - use gemini for now
```

## Contributing

To add support for a new LLM provider:

1. Update `pkg/llm/factory.go`:
   - Add a new case in `CreateModel()`
   - Implement `create<Provider>Model()` method

2. Update `pkg/config/config.go`:
   - Add provider-specific configuration fields
   - Update `LoadConfig()` to read new environment variables
   - Add default model in `getDefaultModel()`

3. Update documentation:
   - Add provider to README.md
   - Update this LLM_CONFIGURATION.md
   - Update .env.example

4. Test thoroughly with the new provider

## See Also

- [README.md](README.md) - Main project documentation
- [.env.example](.env.example) - Example environment configuration
- [pkg/config/config.go](pkg/config/config.go) - Configuration implementation
- [pkg/llm/factory.go](pkg/llm/factory.go) - Model factory implementation
