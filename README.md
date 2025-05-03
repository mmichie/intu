# intu

intu is an AI-powered command-line tool that leverages language models to assist with various tasks, including code analysis, file content processing, git operations, and interactive AI collaborations.

## Features

- Process input with AI using custom prompts
- Generate git commit messages and code reviews
- Support for multiple AI providers (OpenAI, Claude, Gemini, Grok)
- Interactive Text User Interface (TUI)
- Advanced AI pipeline architecture (serial, parallel, collaborative)
- AI-powered code and security reviews
- Jury-based evaluation system for AI responses
- Extensible filter system for text processing

## Commands

- `ai`:
  - `models`: List available AI models
  - `ask`: Ask the AI a free-form question
  - `jury`: Use an AI jury to evaluate responses
  - `collab`: Collaborative discussion between AI providers
  - `pipeline`: Run a predefined pipeline
- `cat`: Concatenate and display file contents with optional filters
- `commit`: Generate a git commit message based on the provided diff
- `codereview`: Generate code review for files
- `securityreview`: Perform security reviews for code
- `tui`: Start interactive Text User Interface
- `ls`: List files and directories with metadata
- `grep`: Search for patterns in files
- `glob`: Find files matching patterns
- `read`: Read and display file contents
- `edit`: Edit files with precise replacements
- `write`: Create or overwrite files
- `bash`: Execute shell commands with permission checking
- `batch`: Execute multiple tools in parallel
- `task`: Execute complex operations with an AI agent

## Installation

### From Source

1. Ensure you have Go installed (1.16+)
2. Clone the repository:
   ```
   git clone https://github.com/mmichie/intu.git
   cd intu
   ```
3. Build the project:
   ```
   make build
   ```
4. The compiled binary will be in the `bin` directory

### Cross-Platform Builds

To build for specific platforms:
```
make build-linux      # Linux amd64
make build-linux-arm64  # Linux arm64
make build-darwin     # macOS amd64
make build-darwin-arm64 # macOS arm64
make build-windows    # Windows amd64
make build-all        # All platforms
```

## Usage

```
intu [command] [flags]
```

For detailed usage of each command, use the `--help` flag:

```
intu [command] --help
```

### AI Providers

intu requires at least one AI provider API key:

- OpenAI: Set `OPENAI_API_KEY` environment variable
- Claude: Set `CLAUDE_API_KEY` environment variable
- Gemini: Set `GEMINI_API_KEY` environment variable
- Grok: Set `GROK_API_KEY` environment variable

### Examples

Generate a commit message:
```
intu commit
```

Get AI to analyze code:
```
intu ai ask "Explain the functionality of this code" main.go
```

Run a security review on a file:
```
intu securityreview pkg/aikit/providers/openai.go
```

Use the AI collaboration feature:
```
intu ai collab "How should we implement caching?" --providers=openai,claude
```

Use AI jury to evaluate responses:
```
intu ai jury "What's the best approach for error handling?" --jurors=3
```

Display file contents with filter:
```
intu cat --filter=go main.go
```

Run a custom pipeline:
```
intu ai pipeline codereview pkg/aikit/pipeline.go
```

Interactive mode:
```
intu tui
```

Code review multiple files:
```
intu codereview commands/ai.go commands/ai_handlers.go
```

List available AI models:
```
intu ai models
```

Run multiple tools in parallel:
```
intu batch -f examples/batch_example.json
```

Execute complex tasks with an AI agent:
```
intu task -f examples/task_example.txt
```

## Configuration

intu uses a configuration file located at `$HOME/.intu.yaml`. You can specify a different config file using the `--config` flag.

## Filters

intu includes the following filters:
- Go AST Compressor: Compresses Go source code
- TF-IDF: Filters text based on term frequency-inverse document frequency

## Advanced Features

### AI Pipelines

intu supports advanced AI pipeline architectures:
- Serial: Pass output from one provider to another
  ```
  intu ai pipeline serial "Optimize this code" --providers=gemini,claude main.go
  ```

- Parallel: Run multiple providers and combine results
  ```
  intu ai pipeline parallel "Generate unit tests" --providers=openai,claude,gemini pkg/aikit/agent.go
  ```

- Collaborative: Multi-round discussions between AI providers
  ```
  intu ai collab "How would you refactor this code?" --rounds=3 --providers=claude,openai fileops/fileops.go
  ```

- Nested: Compose multiple pipelines together
  ```
  intu ai pipeline nested "Design a new feature" --config=pipelines.yaml
  ```

### Jury System

The jury system allows multiple AI "jurors" to evaluate and select the best response using different voting methods:
```
intu ai jury "What's the best database for this use case?" --jurors=5 --voting=weighted
```

## Testing

Run the tests using:

```
go test ./...
```

For verbose test output:
```
make test-verbose
```