// Package embeddings provides text embedding services
package embeddings

import "context"

// Provider defines the interface for embedding generation
type Provider interface {
	// Embed generates an embedding vector for the given text
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Model returns the model name being used
	Model() string

	// Dimensions returns the embedding vector dimensions
	Dimensions() int
}
