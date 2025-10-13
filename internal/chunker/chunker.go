package chunker

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Chunk represents a single chunk of content with metadata.
type Chunk struct {
	ID         string
	Content    string
	URL        string
	SourceFile string
	H2Heading  string
	H3Heading  string
	Metadata   map[string]string
}

// ChunkDocument processes a markdown document and returns contextual chunks.
// It implements the contextual retrieval strategy:
// 1. Discard H1 and preamble
// 2. Capture H2 context
// 3. Prepend context to H3 sections
// 4. Generate SHA1 ID for each chunk
// 5. Construct deep-link URLs
func ChunkDocument(content []byte, baseURL string, sourcePath string, metadata map[string]string) ([]Chunk, error) {
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(content))

	var chunks []Chunk
	var currentH2 *ast.Heading
	var h2Context strings.Builder

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		switch heading.Level {
		case 1:
			// Skip H1 - we discard it
			return ast.WalkSkipChildren, nil

		case 2:
			// Capture H2 heading and context
			currentH2 = heading
			h2Context.Reset()
			h2Context.WriteString("## ")
			h2Context.WriteString(extractText(heading, content))
			h2Context.WriteString("\n\n")

			// Get content after H2 until next heading
			h2Context.WriteString(extractContentUntilNextHeading(heading, content))

		case 3:
			if currentH2 == nil {
				// H3 before any H2, skip it
				return ast.WalkSkipChildren, nil
			}

			// Create chunk with H2 context + H3 section
			h3Text := extractText(heading, content)
			h3Content := extractContentUntilNextHeading(heading, content)

			var chunkContent strings.Builder
			chunkContent.WriteString(h2Context.String())
			chunkContent.WriteString("### ")
			chunkContent.WriteString(h3Text)
			chunkContent.WriteString("\n\n")
			chunkContent.WriteString(h3Content)

			chunkText := chunkContent.String()
			id := generateID(chunkText)
			url := constructURL(baseURL, h3Text)

			chunk := Chunk{
				ID:         id,
				Content:    chunkText,
				URL:        url,
				SourceFile: sourcePath,
				H2Heading:  extractText(currentH2, content),
				H3Heading:  h3Text,
				Metadata:   copyMetadata(metadata),
			}
			chunks = append(chunks, chunk)
		}

		return ast.WalkSkipChildren, nil
	})

	if err != nil {
		return nil, err
	}

	return chunks, nil
}

// extractText extracts the text content from a heading node.
func extractText(heading *ast.Heading, source []byte) string {
	var buf bytes.Buffer
	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		segment := child.Text(source)
		buf.Write(segment)
	}
	return buf.String()
}

// extractContentUntilNextHeading extracts content after a heading until the next heading.
func extractContentUntilNextHeading(heading *ast.Heading, source []byte) string {
	var buf bytes.Buffer

	// Start from the next sibling after the heading
	node := heading.NextSibling()
	for node != nil {
		// Stop if we hit another heading at same or higher level
		if h, ok := node.(*ast.Heading); ok && h.Level <= heading.Level {
			break
		}

		// Extract text from this node
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}

		node = node.NextSibling()
	}

	return strings.TrimSpace(buf.String())
}

// generateID creates a SHA1 hash of the chunk content.
func generateID(content string) string {
	hash := sha1.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// slugify converts a heading to a URL fragment.
func slugify(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Replace spaces and underscores with hyphens
	text = strings.ReplaceAll(text, " ", "-")
	text = strings.ReplaceAll(text, "_", "-")

	// Remove special characters
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	text = reg.ReplaceAllString(text, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	text = reg.ReplaceAllString(text, "-")

	// Trim hyphens from start and end
	text = strings.Trim(text, "-")

	return text
}

// constructURL builds a deep-link URL with the heading fragment.
func constructURL(baseURL, heading string) string {
	fragment := slugify(heading)
	if fragment == "" {
		return baseURL
	}
	return baseURL + "#" + fragment
}

// copyMetadata creates a copy of the metadata map.
func copyMetadata(metadata map[string]string) map[string]string {
	copy := make(map[string]string, len(metadata))
	for k, v := range metadata {
		copy[k] = v
	}
	return copy
}
