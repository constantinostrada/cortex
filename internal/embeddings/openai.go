package embeddings

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

const (
	defaultModel      = "text-embedding-3-small"
	defaultDimensions = 1536
)

// OpenAI implements the Provider interface using OpenAI's API
type OpenAI struct {
	client     *openai.Client
	model      string
	dimensions int
}

// NewOpenAI creates a new OpenAI embedding provider
func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		client:     openai.NewClient(apiKey),
		model:      defaultModel,
		dimensions: defaultDimensions,
	}
}

// NewOpenAIWithModel creates a new OpenAI provider with a custom model
func NewOpenAIWithModel(apiKey, model string, dimensions int) *OpenAI {
	return &OpenAI{
		client:     openai.NewClient(apiKey),
		model:      model,
		dimensions: dimensions,
	}
}

// Embed generates an embedding for a single text
func (o *OpenAI) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := o.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(o.model),
	})
	if err != nil {
		return nil, fmt.Errorf("openai embedding error: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return resp.Data[0].Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (o *OpenAI) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	resp, err := o.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(o.model),
	})
	if err != nil {
		return nil, fmt.Errorf("openai batch embedding error: %w", err)
	}

	embeddings := make([][]float32, len(texts))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// Model returns the model name
func (o *OpenAI) Model() string {
	return o.model
}

// Dimensions returns the embedding dimensions
func (o *OpenAI) Dimensions() int {
	return o.dimensions
}
