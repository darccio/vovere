package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTags(t *testing.T) {
	// Create a tag service with nil repository (we don't need it for extraction)
	tagService := NewTagService(nil)

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "Simple tags",
			content:  "This is a #test with #simple tags",
			expected: []string{"test", "simple"},
		},
		{
			name:     "Tags with dots and underscores",
			content:  "Testing #tag.with.dots and #tag_with_underscores",
			expected: []string{"tag.with.dots", "tag_with_underscores"},
		},
		{
			name:     "Tags with colons",
			content:  "Testing #namespace:name and #project:task:subtask",
			expected: []string{"namespace:name", "project:task:subtask"},
		},
		{
			name:     "Tags with special characters",
			content:  "Complex #tag-with-hyphens and #tag+plus+signs",
			expected: []string{"tag-with-hyphens", "tag+plus+signs"},
		},
		{
			name:     "Tags at start and end of line",
			content:  "#startoftag\nMiddle line\n#endoftag",
			expected: []string{"startoftag", "endoftag"},
		},
		{
			name:     "Tags with punctuation",
			content:  "This #tag! is a #goodtag: and that #tag, and another #tag.",
			expected: []string{"tag", "goodtag"},
		},
		{
			name:     "Tags with internal punctuation",
			content:  "Complex #tag.with.dots! and #tag:with:colons.",
			expected: []string{"tag.with.dots", "tag:with:colons"},
		},
		{
			name:     "Multiple tags in succession",
			content:  "Multiple #tag1 #tag2 #tag3",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "No tags",
			content:  "This content has no tags",
			expected: nil,
		},
		{
			name:     "Empty content",
			content:  "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tagService.ExtractTags(tt.content)

			// Since map iteration for building the result slice is non-deterministic,
			// we need to check that all expected tags are in the result and vice versa,
			// rather than expecting a specific order
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, len(tt.expected), len(result), "Tag count should match")

				// Check that all expected tags are in the result
				for _, expectedTag := range tt.expected {
					assert.Contains(t, result, expectedTag)
				}
			}
		})
	}
}
