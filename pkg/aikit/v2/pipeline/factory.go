package pipeline

import (
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/v2/agent"
	"github.com/mmichie/intu/pkg/aikit/v2/config"
	aierrors "github.com/mmichie/intu/pkg/aikit/v2/errors"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// Factory provides convenient methods for creating pipelines
type Factory struct {
	registry *provider.Registry
	config   config.Config
}

// NewFactory creates a new pipeline factory
func NewFactory() *Factory {
	return &Factory{
		registry: provider.NewRegistry(),
	}
}

// NewFactoryWithRegistry creates a factory with a custom provider registry
func NewFactoryWithRegistry(registry *provider.Registry) *Factory {
	return &Factory{
		registry: registry,
	}
}

// WithConfig sets the default configuration for creating providers
func (f *Factory) WithConfig(cfg config.Config) *Factory {
	f.config = cfg
	return f
}

// CreateSimple creates a simple pipeline with a single provider
func (f *Factory) CreateSimple(providerName string, opts ...Option) (Pipeline, error) {
	p, err := f.createProvider(providerName)
	if err != nil {
		return nil, err
	}
	return NewSimplePipeline(p, opts...), nil
}

// CreateSerial creates a serial pipeline that processes through providers sequentially
func (f *Factory) CreateSerial(providerNames []string, opts ...Option) (Pipeline, error) {
	providers, err := f.createProviders(providerNames)
	if err != nil {
		return nil, err
	}
	return NewSerialPipeline(providers, opts...), nil
}

// CreateParallel creates a parallel pipeline with the specified combiner
func (f *Factory) CreateParallel(providerNames []string, combiner ResultCombiner, opts ...Option) (Pipeline, error) {
	providers, err := f.createProviders(providerNames)
	if err != nil {
		return nil, err
	}
	return NewParallelPipeline(providers, combiner, opts...), nil
}

// CreateParallelWithBestPicker creates a parallel pipeline that picks the best result
func (f *Factory) CreateParallelWithBestPicker(providerNames []string, pickerName string, opts ...Option) (Pipeline, error) {
	providers, err := f.createProviders(providerNames)
	if err != nil {
		return nil, err
	}

	picker, err := f.createProvider(pickerName)
	if err != nil {
		return nil, err
	}

	combiner := NewBestPickerCombiner(picker)
	return NewParallelPipeline(providers, combiner, opts...), nil
}

// CreateCollaborative creates a collaborative pipeline with the specified rounds
func (f *Factory) CreateCollaborative(providerNames []string, rounds int, opts ...Option) (Pipeline, error) {
	providers, err := f.createProviders(providerNames)
	if err != nil {
		return nil, err
	}
	return NewCollaborativePipeline(providers, rounds, opts...), nil
}

// CreateWithAgents creates a pipeline from pre-configured agents
func (f *Factory) CreateWithAgents(agents []*agent.Agent, opts ...Option) Pipeline {
	providers := make([]provider.Provider, len(agents))
	for i, a := range agents {
		providers[i] = a.Provider()
	}
	return NewSerialPipeline(providers, opts...)
}

// CreateFallback creates a pipeline with automatic fallback providers
func (f *Factory) CreateFallback(primaryName string, fallbackNames []string, opts ...Option) (Pipeline, error) {
	primary, err := f.createProvider(primaryName)
	if err != nil {
		return nil, err
	}

	// Create the primary pipeline
	pipeline := NewSimplePipeline(primary, opts...)

	// Add fallbacks in reverse order (last fallback first)
	for i := len(fallbackNames) - 1; i >= 0; i-- {
		fallback, err := f.createProvider(fallbackNames[i])
		if err != nil {
			return nil, err
		}

		// Create a new pipeline with this fallback
		pipeline = pipeline.WithOptions(WithFallback(fallback)).(*SimplePipeline)
	}

	return pipeline, nil
}

// CreateBalanced creates a load-balanced pipeline across multiple instances
func (f *Factory) CreateBalanced(providerName string, instances int, opts ...Option) (Pipeline, error) {
	providers := make([]provider.Provider, instances)
	for i := 0; i < instances; i++ {
		p, err := f.createProvider(providerName)
		if err != nil {
			return nil, err
		}
		providers[i] = p
	}

	// Use round-robin combiner for load balancing
	combiner := NewRoundRobinCombiner()
	return NewParallelPipeline(providers, combiner, opts...), nil
}

// CreateNested creates a nested pipeline from existing pipelines
func (f *Factory) CreateNested(stages []Pipeline, opts ...Option) (Pipeline, error) {
	if len(stages) == 0 {
		return nil, fmt.Errorf("at least one stage required for nested pipeline")
	}
	return NewNestedPipeline(stages, opts...), nil
}

// CreateTransform creates a pipeline with input/output transformations
func (f *Factory) CreateTransform(pipeline Pipeline, inputTransform, outputTransform ProcessFunc, opts ...Option) Pipeline {
	return NewTransformAdapter(pipeline, inputTransform, outputTransform, opts...)
}

// CreateFunction creates a pipeline from a processing function
func (f *Factory) CreateFunction(name string, fn ProcessFunc, opts ...Option) Pipeline {
	return NewFunctionAdapter(name, fn, opts...)
}

// CreateChain creates a chain of transformations
func (f *Factory) CreateChain(providerName string, transforms ...ProcessFunc) (Pipeline, error) {
	// Create the base provider pipeline
	basePipeline, err := f.CreateSimple(providerName)
	if err != nil {
		return nil, err
	}

	// If no transforms, return the base pipeline
	if len(transforms) == 0 {
		return basePipeline, nil
	}

	// Build a nested pipeline with transform adapters
	stages := []Pipeline{basePipeline}

	for i, transform := range transforms {
		name := fmt.Sprintf("transform_%d", i)
		adapter := NewFunctionAdapter(name, transform)
		stages = append(stages, adapter)
	}

	return NewNestedPipeline(stages), nil
}

// Helper methods

func (f *Factory) createProvider(name string) (provider.Provider, error) {
	if f.registry != nil {
		return f.registry.CreateProvider(name, f.config)
	}
	return provider.Create(name, f.config)
}

func (f *Factory) createProviders(names []string) ([]provider.Provider, error) {
	providers := make([]provider.Provider, len(names))
	for i, name := range names {
		p, err := f.createProvider(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", name, err)
		}
		providers[i] = p
	}
	return providers, nil
}

// Builder provides a fluent interface for building pipelines
type Builder struct {
	factory    *Factory
	providers  []provider.Provider
	transforms []ProcessFunc
	options    []Option
	err        error
}

// NewBuilder creates a new pipeline builder
func NewBuilder() *Builder {
	return &Builder{
		factory: NewFactory(),
	}
}

// WithFactory sets the factory to use
func (b *Builder) WithFactory(f *Factory) *Builder {
	b.factory = f
	return b
}

// WithProvider adds a provider by name
func (b *Builder) WithProvider(name string) *Builder {
	if b.err != nil {
		return b
	}

	p, err := b.factory.createProvider(name)
	if err != nil {
		b.err = err
		return b
	}

	b.providers = append(b.providers, p)
	return b
}

// WithProviders adds multiple providers
func (b *Builder) WithProviders(names ...string) *Builder {
	for _, name := range names {
		b.WithProvider(name)
	}
	return b
}

// WithOptions adds pipeline options
func (b *Builder) WithOptions(opts ...Option) *Builder {
	b.options = append(b.options, opts...)
	return b
}

// BuildSimple builds a simple pipeline with the first provider
func (b *Builder) BuildSimple() (Pipeline, error) {
	if b.err != nil {
		return nil, b.err
	}

	if len(b.providers) == 0 {
		return nil, aierrors.New("factory", "build",
			fmt.Errorf("no providers configured"))
	}

	return NewSimplePipeline(b.providers[0], b.options...), nil
}

// BuildSerial builds a serial pipeline
func (b *Builder) BuildSerial() (Pipeline, error) {
	if b.err != nil {
		return nil, b.err
	}

	if len(b.providers) == 0 {
		return nil, aierrors.New("factory", "build",
			fmt.Errorf("no providers configured"))
	}

	return NewSerialPipeline(b.providers, b.options...), nil
}

// BuildParallel builds a parallel pipeline with the given combiner
func (b *Builder) BuildParallel(combiner ResultCombiner) (Pipeline, error) {
	if b.err != nil {
		return nil, b.err
	}

	if len(b.providers) == 0 {
		return nil, aierrors.New("factory", "build",
			fmt.Errorf("no providers configured"))
	}

	return NewParallelPipeline(b.providers, combiner, b.options...), nil
}

// BuildCollaborative builds a collaborative pipeline
func (b *Builder) BuildCollaborative(rounds int) (Pipeline, error) {
	if b.err != nil {
		return nil, b.err
	}

	if len(b.providers) == 0 {
		return nil, aierrors.New("factory", "build",
			fmt.Errorf("no providers configured"))
	}

	return NewCollaborativePipeline(b.providers, rounds, b.options...), nil
}

// WithTransform adds a transformation stage
func (b *Builder) WithTransform(transform ProcessFunc) *Builder {
	// This will be used in BuildChain
	if b.transforms == nil {
		b.transforms = []ProcessFunc{}
	}
	b.transforms = append(b.transforms, transform)
	return b
}

// BuildChain builds a pipeline with transformations
func (b *Builder) BuildChain() (Pipeline, error) {
	if b.err != nil {
		return nil, b.err
	}

	if len(b.providers) == 0 {
		return nil, aierrors.New("factory", "build",
			fmt.Errorf("no providers configured"))
	}

	// Start with the first provider or serial pipeline
	var pipeline Pipeline
	if len(b.providers) == 1 {
		pipeline = NewSimplePipeline(b.providers[0], b.options...)
	} else {
		pipeline = NewSerialPipeline(b.providers, b.options...)
	}

	// Apply transforms if any
	if len(b.transforms) > 0 {
		stages := []Pipeline{pipeline}
		for i, transform := range b.transforms {
			name := fmt.Sprintf("transform_%d", i)
			adapter := NewFunctionAdapter(name, transform)
			stages = append(stages, adapter)
		}
		pipeline = NewNestedPipeline(stages, b.options...)
	}

	return pipeline, nil
}

// Preset pipeline configurations

// CreateDefaultPipeline creates a standard pipeline with fallback
func CreateDefaultPipeline(cfg config.Config) (Pipeline, error) {
	factory := NewFactory().WithConfig(cfg)

	// Try to create with the default provider
	defaultProvider, err := provider.GetDefault()
	if err != nil {
		return nil, err
	}

	primaryName := defaultProvider.Name()

	// Define common fallback chain
	fallbacks := []string{}
	allProviders := provider.List()

	// Add other providers as fallbacks
	for _, name := range allProviders {
		if name != primaryName {
			fallbacks = append(fallbacks, name)
		}
	}

	return factory.CreateFallback(primaryName, fallbacks, WithRetries(2))
}

// CreateHighAvailabilityPipeline creates a pipeline optimized for reliability
func CreateHighAvailabilityPipeline(cfg config.Config, providers []string) (Pipeline, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one provider required")
	}

	factory := NewFactory().WithConfig(cfg)

	// Create primary with fallbacks
	primary := providers[0]
	fallbacks := providers[1:]

	return factory.CreateFallback(primary, fallbacks,
		WithRetries(3),
		WithCache(300), // 5 minute cache
	)
}

// CreateConsensuspipeline creates a pipeline that seeks consensus among providers
func CreateConsensusPipeline(cfg config.Config, providers []string, consensusProvider string) (Pipeline, error) {
	factory := NewFactory().WithConfig(cfg)

	// Create providers
	providerInstances, err := factory.createProviders(providers)
	if err != nil {
		return nil, err
	}

	// Create consensus evaluator
	evaluator, err := factory.createProvider(consensusProvider)
	if err != nil {
		return nil, err
	}

	// Use a consensus combiner
	combiner := NewConsensusCombiner(evaluator)

	return NewParallelPipeline(providerInstances, combiner), nil
}

// Quick creation functions

// Simple creates a simple pipeline with a single provider
func Simple(providerName string, cfg config.Config) (Pipeline, error) {
	return NewFactory().WithConfig(cfg).CreateSimple(providerName)
}

// Serial creates a serial pipeline
func Serial(providerNames []string, cfg config.Config) (Pipeline, error) {
	return NewFactory().WithConfig(cfg).CreateSerial(providerNames)
}

// Parallel creates a parallel pipeline with majority voting
func Parallel(providerNames []string, cfg config.Config) (Pipeline, error) {
	factory := NewFactory().WithConfig(cfg)
	providers, err := factory.createProviders(providerNames)
	if err != nil {
		return nil, err
	}
	return NewParallelPipeline(providers, NewMajorityVoteCombiner()), nil
}

// Collaborative creates a collaborative pipeline
func Collaborative(providerNames []string, rounds int, cfg config.Config) (Pipeline, error) {
	return NewFactory().WithConfig(cfg).CreateCollaborative(providerNames, rounds)
}
