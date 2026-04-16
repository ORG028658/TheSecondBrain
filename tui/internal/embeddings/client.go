package embeddings

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	model  string
}

// New creates an embedding client pointed at the Rakuten AI Gateway.
// It uses the OpenAI-compatible endpoint with your Rakuten API key.
func New(model, baseURL string) *Client {
	cfg := openai.DefaultConfig(os.Getenv("LLM_COMPATIBLE_API_KEY"))
	cfg.BaseURL = baseURL
	return &Client{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

// Embed returns embedding vectors for a batch of texts.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(c.model),
	})
	if err != nil {
		return nil, fmt.Errorf("embedding request: %w", err)
	}

	result := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		result[i] = d.Embedding
	}
	return result, nil
}

// EmbedOne is a convenience wrapper for a single text.
func (c *Client) EmbedOne(ctx context.Context, text string) ([]float32, error) {
	vecs, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return vecs[0], nil
}
