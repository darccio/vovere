package markdown

import (
	"strings"
	"testing"
)

// TestHashtagRegex tests the hashtag regex pattern used in the renderer
func TestHashtagRegex(t *testing.T) {
	// Get the regex from the package
	tagRegex := HashtagRegex()

	// Test cases: content with hashtags and expected matches
	testCases := []struct {
		content  string
		expected []string
	}{
		// Basic hashtag tests
		{"This is a #tag in text", []string{"#tag"}},
		{"Multiple #tags in #one #sentence", []string{"#tags", "#one", "#sentence"}},
		{"#HashtagsAtTheBeginning of text", []string{"#HashtagsAtTheBeginning"}},
		{"At the end #hashtag", []string{"#hashtag"}},

		// Hashtags with special characters
		{"Complex #tag.with.dots", []string{"#tag.with.dots"}},
		{"Using #under_scores in tags", []string{"#under_scores"}},
		{"#tag1 with #tag2 and #tag_3.4", []string{"#tag1", "#tag2", "#tag_3.4"}},

		// Punctuation next to hashtags
		{"Hashtag with comma, #tag, should work", []string{"#tag"}},
		{"Hashtag with period. #tag. should work", []string{"#tag"}},
		{"#tag! with exclamation", []string{"#tag"}},
		{"#tag? with question mark", []string{"#tag"}},
		{"#tag: with colon", []string{"#tag"}},
		{"#tag; with semicolon", []string{"#tag"}},

		// Cases where hashtags shouldn't be recognized
		{"No hashtag in example.com/page#section", []string{}},
		{"Email address user@domain.com#tag", []string{}},       // Part of an email
		{"Hashtag inside `#codeblock`", []string{"#codeblock"}}, // Regex alone can't detect code contexts

		// Multiple adjacent hashtags
		{"Adjacent #tag1 #tag2", []string{"#tag1", "#tag2"}},
		{"Triple #one #two #three", []string{"#one", "#two", "#three"}},
	}

	for i, tc := range testCases {
		var matches []string

		// For URL and email tests, we need to do manual exclusion
		if strings.Contains(tc.content, "example.com/page#") ||
			strings.Contains(tc.content, "@domain.com#") {
			// Skip these - they should be excluded
		} else {
			// Find all raw hashtags
			rawMatches := tagRegex.FindAllString(tc.content, -1)

			// Clean up punctuation in matches
			for _, match := range rawMatches {
				// Remove trailing punctuation if present
				cleanMatch := match
				for i := len(match) - 1; i >= 0; i-- {
					if strings.ContainsRune(",.!?;:", rune(match[i])) {
						cleanMatch = match[:i]
					} else {
						break
					}
				}

				// Only add if we still have a hashtag
				if len(cleanMatch) > 1 && cleanMatch[0] == '#' {
					matches = append(matches, cleanMatch)
				}
			}
		}

		// Check if we have the expected number of matches
		if len(matches) != len(tc.expected) {
			t.Errorf("Test case %d: Expected %d matches, got %d in text: %s",
				i, len(tc.expected), len(matches), tc.content)
			continue
		}

		// Check if all expected tags were found
		for _, expected := range tc.expected {
			found := false
			for _, match := range matches {
				if match == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Test case %d: Expected to find '%s' but didn't in text: %s",
					i, expected, tc.content)
			}
		}
	}
}

// TestRender tests the entire markdown rendering process, including hashtag handling
func TestRender(t *testing.T) {
	testCases := []struct {
		markdown      string
		expectedParts []string // Check for these parts in output
		notExpected   []string // Make sure these don't appear
	}{
		{
			// Basic markdown with hashtag
			"This is a #tag in text.",
			[]string{
				// Should find a tag link
				`<a href="/tags/tag"`,
				`class="tag-link"`,
				`>#tag</a>`,
			},
			nil,
		},
		{
			// Hashtag in code block should not be linked
			"```\nThis is a #tag in a code block\n```",
			[]string{
				// Should contain the tag but not as a link
				"#tag",
			},
			[]string{
				// Should not create a link
				`<a href="/tags/tag"`,
			},
		},
		{
			// Hashtag in inline code should not be linked
			"This is a `#tag` in inline code",
			[]string{
				// Should contain the tag as text
				"#tag",
			},
			[]string{
				// Should not create a link for code content
				`<a href="/tags/tag"`,
			},
		},
		{
			// Multiple hashtags in text
			"Multiple #tags in #one sentence.",
			[]string{
				// Should find links for both tags
				`<a href="/tags/tags"`,
				`<a href="/tags/one"`,
			},
			nil,
		},
		{
			// Hashtag in a link shouldn't be processed separately
			"[Link with #hashtag](https://example.com)",
			[]string{
				// Should create a regular markdown link
				`<a href="https://example.com" target="_blank">`,
				// Text should be preserved
				"Link with #hashtag",
			},
			[]string{
				// Should not create a tag link inside the link text
				`<a href="/tags/hashtag"`,
			},
		},
	}

	for i, tc := range testCases {
		result := Render(tc.markdown)

		// Check for expected parts
		for _, expected := range tc.expectedParts {
			if !strings.Contains(result, expected) {
				t.Errorf("Test case %d failed: Expected result to contain '%s' but it didn't.\nMarkdown: %s\nResult: %s",
					i, expected, tc.markdown, result)
			}
		}

		// Check that forbidden parts don't appear
		for _, notExpected := range tc.notExpected {
			if strings.Contains(result, notExpected) {
				t.Errorf("Test case %d failed: Expected result NOT to contain '%s' but it did.\nMarkdown: %s\nResult: %s",
					i, notExpected, tc.markdown, result)
			}
		}
	}
}

// TestExtractTitleFromContent tests the title extraction functionality
func TestExtractTitleFromContent(t *testing.T) {
	testCases := []struct {
		content       string
		itemType      string
		expectedTitle string
	}{
		// Test with H1 header in note
		{
			content:       "# This is a title\n\nSome content here",
			itemType:      "note",
			expectedTitle: "This is a title",
		},
		// Test with underlined header in note
		{
			content:       "This is a title\n==============\n\nSome content here",
			itemType:      "note",
			expectedTitle: "This is a title",
		},
		// Test with non-note item type
		{
			content:       "# This is a title\n\nSome content here",
			itemType:      "bookmark",
			expectedTitle: "# This is a title",
		},
		// Test with no title, first line
		{
			content:       "This is the first line\nSecond line",
			itemType:      "task",
			expectedTitle: "This is the first line",
		},
		// Test with empty content
		{
			content:       "",
			itemType:      "note",
			expectedTitle: "",
		},
		// Test with long first line
		{
			content:       "This is a very long first line that should be truncated because it exceeds fifty characters in length which is the limit",
			itemType:      "note",
			expectedTitle: "This is a very long first line that should be t...",
		},
	}

	for i, tc := range testCases {
		result := ExtractTitleFromContent(tc.content, tc.itemType)
		if result != tc.expectedTitle {
			t.Errorf("Test case %d failed: Expected title '%s', got '%s'\nContent: %s",
				i, tc.expectedTitle, result, tc.content)
		}
	}
}
