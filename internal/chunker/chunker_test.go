package chunker

import (
	"testing"
)

func TestChunkDocument(t *testing.T) {
	tests := map[string]struct {
		content    string
		baseURL    string
		sourcePath string
		metadata   map[string]string
		wantChunks int
		checkFirst func(*testing.T, Chunk)
	}{
		"basic H2 and H3": {
			content: `# Title

## Introduction

Some intro text about errors.

### Error Wrapping

Use fmt.Errorf with %w:

    if err != nil {
        return fmt.Errorf("failed: %w", err)
    }

### Error Checking

Use errors.Is and errors.As.

## Best Practices

Some best practices intro.

### Testing

Write tests for errors.
`,
			baseURL:    "https://example.com/doc",
			sourcePath: "/path/to/doc.md",
			metadata: map[string]string{
				"language": "Go",
			},
			wantChunks: 3,
			checkFirst: func(t *testing.T, c Chunk) {
				if c.H2Heading != "Introduction" {
					t.Errorf("H2Heading = %q, want %q", c.H2Heading, "Introduction")
				}
				if c.H3Heading != "Error Wrapping" {
					t.Errorf("H3Heading = %q, want %q", c.H3Heading, "Error Wrapping")
				}
				if c.URL != "https://example.com/doc#error-wrapping" {
					t.Errorf("URL = %q, want %q", c.URL, "https://example.com/doc#error-wrapping")
				}
				if c.SourceFile != "/path/to/doc.md" {
					t.Errorf("SourceFile = %q, want %q", c.SourceFile, "/path/to/doc.md")
				}
				// Check that content includes H2 context
				if c.Content[:len("## Introduction")] != "## Introduction" {
					t.Errorf("Content should start with H2 heading")
				}
				// Check metadata
				if c.Metadata["language"] != "Go" {
					t.Errorf("Metadata language = %q, want %q", c.Metadata["language"], "Go")
				}
			},
		},
		"H3 before H2 should be skipped": {
			content: `# Title

### Orphan H3

This should be skipped.

## Section

Some text.

### Valid H3

This should be included.
`,
			baseURL:    "https://example.com/doc",
			sourcePath: "/path/to/doc.md",
			metadata:   nil,
			wantChunks: 1,
			checkFirst: func(t *testing.T, c Chunk) {
				if c.H3Heading != "Valid H3" {
					t.Errorf("H3Heading = %q, want %q", c.H3Heading, "Valid H3")
				}
			},
		},
		"no H2 or H3": {
			content: `# Title

Just some preamble text.

More text here.
`,
			baseURL:    "https://example.com/doc",
			sourcePath: "/path/to/doc.md",
			metadata:   nil,
			wantChunks: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			chunks, err := ChunkDocument([]byte(tc.content), tc.baseURL, tc.sourcePath, tc.metadata)
			if err != nil {
				t.Fatalf("ChunkDocument failed: %v", err)
			}

			if len(chunks) != tc.wantChunks {
				t.Errorf("got %d chunks, want %d", len(chunks), tc.wantChunks)
			}

			if tc.wantChunks > 0 && tc.checkFirst != nil {
				tc.checkFirst(t, chunks[0])
			}

			// Verify all chunks have unique IDs
			ids := make(map[string]bool)
			for _, chunk := range chunks {
				if ids[chunk.ID] {
					t.Errorf("duplicate chunk ID: %s", chunk.ID)
				}
				ids[chunk.ID] = true

				// Verify ID is valid SHA1 (40 hex chars)
				if len(chunk.ID) != 40 {
					t.Errorf("chunk ID length = %d, want 40 (SHA1 hex)", len(chunk.ID))
				}
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"simple":           {input: "Error Wrapping", want: "error-wrapping"},
		"special chars":    {input: "Error: Handling?", want: "error-handling"},
		"underscores":      {input: "Test_Function_Name", want: "test-function-name"},
		"multiple spaces":  {input: "Multiple   Spaces", want: "multiple-spaces"},
		"leading trailing": {input: "  Trim Me  ", want: "trim-me"},
		"multiple hyphens": {input: "Already--Has---Hyphens", want: "already-has-hyphens"},
		"mixed":            {input: "Go 1.21+ Features!", want: "go-121-features"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := slugify(tc.input)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestConstructURL(t *testing.T) {
	tests := map[string]struct {
		baseURL string
		heading string
		want    string
	}{
		"basic":         {baseURL: "https://example.com/doc", heading: "Error Wrapping", want: "https://example.com/doc#error-wrapping"},
		"empty heading": {baseURL: "https://example.com/doc", heading: "", want: "https://example.com/doc"},
		"complex":       {baseURL: "https://go.dev/doc/effective_go", heading: "Package Names", want: "https://go.dev/doc/effective_go#package-names"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := constructURL(tc.baseURL, tc.heading)
			if got != tc.want {
				t.Errorf("constructURL(%q, %q) = %q, want %q", tc.baseURL, tc.heading, got, tc.want)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	content1 := "Test content 1"
	content2 := "Test content 2"

	id1 := generateID(content1)
	id2 := generateID(content2)
	id1Again := generateID(content1)

	// Same content should produce same ID
	if id1 != id1Again {
		t.Errorf("same content produced different IDs: %s vs %s", id1, id1Again)
	}

	// Different content should produce different IDs
	if id1 == id2 {
		t.Errorf("different content produced same ID: %s", id1)
	}

	// ID should be 40 characters (SHA1 hex)
	if len(id1) != 40 {
		t.Errorf("ID length = %d, want 40", len(id1))
	}
}

func TestCopyMetadata(t *testing.T) {
	original := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	copy := copyMetadata(original)

	// Check values match
	for k, v := range original {
		if copy[k] != v {
			t.Errorf("copy[%q] = %q, want %q", k, copy[k], v)
		}
	}

	// Modify copy and ensure original is unchanged
	copy["key1"] = "modified"
	if original["key1"] != "value1" {
		t.Errorf("modifying copy affected original")
	}

	// Modify original and ensure copy is unchanged
	original["key3"] = "new"
	if _, ok := copy["key3"]; ok {
		t.Errorf("modifying original affected copy")
	}
}
