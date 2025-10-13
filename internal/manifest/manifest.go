package manifest

import (
	"encoding/json"
	"os"
)

// Document represents a single document in the manifest.
type Document struct {
	Path     string            `json:"path"`
	URI      string            `json:"uri"`
	Metadata map[string]string `json:"metadata"`
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
