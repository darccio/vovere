package markdown

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

// Transformer is an interface for applying transformations to markdown nodes
type Transformer interface {
	// Transform processes a text node and returns whether it was handled and the transformation status
	Transform(w io.Writer, node ast.Node, text string) (handled bool, status ast.WalkStatus)

	// CanTransform determines if this transformer can handle the given node
	CanTransform(node ast.Node) bool
}

// HashtagTransformer transforms hashtags into links
type HashtagTransformer struct {
	// Regular expression for matching hashtags without trailing punctuation
	TagRegex *regexp.Regexp
}

// NewHashtagTransformer creates a new hashtag transformer
func NewHashtagTransformer() *HashtagTransformer {
	return &HashtagTransformer{
		// Match bare hashtags, punctuation will be handled separately
		TagRegex: regexp.MustCompile(`#[a-zA-Z0-9_\.\-]+`),
	}
}

// CanTransform determines if this transformer can handle the given node
func (t *HashtagTransformer) CanTransform(node ast.Node) bool {
	// Skip hashtag processing for nodes within code contexts or links
	// Check if any parent is a code block or code span
	parent := node.GetParent()
	for parent != nil {
		switch parent.(type) {
		case *ast.CodeBlock, *ast.Code, *ast.Link:
			// Don't process hashtags in code blocks, inline code, or links
			return false
		default:
			parent = parent.GetParent()
		}
	}
	return true
}

// Transform processes text to convert hashtags to links
func (t *HashtagTransformer) Transform(w io.Writer, node ast.Node, text string) (bool, ast.WalkStatus) {
	if !strings.Contains(text, "#") {
		return false, ast.GoToNext
	}

	// Using a simple parsing approach to handle hashtags and punctuation properly
	var result strings.Builder
	i := 0
	transformed := false

	for i < len(text) {
		// Look for hashtag start
		hashIndex := strings.IndexByte(text[i:], '#')
		if hashIndex == -1 {
			// No more hashtags, add remaining text
			result.WriteString(text[i:])
			break
		}

		// Absolute position of the hashtag
		hashPos := i + hashIndex

		// Check if it's part of URL or email
		if isPartOfUrlOrEmail(text, hashPos) {
			// Add text up to and including the # character
			result.WriteString(text[i : hashPos+1])
			i = hashPos + 1
			continue
		}

		// Add text before the hashtag
		result.WriteString(text[i:hashPos])
		i = hashPos

		// Extract the hashtag: start with # and include alphanumeric, underscores, dots, dashes
		tagStart := i
		i++ // Skip the # character
		for i < len(text) {
			if (text[i] >= 'a' && text[i] <= 'z') ||
				(text[i] >= 'A' && text[i] <= 'Z') ||
				(text[i] >= '0' && text[i] <= '9') ||
				text[i] == '_' || text[i] == '.' || text[i] == '-' {
				i++
			} else {
				break
			}
		}

		// If we didn't advance past the #, it's not a valid hashtag
		if i == tagStart+1 {
			result.WriteByte('#')
			continue
		}

		// Extract the tag content
		hashtag := text[tagStart:i]
		tag := hashtag[1:] // Remove the # prefix

		// Create the link for the hashtag
		result.WriteString(fmt.Sprintf(`<a href="/tags/%s" class="tag-link">%s</a>`, tag, hashtag))
		transformed = true
	}

	if transformed {
		io.WriteString(w, result.String())
		return true, ast.GoToNext
	}

	return false, ast.GoToNext
}

// isPartOfUrlOrEmail checks if a hashtag at the given position is part of a URL or email
func isPartOfUrlOrEmail(text string, position int) bool {
	// Check if there's a @ or // or : before the hashtag without whitespace
	for i := position - 1; i >= 0; i-- {
		if text[i] == ' ' || text[i] == '\n' || text[i] == '\t' {
			// Found whitespace, so this can't be part of a URL or email
			return false
		}
		if text[i] == '@' ||
			(i > 0 && text[i-1] == '/' && text[i] == '/') ||
			text[i] == ':' {
			// Found @ or // or :, likely part of URL or email
			return true
		}
	}
	return false
}
