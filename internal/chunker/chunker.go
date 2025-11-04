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

// ChunkingStrategy defines how to chunk a markdown document.
type ChunkingStrategy string

const (
	// StrategyH2H3 chunks by H2 context + H3 sections.
	StrategyH2H3 ChunkingStrategy = "h2_h3"

	// StrategyH2H3H4 chunks by H2 context + H3 sections, with H4 subsections when present.
	// If an H3 has H4 children, creates chunks for each H4.
	// If an H3 has no H4 children, creates one chunk for the entire H3.
	StrategyH2H3H4 ChunkingStrategy = "h2_h3+h4"

	// StrategyH2 chunks by H2 sections only.
	StrategyH2 ChunkingStrategy = "h2"
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
// The strategy parameter determines the chunking behavior:
// - h2_h3: H2 context + H3 sections (default)
// - h2_h3+h4: H2 context + H3 sections, splitting H4s when present
// - h2: Each H2 section becomes a chunk
func ChunkDocument(content []byte, baseURL string, sourcePath string, metadata map[string]string, strategy ChunkingStrategy) ([]Chunk, error) {
	if strategy == "" {
		strategy = StrategyH2H3
	}

	switch strategy {
	case StrategyH2H3:
		return chunkH2H3(content, baseURL, sourcePath, metadata)
	case StrategyH2H3H4:
		return chunkH2H3H4(content, baseURL, sourcePath, metadata)
	case StrategyH2:
		return chunkH2(content, baseURL, sourcePath, metadata)
	default:
		return chunkH2H3(content, baseURL, sourcePath, metadata)
	}
}

// chunkH2H3 implements the H2 context + H3 sections strategy.
func chunkH2H3(content []byte, baseURL string, sourcePath string, metadata map[string]string) ([]Chunk, error) {
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
			h2ContextStr := strings.TrimSpace(h2Context.String())
			chunkContent.WriteString(h2ContextStr)

			// Only add blank line if H2 context is not empty
			if h2ContextStr != "" {
				chunkContent.WriteString("\n\n") // Blank line before H3
			}

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

// chunkH2 implements the H2 section strategy.
// Each H2 section (including all its content until the next H2) becomes a chunk.
func chunkH2(content []byte, baseURL string, sourcePath string, metadata map[string]string) ([]Chunk, error) {
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(content))

	var chunks []Chunk

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
			// Create chunk for entire H2 section
			h2Text := extractText(heading, content)
			h2Content := extractContentUntilNextHeading(heading, content)

			var chunkContent strings.Builder
			chunkContent.WriteString("## ")
			chunkContent.WriteString(h2Text)
			chunkContent.WriteString("\n\n")
			chunkContent.WriteString(h2Content)

			chunkText := strings.TrimSpace(chunkContent.String())
			id := generateID(chunkText)
			url := constructURL(baseURL, h2Text)

			chunk := Chunk{
				ID:         id,
				Content:    chunkText,
				URL:        url,
				SourceFile: sourcePath,
				H2Heading:  h2Text,
				H3Heading:  "", // No H3 in this strategy
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

// chunkH2H3H4 implements the hybrid H2 context + H3/H4 sections strategy.
// If an H3 has H4 children, creates chunks for each H4 (with H2+H3 context).
// If an H3 has no H4 children, creates one chunk for the entire H3 (with H2 context).
func chunkH2H3H4(content []byte, baseURL string, sourcePath string, metadata map[string]string) ([]Chunk, error) {
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(content))

	var chunks []Chunk
	var currentH2 *ast.Heading
	var currentH3 *ast.Heading
	var h2Context strings.Builder
	var h3Context strings.Builder

	// First pass: identify which H3s have H4 children
	h3HasH4 := make(map[ast.Node]bool)
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		if heading.Level == 3 {
			currentH3 = heading
		} else if heading.Level == 4 && currentH3 != nil {
			h3HasH4[currentH3] = true
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: create chunks
	currentH2 = nil
	currentH3 = nil

	err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		switch heading.Level {
		case 1:
			return ast.WalkSkipChildren, nil

		case 2:
			currentH2 = heading
			h2Context.Reset()
			h2Context.WriteString("## ")
			h2Context.WriteString(extractText(heading, content))
			h2Context.WriteString("\n\n")
			h2Context.WriteString(extractH2Context(heading, content))

		case 3:
			if currentH2 == nil {
				return ast.WalkSkipChildren, nil
			}

			currentH3 = heading
			h3Text := extractText(heading, content)

			// Check if this H3 has H4 children
			if h3HasH4[heading] {
				// Has H4s - prepare H3 context for H4 chunks
				h3Context.Reset()
				h2ContextStr := strings.TrimSpace(h2Context.String())
				if h2ContextStr != "" {
					h3Context.WriteString(h2ContextStr)
					h3Context.WriteString("\n\n")
				}
				h3Context.WriteString("### ")
				h3Context.WriteString(h3Text)
				h3Context.WriteString("\n\n")
				h3Context.WriteString(extractH3Context(heading, content))
			} else {
				// No H4s - create chunk for entire H3
				h3Content := extractContentUntilNextHeading(heading, content)

				var chunkContent strings.Builder
				h2ContextStr := strings.TrimSpace(h2Context.String())
				if h2ContextStr != "" {
					chunkContent.WriteString(h2ContextStr)
					chunkContent.WriteString("\n\n")
				}
				chunkContent.WriteString("### ")
				chunkContent.WriteString(h3Text)
				chunkContent.WriteString("\n\n")
				chunkContent.WriteString(h3Content)

				chunkText := strings.TrimSpace(chunkContent.String())
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

		case 4:
			if currentH3 == nil || !h3HasH4[currentH3] {
				return ast.WalkSkipChildren, nil
			}

			// Create chunk with H2 context + H3 context + H4 section
			h4Text := extractText(heading, content)
			h4Content := extractContentUntilNextHeading(heading, content)

			var chunkContent strings.Builder
			h3ContextStr := strings.TrimSpace(h3Context.String())
			if h3ContextStr != "" {
				chunkContent.WriteString(h3ContextStr)
				chunkContent.WriteString("\n\n")
			}
			chunkContent.WriteString("#### ")
			chunkContent.WriteString(h4Text)
			chunkContent.WriteString("\n\n")
			chunkContent.WriteString(h4Content)

			chunkText := strings.TrimSpace(chunkContent.String())
			id := generateID(chunkText)
			url := constructURL(baseURL, h4Text)

			chunk := Chunk{
				ID:         id,
				Content:    chunkText,
				URL:        url,
				SourceFile: sourcePath,
				H2Heading:  extractText(currentH2, content),
				H3Heading:  extractText(currentH3, content),
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

// extractH3Context extracts content after H3 until first H4 or any other heading.
func extractH3Context(heading *ast.Heading, source []byte) string {
	var buf bytes.Buffer

	node := heading.NextSibling()
	for node != nil {
		// Stop if we hit ANY heading (H4, H3, or H2)
		if _, ok := node.(*ast.Heading); ok {
			break
		}

		renderNodeToMarkdown(&buf, node, source)
		node = node.NextSibling()
	}

	return strings.TrimSpace(buf.String())
}

// copyMetadata creates a copy of the metadata map.
func copyMetadata(metadata map[string]string) map[string]string {
	copy := make(map[string]string, len(metadata))
	for k, v := range metadata {
		copy[k] = v
	}
	return copy
}
