package embedder

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
)

// Embedder wraps the Azure OpenAI embedding client.
type Embedder struct {
	client *openai.Client
	model  string
}

// New creates a new Embedder instance.
func New(endpoint, apiKey, model string) (*Embedder, error) {
	const apiVersion = "2024-02-01"
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		azure.WithEndpoint(endpoint, apiVersion),
	)

	return &Embedder{
		client: &client,
		model:  model,
	}, nil
}

// Embed generates embeddings for the given text.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := e.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(e.model),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert []float64 to []float32
	embedding64 := resp.Data[0].Embedding
	embedding32 := make([]float32, len(embedding64))
	for i, v := range embedding64 {
		embedding32[i] = float32(v)
	}

	return embedding32, nil
}
