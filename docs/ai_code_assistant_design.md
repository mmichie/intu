# AI Code Assistant Design Document

## Introduction

This document outlines the design for transforming Intu into a powerful AI-driven code assistant - a system that can understand codebases, execute commands, modify files, and assist with software development tasks through natural language interaction.

The design prioritizes:
- **LLM Provider Independence**: Works with any AI provider supporting function calling
- **Security**: Tiered permission model with sandboxed execution
- **Extensibility**: Modular tool architecture
- **User Experience**: Intuitive CLI/TUI interaction

## System Architecture

```
                             +-------------------+
                             | User Interaction  |
                             |  - CLI Interface  |
                             |  - TUI Interface  |
                             +--------+----------+
                                      |
                                      v
  +----------------+        +-------------------+        +---------------+
  | Security Layer |<------>| Command Processor |<------>| Context Store |
  | - Permissions  |        | - Command Parsing |        | - Session     |
  | - Sandboxing   |        | - Pipeline Mgmt   |        | - Project     |
  +----------------+        +--------+----------+        | - User        |
                                     |                   +---------------+
                                     v
  +-----------------+       +-------------------+        +---------------+
  | Tool Registry   |<----->| Function Calling  |<------>| Provider API  |
  | - Core Tools    |       | - LLM Integration |        | - Abstraction |
  | - User Tools    |       | - Tool Execution  |        | - Adapters    |
  | - MCP Tools     |       | - Batch Execution |        | - Models      |
  +-----------------+       +--------+----------+        +---------------+
          |                          |
          v                          v
  +-----------------------------------------------------+
  |                      Tool System                     |
  | +------------------+     +----------------------+    |
  | | Read-Only Tools  |     | File Editing Tools   |    |
  | | - LS             |     | - Edit               |    |
  | | - Grep           |     | - Write              |    |
  | | - Glob           |     +----------------------+    |
  | | - Read           |     +----------------------+    |
  | +------------------+     | Execution Tools      |    |
  | +------------------+     | - Bash               |    |
  | | Utility Tools    |     | - NotebookEdit       |    |
  | | - Task           |     +----------------------+    |
  | | - Batch          |     +----------------------+    |
  | | - TodoRead       |     | Network Tools        |    |
  | | - TodoWrite      |     | - WebFetch           |    |
  | +------------------+     +----------------------+    |
  +-----------------------------------------------------+

  +-----------------------------------------------------+
  |                   File Operations                    |
  | - Finding        - Metadata      - Editing          |
  | - Reading        - Writing       - Permissions      |
  +-----------------------------------------------------+
```

## LLM Provider Abstraction Layer

The system is designed to be LLM provider-independent through a robust abstraction layer.

### Provider Interface

```go
// Enhanced provider interface with function calling support
type Provider interface {
    // Existing methods
    GenerateResponse(ctx context.Context, prompt string) (string, error)
    Name() string
    GetSupportedModels() []string
    
    // New methods for function calling
    SupportsFunctionCalling() bool
    RegisterFunction(def FunctionDefinition) error
    GenerateResponseWithFunctions(ctx context.Context, prompt string, 
        functionExecutor FunctionExecutorFunc) (string, error)
}
```

### Function Calling Abstraction

```go
// Provider-agnostic function definitions and calls
type FunctionDefinition struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"`
}

type FunctionCall struct {
    Name       string          `json:"name"`
    Parameters json.RawMessage `json:"parameters"`
}

type FunctionResponse struct {
    Name     string      `json:"name"`
    Content  interface{} `json:"content"`
    Error    string      `json:"error,omitempty"`
}

// Function executor type
type FunctionExecutorFunc func(context.Context, FunctionCall) (FunctionResponse, error)
```

### Provider Adapters

Each LLM provider has an adapter that translates between our internal function calling format and the provider's native format:

- **Anthropic Adapter**: For Claude models
- **OpenAI Adapter**: For OpenAI models 
- **Google Adapter**: For Gemini models
- **xAI Adapter**: For Grok models

The adapters handle:
1. Converting our function definitions to provider-specific formats
2. Translating function calls from provider-specific formats to our internal format
3. Managing provider-specific error handling and retry logic

## Tool System

### Tool Interface

```go
// Tool defines the interface for all tools
type Tool interface {
    // Name returns the unique identifier for this tool
    Name() string
    
    // Description returns the tool's description
    Description() string
    
    // ParameterSchema returns the JSON schema for the tool's parameters
    ParameterSchema() map[string]interface{}
    
    // GetPermissionLevel returns the permission level required to use this tool
    GetPermissionLevel() PermissionLevel
    
    // Execute runs the tool with the given parameters
    Execute(ctx context.Context, params json.RawMessage) (interface{}, error)
    
    // ToFunctionDefinition converts the tool to a function definition
    ToFunctionDefinition() FunctionDefinition
}
```

### Tool Registry

The Tool Registry is responsible for:
1. Registering tools and maintaining a catalog
2. Providing tools to the function calling system
3. Validating tool parameters before execution
4. Enforcing permission checks through the security layer

### Key Tools

1. **File System Tools**
   - `LS`: List directory contents
   - `Grep`: Search file contents with regex
   - `Glob`: Find files by pattern
   - `Read`: Read file contents
   - `Edit`: Make targeted edits to files
   - `Write`: Create or overwrite files

2. **Execution Tools**
   - `Bash`: Execute shell commands in a sandboxed environment
   - `NotebookEdit`: Jupyter notebook specific operations

3. **Utility Tools**
   - `Task`: Run sub-agents for complex tasks
   - `Batch`: Execute multiple tools in parallel
   - `TodoRead`/`TodoWrite`: Manage task lists

4. **Network Tools**
   - `WebFetch`: Retrieve and process web content

## Security Model

### Permission Levels

```go
type PermissionLevel int

const (
    PermissionReadOnly PermissionLevel = iota
    PermissionShellExec
    PermissionFileWrite
    PermissionNetwork
)
```

### Permission Management

The Permission Manager:
1. Tracks allowed tools and operations
2. Prompts user for elevated permissions
3. Remembers permissions granted for a session or permanently
4. Validates file paths and command safety

### Sandboxed Execution

For shell commands:
1. Command whitelisting/blacklisting
2. Path restriction
3. Timeout enforcement
4. Resource limiting
5. Output size limiting

## Enhanced File Operations

The file operations system provides:
1. Safe file finding with pattern matching
2. Content reading with size limits
3. Secure editing with validation
4. Path verification and normalization
5. File metadata extraction
6. Directory operations

## Context Management

The context store provides:
1. Hierarchical storage (session, project, user)
2. Persistence to disk for project and user contexts
3. Serialization/deserialization
4. Support for structured data (like todos)

## Implementation Roadmap

### Phase 1: Foundation (Months 1-2)
- Implement provider abstraction layer
- Create basic tool framework
- Build simple security model
- Implement core file operations

### Phase 2: Core Tools (Months 2-3)
- Implement read-only tools (LS, Grep, Glob, Read)
- Build file editing tools (Edit, Write)
- Create execution tool (Bash)
- Implement batch execution

### Phase 3: Advanced Features (Months 3-4)
- Implement context management
- Build Todo management
- Create Task tool for sub-agents
- Enhance security model with sandboxing

### Phase 4: Integration (Months 4-5)
- Connect with existing pipeline architecture
- Integrate with command system
- Update TUI for interactive use
- Add streaming response support

### Phase 5: Polish (Months 5-6)
- Add advanced tool features
- Implement advanced provider-specific optimizations
- Implement slash commands
- Add comprehensive error handling
- Performance optimization

## Provider-Specific Considerations

### Anthropic
- Supports function calling in newer Claude models
- Particularly strong at code understanding
- Streaming capability

### OpenAI
- Well-established function calling API
- Different token limits and pricing
- May require different prompt engineering

### Google
- Function calling support in Gemini models
- Different strengths in code generation

### xAI
- Grok models with growing capabilities
- May require adapter updates as API evolves

## Getting Started for Developers

To begin implementing this system:
1. Start with the provider abstraction layer
2. Implement the tool interface and basic tools
3. Create the security model
4. Build file operations enhancements
5. Implement context storage

## Conclusion

This design allows for a powerful, extensible system that can operate with multiple LLM providers while providing the core capabilities of an AI code assistant. By following the provider-agnostic approach, the system remains flexible and future-proof as AI technology evolves.