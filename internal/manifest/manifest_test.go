package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := map[string]struct {
		content     string
		wantErr     bool
		wantDocs    int
		checkFirst  func(*testing.T, Document)
	}{
		"valid manifest": {
			content: `{
  "documents": [
    {
      "path": "docs/example.md",
      "uri": "https://example.com/example",
      "metadata": {
        "language": "Go",
        "category": "best-practices"
      }
    },
    {
      "path": "docs/another.md",
      "uri": "https://example.com/another",
      "metadata": {
        "language": "Go"
      }
    }
  ]
}`,
			wantErr:  false,
			wantDocs: 2,
			checkFirst: func(t *testing.T, d Document) {
				if d.Path != "docs/example.md" {
					t.Errorf("Path = %q, want %q", d.Path, "docs/example.md")
				}
				if d.URI != "https://example.com/example" {
					t.Errorf("URI = %q, want %q", d.URI, "https://example.com/example")
				}
				if d.Metadata["language"] != "Go" {
					t.Errorf("Metadata[language] = %q, want %q", d.Metadata["language"], "Go")
				}
				if d.Metadata["category"] != "best-practices" {
					t.Errorf("Metadata[category] = %q, want %q", d.Metadata["category"], "best-practices")
				}
			},
		},
		"empty documents": {
			content: `{
  "documents": []
}`,
			wantErr:  false,
			wantDocs: 0,
		},
		"invalid json": {
			content: `{
  "documents": [
    {
      "path": "incomplete
}`,
			wantErr:  true,
			wantDocs: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "manifest.json")

			err := os.WriteFile(tmpFile, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			// Load manifest
			m, err := Load(tmpFile)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}

			if len(m.Documents) != tc.wantDocs {
				t.Errorf("got %d documents, want %d", len(m.Documents), tc.wantDocs)
			}

			if tc.wantDocs > 0 && tc.checkFirst != nil {
				tc.checkFirst(t, m.Documents[0])
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/manifest.json")
	if err == nil {
		t.Error("Load() with nonexistent file expected error, got nil")
	}
}
