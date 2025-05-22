# AIKit V2 API

This is the next-generation AI library API with improved design patterns.

## Key Improvements over V1

- **Unified Provider Interface**: All AI providers implement the same interface
- **Domain-specific Types**: Request/Response types instead of string-based APIs
- **Structured Error Handling**: Rich error types with context and provider information
- **Explicit Configuration**: Clear configuration with environment variable support
- **Composable Pipelines**: Serial, parallel, and collaborative execution patterns
- **Function Calling**: Centralized function definition and execution system
- **Thread-Safe Operations**: Concurrent-safe function registry with mutex protection
- **Base Provider Implementation**: Reduces code duplication by 40-50%
- **SSE Parser Abstraction**: Unified Server-Sent Events parsing for streaming
- **Validation Framework**: Comprehensive input/output validation with extensible rules

## Package Structure

```
pkg/aikit/v2/
├── config/          # Configuration management
├── errors/          # Domain-specific error types
├── function/        # Function calling definitions (thread-safe)
├── pipeline/        # Execution pipelines (serial, parallel, collaborative)
├── provider/        # Unified provider interface and implementations
├── streaming/       # SSE parser and stream processing utilities
└── validation/      # Request/response validation framework
```

## Quick Start

```go
import (
    "context"
    "github.com/mmichie/intu/pkg/aikit/v2/config"
    "github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// Create a provider
cfg := config.FromEnvironment("CLAUDE")
factory := &provider.ClaudeFactory{}
provider, err := factory.Create(cfg)

// Make a request
request := provider.Request{
    Prompt: "Hello, world!",
}
response, err := provider.GenerateResponse(context.Background(), request)
fmt.Println(response.Content)
```

## Migration Strategy

The V2 API is designed to coexist with V1. You can gradually migrate components:

1. Start using V2 for new features
2. Create adapters between V1 and V2 as needed
3. Migrate existing code component by component
4. Eventually deprecate V1 when migration is complete

## New Features Usage

### Thread-Safe Function Registry
```go
registry := function.NewRegistry()
registry.Register(function.FunctionDefinition{
    Name:        "get_weather",
    Description: "Get weather for a location",
    Parameters:  map[string]interface{}{"type": "object"},
})
```

### Validation
```go
validator := validation.NewRequestValidator()
if errors := validator.Validate(request); len(errors) > 0 {
    return errors
}
```

### SSE Streaming
```go
processor := streaming.NewJSONStreamProcessor(reader).
    WithJSONHandler(func(data json.RawMessage) error {
        // Process streaming JSON chunks
        return nil
    })
```

## Testing

Run all v2 tests:
```bash
go test ./pkg/aikit/v2/... -v
```

## Status

This API has been significantly enhanced with production-ready features including thread safety, validation, and improved streaming support. See [IMPROVEMENTS.md](./IMPROVEMENTS.md) for detailed changes.