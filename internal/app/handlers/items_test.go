package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"vovere/internal/app/models"
	"vovere/internal/app/services"
)

// TestHashtagRegex tests the hashtag regex pattern used in the renderer
func TestHashtagRegex(t *testing.T) {
	// The regex pattern we're using in renderMarkdown
	tagRegex := regexp.MustCompile(`\B(#[a-zA-Z0-9_\.]+)\b`)

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
		found := tagRegex.FindAllStringSubmatch(tc.content, -1)
		for _, match := range found {
			if len(match) > 1 {
				matches = append(matches, match[0])
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
				if strings.Contains(match, expected) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Test case %d: Expected to find tag %s but didn't in text: %s",
					i, expected, tc.content)
			}
		}
	}
}

// TestRenderMarkdown tests the entire markdown rendering process, including hashtag handling
func TestRenderMarkdown(t *testing.T) {
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
		result := renderMarkdown(tc.markdown)

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
	handler := NewItemHandler(repo)

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

	// Check for the HX-Redirect header
	redirectHeader := w.Header().Get("HX-Redirect")
	if redirectHeader == "" {
		t.Errorf("Expected HX-Redirect header to be set")
	}

	// The header should point to the edit view for the new item
	if !strings.Contains(redirectHeader, "/items/note/") {
		t.Errorf("Redirect URL doesn't contain expected path, got: %s", redirectHeader)
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
	handler := NewItemHandler(repo)

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
	handler := NewItemHandler(repo)

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
	handler := NewItemHandler(repo)

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
	handler := NewItemHandler(repo)

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

// TestDeleteItemWithTags verifies that when an item with tags is deleted,
// the item is properly removed from all tag associations
func TestDeleteItemWithTags(t *testing.T) {
	// Setup test environment
	repo, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create tag service
	tagService := services.NewTagService(repo)

	// Create a test item (initially without tags)
	item := models.NewItem(models.TypeNote, "tagged-item-to-delete")
	item.Title = "Tagged Item To Delete"
	if err := repo.SaveItem(item, "Content of tagged item"); err != nil {
		t.Fatalf("Failed to save test item: %v", err)
	}

	// Add tags to the item (using empty previous tags as this is the first update)
	previousTags := []string{}
	item.Tags = []string{"test-tag1", "test-tag2"}
	if err := tagService.UpdateItemTags(item, previousTags); err != nil {
		t.Fatalf("Failed to add tags to item: %v", err)
	}

	// Verify tags were added correctly
	items1, err := tagService.GetItemsByTag("test-tag1")
	if err != nil {
		t.Fatalf("Failed to get items by tag: %v", err)
	}
	if len(items1) != 1 {
		t.Errorf("Expected 1 item with tag1, got %d", len(items1))
	}

	// Create item handler
	handler := NewItemHandler(repo)

	// Create a test request for deleting the item
	r := httptest.NewRequest("DELETE", "/note/tagged-item-to-delete", nil)
	w := httptest.NewRecorder()

	// Add Chi URL params to the request
	r = addChiURLParams(r, map[string]string{
		"type": "note",
		"id":   "tagged-item-to-delete",
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

	// Verify the item was removed from all tags
	items1, err = tagService.GetItemsByTag("test-tag1")
	if err != nil {
		t.Fatalf("Failed to get items by tag after deletion: %v", err)
	}
	if len(items1) != 0 {
		t.Errorf("Expected 0 items with tag1 after deletion, got %d", len(items1))
	}

	items2, err := tagService.GetItemsByTag("test-tag2")
	if err != nil {
		t.Fatalf("Failed to get items by tag after deletion: %v", err)
	}
	if len(items2) != 0 {
		t.Errorf("Expected 0 items with tag2 after deletion, got %d", len(items2))
	}
}
