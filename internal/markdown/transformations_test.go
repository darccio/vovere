package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gomarkdown/markdown/ast"
)

// TestHashtagTransformer tests the hashtag transformer
func TestHashtagTransformer(t *testing.T) {
	transformer := NewHashtagTransformer()

	testCases := []struct {
		name     string
		input    string
		expected string
		handled  bool
	}{
		{
			name:     "Basic hashtag",
			input:    "This is a #tag in text",
			expected: `This is a <a href="/tags/tag" class="tag-link">#tag</a> in text`,
			handled:  true,
		},
		{
			name:     "Multiple hashtags",
			input:    "Multiple #tags in #one sentence",
			expected: `Multiple <a href="/tags/tags" class="tag-link">#tags</a> in <a href="/tags/one" class="tag-link">#one</a> sentence`,
			handled:  true,
		},
		{
			name:     "Hashtag at beginning",
			input:    "#HashtagAtBeginning of text",
			expected: `<a href="/tags/HashtagAtBeginning" class="tag-link">#HashtagAtBeginning</a> of text`,
			handled:  true,
		},
		{
			name:     "Hashtag with dots",
			input:    "Complex #tag.with.dots here",
			expected: `Complex <a href="/tags/tag.with.dots" class="tag-link">#tag.with.dots</a> here`,
			handled:  true,
		},
		{
			name:     "Hashtag with dashes",
			input:    "Complex #tag-with-dashes here",
			expected: `Complex <a href="/tags/tag-with-dashes" class="tag-link">#tag-with-dashes</a> here`,
			handled:  true,
		},
		{
			name:     "Adjacent hashtags",
			input:    "Adjacent #tags#more here",
			expected: `Adjacent <a href="/tags/tags" class="tag-link">#tags</a><a href="/tags/more" class="tag-link">#more</a> here`,
			handled:  true,
		},
		{
			name:     "No hashtags",
			input:    "Text without any hashtags",
			expected: "",
			handled:  false,
		},
	}

	// Create a dummy text node for testing
	dummyNode := &ast.Text{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			handled, _ := transformer.Transform(&buf, dummyNode, tc.input)

			if handled != tc.handled {
				t.Errorf("Expected handled=%v, got %v", tc.handled, handled)
			}

			if handled {
				result := buf.String()
				if result != tc.expected {
					t.Errorf("Expected: %s\nGot: %s", tc.expected, result)
				}
			}
		})
	}
}

// TestMarkdownRendering tests the full rendering process
func TestMarkdownRendering(t *testing.T) {
	testCases := []struct {
		name          string
		markdown      string
		expectedParts []string
		notExpected   []string
	}{
		{
			name: "Basic hashtags",
			markdown: `# Test Document
			
Regular paragraph with #hashtag.`,
			expectedParts: []string{
				`<h1 id="test-document">Test Document</h1>`,
				`Regular paragraph with <a href="/tags/hashtag." class="tag-link">#hashtag.</a>`,
			},
			notExpected: nil,
		},
		{
			name:     "Code blocks should not transform hashtags",
			markdown: "```\nCode block with #hashtag\n```",
			expectedParts: []string{
				"Code block with #hashtag",
			},
			notExpected: []string{
				`<a href="/tags/hashtag"`,
			},
		},
		{
			name:     "Inline code should not transform hashtags",
			markdown: "This is `inline code with #hashtag` in text",
			expectedParts: []string{
				"inline code with #hashtag",
			},
			notExpected: []string{
				`<a href="/tags/hashtag"`,
			},
		},
		{
			name:     "Links should not transform hashtags",
			markdown: "This is a [link with #hashtag](https://example.com)",
			expectedParts: []string{
				`<a href="https://example.com" target="_blank">link with #hashtag</a>`,
			},
			notExpected: []string{
				`<a href="/tags/hashtag"`,
			},
		},
		{
			name:     "Multiple hashtags with formatting",
			markdown: "**Bold text** with #hashtag and *italic* with #another-tag",
			expectedParts: []string{
				`<strong>Bold text</strong> with <a href="/tags/hashtag" class="tag-link">#hashtag</a>`,
				`<em>italic</em> with <a href="/tags/another-tag" class="tag-link">#another-tag</a>`,
			},
			notExpected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Render(tc.markdown)

			// Check for expected parts
			for _, expected := range tc.expectedParts {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s' but it didn't.\nResult: %s",
						expected, result)
				}
			}

			// Check that forbidden parts don't appear
			for _, notExpected := range tc.notExpected {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected result NOT to contain '%s' but it did.\nResult: %s",
						notExpected, result)
				}
			}
		})
	}
}

// TestPeriodAfterHashtag specifically tests handling periods after hashtags
func TestPeriodAfterHashtag(t *testing.T) {
	input := "Test with #hashtag."
	expected := "Test with <a href=\"/tags/hashtag.\" class=\"tag-link\">#hashtag.</a>"

	result := Render(input)
	if !strings.Contains(result, expected) {
		t.Errorf("Period handling failed.\nExpected: %s\nGot: %s", expected, result)
	}
}
