# intu

intu is an AI-powered command-line tool that leverages language models to assist with various tasks, including file content analysis and generating git commit messages.

## Features

- Process input with AI using custom prompts
- Concatenate and display file contents with optional filters
- Generate git commit messages based on diffs
- Support for multiple AI providers (OpenAI and Claude)
- Extensible filter system for text processing

## Commands

- `ai`: Process input with AI using a custom prompt
- `cat`: Concatenate and display file contents with optional filters
- `commit`: Generate a git commit message based on the provided diff

## Installation

[Add installation instructions here]

## Usage

```
intu [command] [flags]
```

For detailed usage of each command, use the `--help` flag:

```
intu [command] --help
```

## Configuration

intu uses a configuration file located at `$HOME/.intu.yaml`. You can specify a different config file using the `--config` flag.

## AI Providers

intu supports the following AI providers:
- OpenAI (default)
- Claude

Set the `OPENAI_API_KEY` or `CLAUDE_API_KEY` environment variable to use the respective provider.

## Filters

intu includes the following filters:
- Go AST Compressor: Compresses Go source code
- TF-IDF: Filters text based on term frequency-inverse document frequency

## Testing

Run the tests using:

```
go test ./...
```
