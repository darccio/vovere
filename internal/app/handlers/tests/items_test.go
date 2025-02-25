package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"vovere/internal/app/handlers"
	"vovere/internal/app/models"
	"vovere/internal/app/services"
)

func setupTestEnv(t *testing.T) (*services.Repository, func()) {
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
	repo := services.NewRepository(tempDir)

	// Return a cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return repo, cleanup
}

// addChiURLParams adds Chi URL parameters to a request context
func addChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestCreateItem(t *testing.T) {
	repo, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a new handler with the test repository
	handler := handlers.NewItemHandler(repo)

	// Create a test request for creating a new note
	r := httptest.NewRequest("POST", "/note", nil)
	w := httptest.NewRecorder()

	// Add Chi URL params to the request
	r = addChiURLParams(r, map[string]string{
		"type": "note",
	})

	// Call the handler
	handler.Routes().ServeHTTP(w, r)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Response should contain HTML for the editor
	if !strings.Contains(w.Body.String(), "New note") {
		t.Errorf("Response doesn't contain expected content")
	}

	// Verify that a new item was created
	items, err := repo.ListItems(models.TypeNote)
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item to be created, got %d", len(items))
	}
}

func TestUpdateContent(t *testing.T) {
	repo, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test item first
	item := models.NewItem(models.TypeNote, "test-update")
	item.Title = "Test Update"
	if err := repo.SaveItem(item, "Initial content"); err != nil {
		t.Fatalf("Failed to save test item: %v", err)
	}

	// Create a new handler with the test repository
	handler := handlers.NewItemHandler(repo)

	// Create form data
	form := url.Values{}
	form.Add("content", "Updated content")

	// Create a test request for updating the content
	r := httptest.NewRequest("PUT", "/note/test-update/content", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Add Chi URL params to the request
	r = addChiURLParams(r, map[string]string{
		"type": "note",
		"id":   "test-update",
	})

	// Call the handler
	handler.Routes().ServeHTTP(w, r)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify that the content was updated
	_, content, err := repo.LoadItem("test-update", models.TypeNote)
	if err != nil {
		t.Fatalf("Failed to load item: %v", err)
	}

	if content != "Updated content" {
		t.Errorf("Content was not updated. Expected 'Updated content', got '%s'", content)
	}
}

func TestViewItem(t *testing.T) {
	repo, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test item first
	item := models.NewItem(models.TypeNote, "test-view")
	item.Title = "Test View"
	if err := repo.SaveItem(item, "# Test View\n\nThis is a test note."); err != nil {
		t.Fatalf("Failed to save test item: %v", err)
	}

	// Create a new handler with the test repository
	handler := handlers.NewItemHandler(repo)

	// Create a test request for viewing the item
	r := httptest.NewRequest("GET", "/note/test-view", nil)
	w := httptest.NewRecorder()

	// Add Chi URL params to the request
	r = addChiURLParams(r, map[string]string{
		"type": "note",
		"id":   "test-view",
	})

	// Call the handler
	handler.Routes().ServeHTTP(w, r)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Response should contain the title and content
	response := w.Body.String()
	if !strings.Contains(response, "Test View") {
		t.Errorf("Response doesn't contain the expected title")
	}

	if !strings.Contains(response, "This is a test note") {
		t.Errorf("Response doesn't contain the expected content")
	}
}

func TestListItems(t *testing.T) {
	repo, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create multiple test items
	for i := 1; i <= 3; i++ {
		id := "test" + string(rune(48+i))
		item := models.NewItem(models.TypeNote, id)
		item.Title = "Test " + string(rune(48+i))
		if err := repo.SaveItem(item, "Content "+id); err != nil {
			t.Fatalf("Failed to save test item: %v", err)
		}
	}

	// Create a new handler with the test repository
	handler := handlers.NewItemHandler(repo)

	// Create a test request for listing items
	r := httptest.NewRequest("GET", "/note", nil)
	w := httptest.NewRecorder()

	// Add Chi URL params to the request
	r = addChiURLParams(r, map[string]string{
		"type": "note",
	})

	// Call the handler
	handler.Routes().ServeHTTP(w, r)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Response should contain all item titles
	response := w.Body.String()
	for i := 1; i <= 3; i++ {
		title := "Test " + string(rune(48+i))
		if !strings.Contains(response, title) {
			t.Errorf("Response doesn't contain expected item: %s", title)
		}
	}
}

func TestDeleteItem(t *testing.T) {
	repo, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test item first
	item := models.NewItem(models.TypeNote, "test-delete")
	item.Title = "Test Delete"
	if err := repo.SaveItem(item, "Content to delete"); err != nil {
		t.Fatalf("Failed to save test item: %v", err)
	}

	// Create a new handler with the test repository
	handler := handlers.NewItemHandler(repo)

	// Create a test request for deleting the item
	r := httptest.NewRequest("DELETE", "/note/test-delete", nil)
	w := httptest.NewRecorder()

	// Add Chi URL params to the request
	r = addChiURLParams(r, map[string]string{
		"type": "note",
		"id":   "test-delete",
	})

	// Call the handler
	handler.Routes().ServeHTTP(w, r)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify that the item was deleted
	items, err := repo.ListItems(models.TypeNote)
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items after deletion, got %d", len(items))
	}
}
