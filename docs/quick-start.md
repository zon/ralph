# Quick Start Guide

Get up and running with Ralph in 5 minutes!

## 1. Install Ralph

```bash
go install github.com/zon/ralph/cmd/ralph@latest
```

Ensure `$GOPATH/bin` is in your PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## 2. Set Up API Key

Create secrets file with your DeepSeek API key:

```bash
mkdir -p ~/.ralph
cat > ~/.ralph/secrets.yaml <<EOF
apiKeys:
  deepseek: sk-your-deepseek-api-key-here
EOF
```

Get your API key from [platform.deepseek.com](https://platform.deepseek.com/)

## 3. Authenticate GitHub

```bash
gh auth login
```

Follow the prompts to authenticate with GitHub.

## 4. Create a Project File

```bash
mkdir -p projects
cat > projects/my-first-feature.yaml <<EOF
name: my-first-feature
description: Add a hello world feature

requirements:
  - category: implementation
    description: Create a simple hello world function
    steps:
      - Add a new file with hello world function
      - Export the function
      - Add basic test
    passing: false
EOF
```

## 5. Test with Dry-Run

Preview what Ralph will do:

```bash
ralph once projects/my-first-feature.yaml --dry-run
```

## 6. Run Single Iteration

Execute one development cycle:

```bash
ralph once projects/my-first-feature.yaml
```

Ralph will:
- Generate a development prompt
- Run the AI agent to implement changes
- Stage the project file

## 7. Full Workflow (Optional)

For automatic branching and PR creation:

```bash
ralph run projects/my-first-feature.yaml --dry-run  # Preview
ralph run projects/my-first-feature.yaml            # Execute
```

Ralph will:
- Create a new git branch
- Run iterations until requirements pass
- Commit changes after each iteration
- Generate PR summary
- Push and create GitHub pull request

## What's Next?

- **Read the full docs**: [README.md](../README.md)
- **Configure services**: [Configuration Guide](configuration.md)
- **Explore examples**: Check `examples/` directory
- **Learn advanced usage**: See README for detailed command options

## Common First-Time Issues

### "ralph: command not found"
Add Go's bin to PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### "No API key found"
Create `~/.ralph/secrets.yaml` with your DeepSeek API key.

### "gh: not authenticated"
Run `gh auth login` to authenticate with GitHub.

### Need Help?
- [Installation Guide](../INSTALL.md)
- [Configuration Guide](configuration.md)
- [GitHub Issues](https://github.com/zon/ralph/issues)
