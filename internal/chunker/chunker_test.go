package chunker

import (
	"strings"
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
			chunks, err := ChunkDocument([]byte(tc.content), tc.baseURL, tc.sourcePath, tc.metadata, StrategyH2H3)
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

func TestChunkH2H3H4Strategy(t *testing.T) {
	tests := map[string]struct {
		content    string
		baseURL    string
		wantChunks int
		checkFunc  func(*testing.T, []Chunk)
	}{
		"H3 without H4s creates single chunk": {
			content: `# Document Title

## Section One

This is H2 intro content.

### Subsection A

This H3 has no H4s, so it should become one chunk.

Some more content here.
`,
			baseURL:    "https://example.com/doc",
			wantChunks: 1,
			checkFunc: func(t *testing.T, chunks []Chunk) {
				if chunks[0].H2Heading != "Section One" {
					t.Errorf("H2Heading = %q, want %q", chunks[0].H2Heading, "Section One")
				}
				if chunks[0].H3Heading != "Subsection A" {
					t.Errorf("H3Heading = %q, want %q", chunks[0].H3Heading, "Subsection A")
				}
				// Should contain H2 context
				if chunks[0].Content[:len("## Section One")] != "## Section One" {
					t.Errorf("Content should start with H2 heading")
				}
			},
		},
		"H3 with H4s creates chunk per H4": {
			content: `# Document Title

## Section One

This is H2 intro content.

### Subsection B

This H3 has H4 children, so each H4 becomes a chunk.

#### First Detail

Content for first detail.

#### Second Detail

Content for second detail.
`,
			baseURL:    "https://example.com/doc",
			wantChunks: 2,
			checkFunc: func(t *testing.T, chunks []Chunk) {
				// First H4 chunk
				if chunks[0].H2Heading != "Section One" {
					t.Errorf("chunks[0] H2Heading = %q, want %q", chunks[0].H2Heading, "Section One")
				}
				if chunks[0].H3Heading != "Subsection B" {
					t.Errorf("chunks[0] H3Heading = %q, want %q", chunks[0].H3Heading, "Subsection B")
				}
				// Should contain H2 + H3 context
				if chunks[0].Content[:len("## Section One")] != "## Section One" {
					t.Errorf("chunks[0] should start with H2 heading")
				}

				// Second H4 chunk
				if chunks[1].H2Heading != "Section One" {
					t.Errorf("chunks[1] H2Heading = %q, want %q", chunks[1].H2Heading, "Section One")
				}
				if chunks[1].H3Heading != "Subsection B" {
					t.Errorf("chunks[1] H3Heading = %q, want %q", chunks[1].H3Heading, "Subsection B")
				}
			},
		},
		"Mixed H3s with and without H4s": {
			content: `# Document Title

## Section One

This is H2 intro content.

### Subsection A

This H3 has no H4s, so it should become one chunk.

Some more content here.

### Subsection B

This H3 has H4 children, so each H4 becomes a chunk.

#### First Detail

Content for first detail.

#### Second Detail

Content for second detail.

## Section Two

Another H2 section.

### Simple Section

No H4s here either.
`,
			baseURL:    "https://example.com/doc",
			wantChunks: 4, // 1 from Subsection A + 2 H4s from Subsection B + 1 from Simple Section
			checkFunc: func(t *testing.T, chunks []Chunk) {
				// Verify we got the right mix
				foundSubsectionA := false
				foundFirstDetail := false
				foundSecondDetail := false
				foundSimpleSection := false

				for _, chunk := range chunks {
					if chunk.H3Heading == "Subsection A" {
						foundSubsectionA = true
						// Should be H3-level chunk (no H4 in heading structure)
						if chunk.Content[:len("## Section One")] != "## Section One" {
							t.Errorf("Subsection A chunk should start with H2 context")
						}
					}
					if chunk.H3Heading == "Subsection B" {
						// These are H4-level chunks
						if chunk.Content[:len("## Section One")] != "## Section One" {
							t.Errorf("Subsection B H4 chunk should start with H2 context")
						}
						// Check which H4 this is by looking for the text
						if strings.Contains(chunk.Content, "First Detail") {
							foundFirstDetail = true
						}
						if strings.Contains(chunk.Content, "Second Detail") {
							foundSecondDetail = true
						}
					}
					if chunk.H3Heading == "Simple Section" {
						foundSimpleSection = true
					}
				}

				if !foundSubsectionA {
					t.Errorf("Missing chunk for Subsection A (H3 without H4s)")
				}
				if !foundFirstDetail {
					t.Errorf("Missing chunk for First Detail H4")
				}
				if !foundSecondDetail {
					t.Errorf("Missing chunk for Second Detail H4")
				}
				if !foundSimpleSection {
					t.Errorf("Missing chunk for Simple Section")
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			chunks, err := ChunkDocument([]byte(tc.content), tc.baseURL, "/test.md", nil, StrategyH2H3H4)
			if err != nil {
				t.Fatalf("ChunkDocument failed: %v", err)
			}

			if len(chunks) != tc.wantChunks {
				t.Errorf("got %d chunks, want %d", len(chunks), tc.wantChunks)
				for i, chunk := range chunks {
					t.Logf("Chunk %d: H2=%q H3=%q", i, chunk.H2Heading, chunk.H3Heading)
				}
			}

			if tc.checkFunc != nil {
				tc.checkFunc(t, chunks)
			}

			// Verify all chunks have unique IDs
			ids := make(map[string]bool)
			for _, chunk := range chunks {
				if ids[chunk.ID] {
					t.Errorf("duplicate chunk ID: %s", chunk.ID)
				}
				ids[chunk.ID] = true
			}
		})
	}
}
