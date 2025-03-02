package markdown

import (
	"io"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// defaultTransformers creates the default set of transformations
func defaultTransformers() []Transformer {
	return []Transformer{
		NewHashtagTransformer(),
	}
}

// RenderOptions holds options for the markdown renderer
type RenderOptions struct {
	// Transformers is a list of transformers to be applied to text nodes
	Transformers []Transformer
}

// DefaultRenderOptions returns the default rendering options
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		Transformers: defaultTransformers(),
	}
}

// Render converts markdown to HTML and applies all registered transformations
func Render(md string) string {
	return RenderWithOptions(md, DefaultRenderOptions())
}

// RenderWithOptions converts markdown to HTML using specified options
func RenderWithOptions(md string, opts RenderOptions) string {
	// Create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	// Parse the markdown document
	doc := p.Parse([]byte(md))

	// Set up custom HTML renderer with options
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	rendererOpts := html.RendererOptions{
		Flags: htmlFlags,
		RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
			if txtNode, ok := node.(*ast.Text); ok && entering {
				// Standard text node handling for transformations
				// Get the text from the node
				text := string(txtNode.Literal)

				// Apply transformations
				for _, transformer := range opts.Transformers {
					if transformer.CanTransform(node) {
						handled, status := transformer.Transform(w, node, text)
						if handled {
							return status, true
						}
					}
				}
			}
			return ast.GoToNext, false
		},
	}
	renderer := html.NewRenderer(rendererOpts)

	// Render to HTML
	return string(markdown.Render(doc, renderer))
}

// ExtractTitleFromContent extracts the title from content
func ExtractTitleFromContent(content string, itemType string) string {
	if content == "" {
		return ""
	}

	// For notes, try to extract the first H1 header
	if itemType == "note" {
		// Look for # Header or === underlined header
		h1Regex := regexp.MustCompile(`(?m)^#\s+(.+)$|^([^\n]+)\n===+\s*$`)
		if matches := h1Regex.FindStringSubmatch(content); len(matches) > 0 {
			for i := 1; i < len(matches); i++ {
				if matches[i] != "" {
					return strings.TrimSpace(matches[i])
				}
			}
		}
	}

	// Default: use first line
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Return first non-empty line, truncate if needed
			if len(line) > 50 {
				return line[:47] + "..."
			}
			return line
		}
	}

	return ""
}

// HashtagRegex returns the regex pattern used for matching hashtags
func HashtagRegex() *regexp.Regexp {
	// For testing purposes, we just want the basic hashtag pattern without punctuation handling
	return regexp.MustCompile(`#[a-zA-Z0-9_\.\-]+`)
}
