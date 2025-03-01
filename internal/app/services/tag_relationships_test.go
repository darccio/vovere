package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"vovere/internal/app/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTagRelationships tests tag relationship functionality
func TestTagRelationships(t *testing.T) {
	// Create a temporary directory for the repository
	tempDir, err := os.MkdirTemp("", "vovere-test")
	require.NoError(t, err)

	// Add cleanup at the beginning to ensure it runs even if the test fails
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	// Create subdirectories needed for the test
	metaDir := filepath.Join(tempDir, ".meta")
	tagsDir := filepath.Join(metaDir, "tags")
	notesDir := filepath.Join(tempDir, "notes")
	metaNotesDir := filepath.Join(metaDir, "notes")

	for _, dir := range []string{metaDir, tagsDir, notesDir, metaNotesDir} {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	// Create a repository service
	repo := NewRepository(tempDir)
	tagService := NewTagService(repo)

	// Create test items with tags
	items := []*models.Item{
		{
			ID:       "item1",
			Type:     models.TypeNote,
			Title:    "Item 1",
			Tags:     []string{"tag1", "tag2"},
			Created:  time.Now(),
			Modified: time.Now(),
		},
		{
			ID:       "item2",
			Type:     models.TypeNote,
			Title:    "Item 2",
			Tags:     []string{"tag2", "tag3"},
			Created:  time.Now(),
			Modified: time.Now(),
		},
		{
			ID:       "item3",
			Type:     models.TypeNote,
			Title:    "Item 3",
			Tags:     []string{"tag1", "tag3", "tag4"},
			Created:  time.Now(),
			Modified: time.Now(),
		},
	}

	// Save items to repository with mock content that includes hashtags
	for i, item := range items {
		// Content with hashtags corresponding to the tags
		content := ""
		for _, tag := range item.Tags {
			content += "#" + tag + " "
		}
		content += "Item content " + item.ID

		// Save item with proper error handling
		err := repo.SaveItem(item, content)
		if err != nil {
			t.Fatalf("Failed to save item %d: %v", i, err)
		}
	}

	// Test GetAllTags
	t.Run("GetAllTags", func(t *testing.T) {
		tags, err := tagService.GetAllTags()
		require.NoError(t, err)

		// Check that we have the expected number of tags
		assert.Equal(t, 4, len(tags), "Expected 4 tags, got %d", len(tags))

		// Verify all tags are present
		expectedTags := []string{"tag1", "tag2", "tag3", "tag4"}
		for _, tag := range expectedTags {
			assert.Contains(t, tags, tag)
		}
	})

	// Test GetItemsByTag
	t.Run("GetItemsByTag", func(t *testing.T) {
		// Test items with tag1
		tag1Items, err := tagService.GetItemsByTag("tag1")
		require.NoError(t, err)
		assert.Equal(t, 2, len(tag1Items), "Expected 2 items with tag1")

		// Test items with tag2
		tag2Items, err := tagService.GetItemsByTag("tag2")
		require.NoError(t, err)
		assert.Equal(t, 2, len(tag2Items), "Expected 2 items with tag2")

		// Test items with tag4
		tag4Items, err := tagService.GetItemsByTag("tag4")
		require.NoError(t, err)
		assert.Equal(t, 1, len(tag4Items), "Expected 1 item with tag4")
	})

	// Test GetItemsByMultipleTags with timeout to prevent hanging
	t.Run("GetItemsByMultipleTags", func(t *testing.T) {
		// Items with both tag1 and tag3
		items, err := tagService.GetItemsByMultipleTags([]string{"tag1", "tag3"})
		require.NoError(t, err)
		assert.Equal(t, 1, len(items), "Expected 1 item with both tag1 and tag3")

		if len(items) > 0 {
			assert.Equal(t, "item3", items[0].ID)
		}

		// No items with both tag1 and tag2 and tag4
		noItems, err := tagService.GetItemsByMultipleTags([]string{"tag1", "tag2", "tag4"})
		require.NoError(t, err)
		assert.Equal(t, 0, len(noItems), "Expected 0 items with tag1, tag2, and tag4")
	})

	// Test GetTagStatistics
	t.Run("GetTagStatistics", func(t *testing.T) {
		stats, err := tagService.GetTagStatistics()
		require.NoError(t, err)
		assert.Equal(t, 2, stats["tag1"], "Expected 2 items with tag1")
		assert.Equal(t, 2, stats["tag2"], "Expected 2 items with tag2")
		assert.Equal(t, 2, stats["tag3"], "Expected 2 items with tag3")
		assert.Equal(t, 1, stats["tag4"], "Expected 1 item with tag4")
	})

	// Test SearchTags
	t.Run("SearchTags", func(t *testing.T) {
		// Search for tags starting with 'tag'
		tagMatches, err := tagService.SearchTags("tag")
		require.NoError(t, err)
		assert.Equal(t, 4, len(tagMatches), "Expected 4 tags starting with 'tag'")

		// Search for specific tag
		specificMatch, err := tagService.SearchTags("tag1")
		require.NoError(t, err)
		assert.Equal(t, 1, len(specificMatch), "Expected 1 tag matching 'tag1'")

		if len(specificMatch) > 0 {
			assert.Equal(t, "tag1", specificMatch[0])
		}

		// Search for non-existent tag
		noMatches, err := tagService.SearchTags("nonexistent")
		require.NoError(t, err)
		assert.Equal(t, 0, len(noMatches), "Expected 0 tags matching 'nonexistent'")
	})

	// Test updating tags with manual fix
	t.Run("UpdateTags", func(t *testing.T) {
		// Get initial count of items with tag1
		initialTag1Items, err := tagService.GetItemsByTag("tag1")
		require.NoError(t, err)
		initialTag1Count := len(initialTag1Items)
		assert.Equal(t, 2, initialTag1Count, "Initially should be 2 items with tag1")

		// Load item3 that initially has tag1, tag3, tag4
		item3, _, err := repo.LoadItem("item3", models.TypeNote)
		require.NoError(t, err)
		require.NotNil(t, item3)
		assert.Contains(t, item3.Tags, "tag1", "item3 should initially have tag1")

		// Create new content with tags "tag3", "tag5" (removing tag1 and tag4, adding tag5)
		newContent := "#tag3 #tag5 Updated content for item3"

		// Update the content which should update the tags
		err = repo.UpdateContent(item3, newContent)
		require.NoError(t, err, "Error updating item3 content")

		// Give file system time to sync
		time.Sleep(100 * time.Millisecond)

		// Load item3 again to verify tags were updated
		updatedItem3, _, err := repo.LoadItem("item3", models.TypeNote)
		require.NoError(t, err)
		require.NotNil(t, updatedItem3)

		// Verify updated tags - should have tag3 and tag5, but not tag1 or tag4
		assert.Equal(t, 2, len(updatedItem3.Tags), "item3 should have exactly 2 tags after update")
		assert.Contains(t, updatedItem3.Tags, "tag3", "item3 should still have tag3")
		assert.Contains(t, updatedItem3.Tags, "tag5", "item3 should now have tag5")
		assert.NotContains(t, updatedItem3.Tags, "tag1", "item3 should no longer have tag1")
		assert.NotContains(t, updatedItem3.Tags, "tag4", "item3 should no longer have tag4")

		// Create a new tag service to ensure we're not getting cached results
		freshTagService := NewTagService(repo)

		// Get updated count of items with tag1 - should decrease by 1
		tag1Items, err := freshTagService.GetItemsByTag("tag1")
		require.NoError(t, err)
		assert.Equal(t, initialTag1Count-1, len(tag1Items), "Should be one fewer item with tag1")

		// Check if tag5 now contains item3
		tag5Items, err := freshTagService.GetItemsByTag("tag5")
		require.NoError(t, err)
		assert.Equal(t, 1, len(tag5Items), "Should be exactly one item with tag5")
		if len(tag5Items) > 0 {
			assert.Equal(t, "item3", tag5Items[0].ID, "The item with tag5 should be item3")
		}

		// Verify tag statistics were updated correctly
		stats, err := freshTagService.GetTagStatistics()
		require.NoError(t, err)
		assert.Equal(t, 1, stats["tag1"], "Should be 1 item with tag1 in statistics")
		assert.Equal(t, 1, stats["tag5"], "Should be 1 item with tag5 in statistics")
	})
}
