package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"vovere/internal/app/models"
)

func setupTestRepo(t *testing.T) (string, *Repository, func()) {
	// Create a temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "vovere-test-*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	// Create necessary subdirectories
	dirs := []string{
		filepath.Join(tempDir, ".meta", "notes"),
		filepath.Join(tempDir, ".meta", "bookmarks"),
		filepath.Join(tempDir, ".meta", "tasks"),
		filepath.Join(tempDir, ".meta", "workstreams"),
		filepath.Join(tempDir, "notes"),
		filepath.Join(tempDir, "bookmarks"),
		filepath.Join(tempDir, "tasks"),
		filepath.Join(tempDir, "files"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create a repository service
	repo := NewRepository(tempDir)

	// Return a cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, repo, cleanup
}

func TestSaveAndLoadItem(t *testing.T) {
	_, repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a test item
	item := models.NewItem(models.TypeNote, "test123")
	item.Title = "Test Note"
	item.Tags = []string{"test", "note"}

	// Test content
	content := "# Test Note\n\nThis is a test note content."

	// Save the item
	err := repo.SaveItem(item, content)
	if err != nil {
		t.Fatalf("Failed to save item: %v", err)
	}

	// Load the item
	loadedItem, loadedContent, err := repo.LoadItem("test123", models.TypeNote)
	if err != nil {
		t.Fatalf("Failed to load item: %v", err)
	}

	// Check item fields
	if loadedItem.ID != "test123" {
		t.Errorf("Expected ID 'test123', got '%s'", loadedItem.ID)
	}

	if loadedItem.Type != models.TypeNote {
		t.Errorf("Expected type '%s', got '%s'", models.TypeNote, loadedItem.Type)
	}

	if loadedItem.Title != "Test Note" {
		t.Errorf("Expected title 'Test Note', got '%s'", loadedItem.Title)
	}

	if len(loadedItem.Tags) != 2 || loadedItem.Tags[0] != "test" || loadedItem.Tags[1] != "note" {
		t.Errorf("Tags don't match expected values: %v", loadedItem.Tags)
	}

	// Check content
	if loadedContent != content {
		t.Errorf("Content doesn't match. Expected '%s', got '%s'", content, loadedContent)
	}
}

func TestListItems(t *testing.T) {
	_, repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create multiple test items
	for i := 1; i <= 3; i++ {
		id := "note" + string(rune(48+i)) // note1, note2, note3
		item := models.NewItem(models.TypeNote, id)
		item.Title = "Test Note " + string(rune(48+i))

		// Add a small delay to ensure different timestamps
		time.Sleep(50 * time.Millisecond)

		if err := repo.SaveItem(item, "Content "+id); err != nil {
			t.Fatalf("Failed to save item %d: %v", i, err)
		}
	}

	// List items
	items, err := repo.ListItems(models.TypeNote)
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	// Check if we got the right number of items
	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}

	// Check if items are sorted by modified time (newest first)
	if len(items) >= 3 {
		if !items[0].Modified.After(items[1].Modified) {
			t.Errorf("Items not sorted correctly by modified time")
		}
		if !items[1].Modified.After(items[2].Modified) {
			t.Errorf("Items not sorted correctly by modified time")
		}
	}
}

func TestUpdateContent(t *testing.T) {
	_, repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a test item
	item := models.NewItem(models.TypeNote, "update-test")
	item.Title = "Update Test"

	// Save the item with initial content
	initialContent := "# Initial content"
	err := repo.SaveItem(item, initialContent)
	if err != nil {
		t.Fatalf("Failed to save item: %v", err)
	}

	// Record the initial modified time
	initialModified := item.Modified

	// Wait to ensure the modification time will be different
	time.Sleep(100 * time.Millisecond)

	// Update the content
	updatedContent := "# Updated content"
	if err := repo.UpdateContent(item, updatedContent); err != nil {
		t.Fatalf("Failed to update content: %v", err)
	}

	// Load the item
	loadedItem, loadedContent, err := repo.LoadItem("update-test", models.TypeNote)
	if err != nil {
		t.Fatalf("Failed to load item: %v", err)
	}

	// Check if content was updated
	if loadedContent != updatedContent {
		t.Errorf("Content was not updated. Expected '%s', got '%s'", updatedContent, loadedContent)
	}

	// Check if modified time was updated
	if !loadedItem.Modified.After(initialModified) {
		t.Errorf("Modified time was not updated")
	}
}

func TestDeleteItem(t *testing.T) {
	_, repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a test item
	item := models.NewItem(models.TypeNote, "delete-test")
	item.Title = "Delete Test"

	// Save the item
	content := "# This item will be deleted"
	err := repo.SaveItem(item, content)
	if err != nil {
		t.Fatalf("Failed to save item: %v", err)
	}

	// Confirm item exists
	_, _, err = repo.LoadItem("delete-test", models.TypeNote)
	if err != nil {
		t.Fatalf("Item should exist before deletion: %v", err)
	}

	// Delete the item
	err = repo.DeleteItem(item)
	if err != nil {
		t.Fatalf("Failed to delete item: %v", err)
	}

	// Try to load the deleted item
	_, _, err = repo.LoadItem("delete-test", models.TypeNote)
	if err == nil {
		t.Errorf("Item should no longer exist after deletion")
	}
}
