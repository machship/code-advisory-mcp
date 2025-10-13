package embedder

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
	"github.com/philippgille/chromem-go"
)

// Backend specifies which embedding backend to use.
type Backend string

const (
	BackendAzureOpenAI Backend = "azure"
	BackendOllama      Backend = "ollama"
)

// Embedder wraps embedding clients for different backends.
type Embedder struct {
	backend    Backend
	client     *openai.Client
	ollamaFunc chromem.EmbeddingFunc
	model      string
	ollamaURL  string
}

// Config holds configuration for creating an embedder.
type Config struct {
	Backend       Backend
	Endpoint      string // Azure OpenAI endpoint
	APIKey        string // Azure OpenAI API key
	Model         string
	OllamaBaseURL string // Optional, defaults to http://localhost:11434
}

// New creates a new Embedder instance based on the backend type.
func New(cfg Config) (*Embedder, error) {
	switch cfg.Backend {
	case BackendAzureOpenAI:
		return newAzureOpenAI(cfg)
	case BackendOllama:
		return newOllama(cfg)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", cfg.Backend)
	}
}

// newAzureOpenAI creates an Azure OpenAI embedder.
func newAzureOpenAI(cfg Config) (*Embedder, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("azure OpenAI endpoint is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("azure OpenAI API key is required")
	}

	const apiVersion = "2024-02-01"
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		azure.WithEndpoint(cfg.Endpoint, apiVersion),
	)

	return &Embedder{
		backend: BackendAzureOpenAI,
		client:  &client,
		model:   cfg.Model,
	}, nil
}

// newOllama creates an Ollama embedder.
func newOllama(cfg Config) (*Embedder, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for Ollama")
	}

	baseURL := cfg.OllamaBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434/api"
	}

	// Create chromem-go's Ollama embedding function
	embeddingFunc := chromem.NewEmbeddingFuncOllama(cfg.Model, baseURL)

	return &Embedder{
		backend:    BackendOllama,
		ollamaFunc: embeddingFunc,
		model:      cfg.Model,
		ollamaURL:  baseURL,
	}, nil
}

// Embed generates embeddings for the given text.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	switch e.backend {
	case BackendAzureOpenAI:
		return e.embedAzureOpenAI(ctx, text)
	case BackendOllama:
		return e.embedOllama(ctx, text)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", e.backend)
	}
}

// embedAzureOpenAI generates embeddings using Azure OpenAI.
func (e *Embedder) embedAzureOpenAI(ctx context.Context, text string) ([]float32, error) {
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

// embedOllama generates embeddings using Ollama.
func (e *Embedder) embedOllama(ctx context.Context, text string) ([]float32, error) {
	return e.ollamaFunc(ctx, text)
}
