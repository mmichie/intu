# Pipeline Configuration System

The v2 pipeline package includes a comprehensive configuration system that allows you to define, save, and reuse complex AI pipeline configurations.

## Overview

Pipeline configurations are stored as JSON files and support all pipeline types:
- Simple (single provider)
- Serial (sequential processing)
- Parallel (concurrent processing with combiners)
- Nested (multi-stage pipelines)
- Transform (with input/output transformations)
- High Availability (automatic failover)
- Consensus (AI-judged consensus)

## Configuration Structure

### Basic Configuration

```json
{
  "name": "my-pipeline",
  "description": "Pipeline description",
  "type": "simple|serial|parallel|nested|transform|high_availability|consensus",
  "version": "1.0",
  
  // Type-specific fields...
  
  "provider_configs": {
    "provider-name": {
      "api_key": "${ENV_VAR}",
      "model": "model-name",
      "temperature": 0.7
    }
  },
  
  "options": {
    "retries": 3,
    "timeout": 30,
    "cache": 300
  }
}
```

### Pipeline Types

#### Simple Pipeline
```json
{
  "type": "simple",
  "provider": "openai"
}
```

#### Serial Pipeline
```json
{
  "type": "serial",
  "providers": ["openai", "claude", "gemini"]
}
```

#### Parallel Pipeline
```json
{
  "type": "parallel",
  "providers": ["openai", "claude"],
  "combiner": "concat|majority_vote|longest|quality_score|best_picker|consensus",
  "combiner_config": {
    "separator": "\n",           // for concat
    "max_tokens": 100,          // for quality_score
    "picker_provider": "gpt-4", // for best_picker
    "judge_provider": "gpt-4"   // for consensus
  }
}
```

#### Nested Pipeline
```json
{
  "type": "nested",
  "stages": [
    {
      "name": "stage1",
      "type": "simple",
      "config": {
        "provider": "openai"
      }
    },
    {
      "name": "stage2",
      "type": "parallel",
      "config": {
        "providers": ["claude", "gemini"],
        "combiner": "concat"
      }
    }
  ]
}
```

#### High Availability Pipeline
```json
{
  "type": "high_availability",
  "providers": ["primary", "backup1", "backup2", "backup3"]
}
```

#### Consensus Pipeline
```json
{
  "type": "consensus",
  "providers": ["openai", "claude", "gemini"],
  "combiner_config": {
    "judge_provider": "gpt-4"
  }
}
```

## Usage

### Programmatic Usage

```go
import "github.com/mmichie/intu/pkg/aikit/v2/pipeline"

// Initialize the config factory
factory := pipeline.GetConfigFactory()

// Load and use a saved configuration
pipeline, err := factory.CreateFromConfig("my-pipeline")
if err != nil {
    log.Fatal(err)
}

result, err := pipeline.Execute(ctx, "Your prompt here")
```

### Configuration Management

```go
// Save a new configuration
config := &pipeline.PipelineConfig{
    Name:     "my-config",
    Type:     pipeline.PipelineTypeSimple,
    Provider: "openai",
}
err := pipeline.SavePipelineConfig(config)

// Load a configuration
loaded, err := pipeline.LoadPipelineConfig("my-config")

// List all configurations
configs := pipeline.ListPipelineConfigs()

// Delete a configuration
err = pipeline.DeletePipelineConfig("my-config")

// Export configuration to JSON
data, err := factory.ExportConfig("my-config")

// Import configuration from JSON
err = factory.ImportConfig(data)
```

### Using the ConfigStore

```go
// Create a custom config store
store := pipeline.NewConfigStore("/path/to/configs.json")
err := store.Load()

// Filter configurations
simpleConfigs := store.FilterByType(pipeline.PipelineTypeSimple)
openaiConfigs := store.FilterByProvider("openai")

// Backup and restore
err = store.Backup()
err = store.Restore("/path/to/backup.json")
```

## Combiner Types

### Concat Combiner
Concatenates all responses with a separator.

### Majority Vote Combiner
Returns the most common response among providers.

### Longest Response Combiner
Returns the longest response.

### Quality Score Combiner
Scores responses based on length, punctuation, and structure.

### Best Picker Combiner
Uses an AI provider to select the best response.

### Consensus Combiner
Uses an AI provider to synthesize a consensus from all responses.

## Environment Variables

Provider configurations can reference environment variables using the `${VAR_NAME}` syntax:

```json
{
  "provider_configs": {
    "openai": {
      "api_key": "${OPENAI_API_KEY}"
    }
  }
}
```

## Examples

See the `examples/` directory for complete configuration examples:
- `simple_pipeline.json` - Basic single-provider pipeline
- `serial_refinement.json` - Sequential processing for iterative refinement
- `parallel_consensus.json` - Multiple providers with consensus
- `best_picker.json` - AI-selected best response
- `high_availability.json` - Automatic failover pipeline
- `nested_analysis.json` - Complex multi-stage analysis

## Default Configuration Location

By default, pipeline configurations are stored in:
```
~/.intu/pipelines.json
```

## Best Practices

1. **Use environment variables** for API keys and sensitive data
2. **Version your configurations** to track changes
3. **Add descriptions** to document pipeline purposes
4. **Set appropriate timeouts** based on provider response times
5. **Use caching** for expensive operations
6. **Test configurations** before production use
7. **Backup configurations** regularly

## Migration from v1

To migrate v1 pipeline configurations:

1. Load v1 configuration
2. Map to v2 structure:
   - `Type` remains the same
   - `Providers` array for multi-provider pipelines
   - `Provider` string for simple pipelines
   - Combiner configuration in `combiner_config`
3. Save using v2 ConfigStore

## Error Handling

The configuration system validates:
- Required fields based on pipeline type
- Provider availability
- Combiner compatibility
- Nested configuration validity

Invalid configurations will return detailed validation errors.