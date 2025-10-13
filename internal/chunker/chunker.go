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

			// Get content after H2 but stop at first H3 (or any other heading)
			h2Context.WriteString(extractH2Context(heading, content))

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
		if textNode, ok := child.(*ast.Text); ok {
			buf.Write(textNode.Segment.Value(source))
		}
	}
	return buf.String()
}

// extractH2Context extracts content after H2 until first H3 or any other heading.
func extractH2Context(heading *ast.Heading, source []byte) string {
	var buf bytes.Buffer

	// Start from the next sibling after the heading
	node := heading.NextSibling()
	for node != nil {
		// Stop if we hit ANY heading (H3 or H2)
		if _, ok := node.(*ast.Heading); ok {
			break
		}

		// Render this node to markdown
		renderNodeToMarkdown(&buf, node, source)

		node = node.NextSibling()
	}

	return strings.TrimSpace(buf.String())
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

		// Render this node to markdown
		renderNodeToMarkdown(&buf, node, source)

		node = node.NextSibling()
	}

	return strings.TrimSpace(buf.String())
}

// renderNodeToMarkdown renders an AST node back to markdown format.
func renderNodeToMarkdown(buf *bytes.Buffer, node ast.Node, source []byte) {
	switch n := node.(type) {
	case *ast.Paragraph:
		renderParagraph(buf, n, source)
		buf.WriteString("\n\n")

	case *ast.FencedCodeBlock:
		renderFencedCodeBlock(buf, n, source)
		buf.WriteString("\n\n")

	case *ast.CodeBlock:
		renderCodeBlock(buf, n, source)
		buf.WriteString("\n\n")

	case *ast.Blockquote:
		renderBlockquote(buf, n, source)
		buf.WriteString("\n\n")

	case *ast.List:
		renderList(buf, n, source, 0)
		buf.WriteString("\n\n")

	case *ast.ThematicBreak:
		buf.WriteString("---\n\n")

	default:
		// For other nodes, extract text from lines
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}
		if lines.Len() > 0 {
			buf.WriteString("\n\n")
		}
	}
}

// renderParagraph renders a paragraph with inline elements.
func renderParagraph(buf *bytes.Buffer, p *ast.Paragraph, source []byte) {
	for child := p.FirstChild(); child != nil; child = child.NextSibling() {
		renderInline(buf, child, source)
	}
}

// renderInline renders inline elements like text, emphasis, code, etc.
func renderInline(buf *bytes.Buffer, node ast.Node, source []byte) {
	switch n := node.(type) {
	case *ast.Text:
		buf.Write(n.Segment.Value(source))
		if n.HardLineBreak() {
			buf.WriteString("\n")
		} else if n.SoftLineBreak() {
			buf.WriteString("\n")
		}

	case *ast.Emphasis:
		buf.WriteString(strings.Repeat("*", n.Level))
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			renderInline(buf, child, source)
		}
		buf.WriteString(strings.Repeat("*", n.Level))

	case *ast.CodeSpan:
		buf.WriteString("`")
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			renderInline(buf, child, source)
		}
		buf.WriteString("`")

	case *ast.Link:
		buf.WriteString("[")
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			renderInline(buf, child, source)
		}
		buf.WriteString("](")
		buf.Write(n.Destination)
		if n.Title != nil {
			buf.WriteString(` "`)
			buf.Write(n.Title)
			buf.WriteString(`"`)
		}
		buf.WriteString(")")

	default:
		// Fallback: extract raw text
		if textNode, ok := node.(*ast.Text); ok {
			buf.Write(textNode.Segment.Value(source))
		}
	}
}

// renderFencedCodeBlock renders a fenced code block with language info.
func renderFencedCodeBlock(buf *bytes.Buffer, cb *ast.FencedCodeBlock, source []byte) {
	buf.WriteString("```")
	if cb.Info != nil {
		buf.Write(cb.Info.Segment.Value(source))
	}
	buf.WriteString("\n")

	lines := cb.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(source))
	}

	buf.WriteString("```")
}

// renderCodeBlock renders an indented code block.
func renderCodeBlock(buf *bytes.Buffer, cb *ast.CodeBlock, source []byte) {
	lines := cb.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.WriteString("    ")
		buf.Write(line.Value(source))
	}
}

// renderBlockquote renders a blockquote.
func renderBlockquote(buf *bytes.Buffer, bq *ast.Blockquote, source []byte) {
	// Render children and prefix each line with "> "
	var childBuf bytes.Buffer
	for child := bq.FirstChild(); child != nil; child = child.NextSibling() {
		renderNodeToMarkdown(&childBuf, child, source)
	}

	// Prefix each line with "> "
	lines := strings.Split(strings.TrimRight(childBuf.String(), "\n"), "\n")
	for _, line := range lines {
		if line != "" {
			buf.WriteString("> ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
}

// renderList renders a list (ordered or unordered).
func renderList(buf *bytes.Buffer, list *ast.List, source []byte, indent int) {
	counter := list.Start
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if item, ok := child.(*ast.ListItem); ok {
			renderListItem(buf, item, list.IsOrdered(), counter, source, indent)
			counter++
		}
	}
}

// renderListItem renders a single list item.
func renderListItem(buf *bytes.Buffer, item *ast.ListItem, ordered bool, counter int, source []byte, indent int) {
	// Write indentation
	buf.WriteString(strings.Repeat("  ", indent))

	// Write marker
	if ordered {
		buf.WriteString(fmt.Sprintf("%d. ", counter))
	} else {
		buf.WriteString("- ")
	}

	// Render children
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		if childList, ok := child.(*ast.List); ok {
			buf.WriteString("\n")
			renderList(buf, childList, source, indent+1)
		} else {
			renderNodeToMarkdown(buf, child, source)
		}
	}
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
