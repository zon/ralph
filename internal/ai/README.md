# AI Client Package

This package provides AI agent execution capabilities for Ralph, using DeepSeek as the LLM provider.

## Features

- **DeepSeek Integration** via go-deepseek client library
- **Dry-run mode** for testing without making API calls
- **PR summary generation** using AI to analyze commits and diffs
- **Configuration flexibility** via .ralph/config.yaml

## Supported Provider

### DeepSeek (Default)
- Uses DeepSeek API via go-deepseek library
- Default model: `deepseek-reasoner` (R1 - advanced reasoning model)
- Provides chain-of-thought reasoning for complex coding tasks
- API endpoint: `https://api.deepseek.com/v1/chat/completions`

## Configuration

API keys are loaded from secrets files (see `internal/config` package):
- `~/.ralph/secrets.yaml` (global)
- `.ralph/secrets.yaml` (project-specific, takes priority)

Example secrets file:
```yaml
apiKeys:
  deepseek: sk-xxxxxxxxxxxxx
```

Provider and model can be configured in `.ralph/config.yaml`:
```yaml
llmProvider: deepseek
llmModel: deepseek-reasoner  # optional, uses R1 by default for reasoning tasks
```

Available DeepSeek models:
- `deepseek-reasoner` (R1) - Advanced reasoning, best for complex coding (default)
- `deepseek-chat` (V3) - General purpose, also strong at coding
- `deepseek-coder` - Specialized for code generation and IDE tasks

## Functions

### RunAgent
Executes an AI agent with the given prompt.

```go
err := ai.RunAgent(ctx, prompt)
```

In dry-run mode, logs what would be executed without calling the API.

### GeneratePRSummary
Generates a pull request summary using AI, including project status, commits, and diff.

```go
summary, err := ai.GeneratePRSummary(ctx, projectFile, iterations)
```

### ValidateConfig
Checks if AI configuration is valid (provider has an API key).

```go
err := ai.ValidateConfig(ralphConfig, secrets)
```

## Error Handling

All functions return descriptive errors for:
- Missing API keys
- Network failures
- API errors
- Invalid responses

## Testing

Tests cover:
- Dry-run mode behavior
- Configuration validation
- Error cases (missing secrets, invalid projects)

Run tests:
```bash
go test ./internal/ai/... -v
```

## Dependencies

- `github.com/go-deepseek/deepseek` - Official DeepSeek Go client library
