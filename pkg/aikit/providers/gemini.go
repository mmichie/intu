package providers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// SupportedGeminiModels is a list of supported Gemini models
var SupportedGeminiModels = map[string]bool{
	"gemini-1.5-pro":           true,
	"gemini-2.5-pro-exp-03-25": true,
	"gemini-1.5-flash":         true,
	"gemini-1.0-pro":           true,
}

type GeminiAIProvider struct {
	BaseProvider
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiAIProvider() (*GeminiAIProvider, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	provider := &GeminiAIProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
		},
		client: client,
	}

	// Get model from environment or use default, and validate it
	modelFromEnv := provider.GetEnvOrDefault("GEMINI_MODEL", "gemini-1.5-pro")
	provider.SetModel(modelFromEnv, SupportedGeminiModels, "gemini-1.5-pro")

	provider.model = client.GenerativeModel(provider.Model)

	return provider, nil
}

func (p *GeminiAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	resp, err := p.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("error generating content: %w", err)
	}

	var fullResponse strings.Builder
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			fullResponse.WriteString(fmt.Sprintf("%v", part))
		}
	}

	return strings.TrimSpace(fullResponse.String()), nil
}

func (p *GeminiAIProvider) Name() string {
	return "gemini"
}

// GetSupportedModels returns a list of supported models for this provider
func (p *GeminiAIProvider) GetSupportedModels() []string {
	models := make([]string, 0, len(SupportedGeminiModels))
	for model := range SupportedGeminiModels {
		models = append(models, model)
	}
	return models
}
