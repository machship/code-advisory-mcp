package manifest

import (
	"encoding/json"
	"os"
)

// ChunkingStrategy defines how to chunk a markdown document.
type ChunkingStrategy string

const (
	// ChunkingStrategyH2H3 chunks by H2 context + H3 sections (default).
	ChunkingStrategyH2H3 ChunkingStrategy = "h2_h3"

	// ChunkingStrategyH2 chunks by H2 sections only.
	ChunkingStrategyH2 ChunkingStrategy = "h2"
)

// Document represents a single document in the manifest.
type Document struct {
	Path             string            `json:"path"`
	URI              string            `json:"uri"`
	Metadata         map[string]string `json:"metadata"`
	ChunkingStrategy ChunkingStrategy  `json:"chunking_strategy,omitempty"`
}

// GetChunkingStrategy returns the chunking strategy for the document,
// defaulting to h2_h3 if not specified.
func (d *Document) GetChunkingStrategy() ChunkingStrategy {
	if d.ChunkingStrategy == "" {
		return ChunkingStrategyH2H3
	}
	return d.ChunkingStrategy
}

// Manifest represents the JSON manifest file structure.
type Manifest struct {
	Documents []Document `json:"documents"`
}

// Load reads and parses the manifest file from the given path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return &m, nil
}
