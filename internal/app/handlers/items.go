package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"vovere/internal/app/models"
	"vovere/internal/app/services"
	md "vovere/internal/markdown"
)

// ItemHandler handles HTTP requests for items
type ItemHandler struct {
	repo       *services.Repository
	tagService *services.TagService
}

// NewItemHandler creates a new item handler
func NewItemHandler(repo *services.Repository) *ItemHandler {
	return &ItemHandler{
		repo:       repo,
		tagService: services.NewTagService(repo),
	}
}

// extractTitleFromContent is now provided by the markdown package

// renderMarkdown is now provided by the markdown package

// Routes returns the router for item endpoints
func (h *ItemHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/{type}", h.createItem)
	r.Get("/{type}", h.listItems)
	r.Get("/{type}/{id}", h.viewItem)
	r.Get("/{type}/{id}/edit", h.editItem)
	r.Put("/{type}/{id}/content", h.updateContent)
	r.Delete("/{type}/{id}", h.deleteItem)
	r.Get("/tags/{tag}", h.listItemsByTag)

	return r
}

// deleteItem handles deletion of an item
func (h *ItemHandler) deleteItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemType := models.ItemType(chi.URLParam(r, "type"))

	item, _, err := h.repo.LoadItem(id, itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Store the previous tags before deletion
	previousTags := make([]string, len(item.Tags))
	copy(previousTags, item.Tags)

	// Delete the item
	if err := h.repo.DeleteItem(item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update tag associations - since the item is deleted, pass empty tags list
	// This will effectively remove the item from all its previous tags
	item.Tags = []string{} // Clear tags since item is deleted
	if err := h.tagService.UpdateItemTags(item, previousTags); err != nil {
		// Just log the error, don't fail the deletion
		log.Printf("Error updating tags after deletion: %v", err)
	}

	// Redirect back to list view
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

// viewItem returns the view interface for an item
func (h *ItemHandler) viewItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemType := models.ItemType(chi.URLParam(r, "type"))

	item, content, err := h.repo.LoadItem(id, itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// If the item doesn't have a title, extract it
	if item.Title == "" {
		item.Title = md.ExtractTitleFromContent(content, string(itemType))
		if item.Title == "" {
			item.Title = item.ID
		}
		// Save the updated metadata
		h.repo.SaveItem(item, "")
	}

	w.Header().Set("Content-Type", "text/html")

	// Breadcrumb data
	breadcrumb := fmt.Sprintf(`
		<a href="/" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0 inline-flex items-center" hx-boost="true">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"></path>
            </svg>
        </a>
		<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
		<a href="/%ss" hx-boost="true" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0 inline-flex items-center">%ss</a>
		<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
		<span class="text-gray-600 dark:text-gray-300 truncate">%s</span>
	`, itemType, strings.Title(string(itemType)), item.Title)

	// Generate HTML
	contentHTML := md.Render(content)

	// Format tags
	tags := "None"
	if len(item.Tags) > 0 {
		tags = strings.Join(item.Tags, ", ")
	}

	// Metadata table HTML for sidebar
	metadataTable := fmt.Sprintf(`
	<div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700 mb-4">
		<h3 class="text-lg font-semibold mb-3 dark:text-gray-200">Metadata</h3>
		<table class="metadata-table class-item-metadata w-full">
			<tr>
				<th class="dark:text-gray-300">ID</th>
				<td class="dark:text-gray-200"><span class="font-mono">%s</span></td>
			</tr>
			<tr>
				<th class="dark:text-gray-300">Type</th>
				<td class="dark:text-gray-200">%s</td>
			</tr>
			<tr>
				<th class="dark:text-gray-300">Created</th>
				<td class="dark:text-gray-200">%s</td>
			</tr>
			<tr>
				<th class="dark:text-gray-300">Modified</th>
				<td class="dark:text-gray-200">%s</td>
			</tr>
			<tr>
				<th class="dark:text-gray-300">Tags</th>
				<td class="dark:text-gray-200">%s</td>
			</tr>`,
		item.ID,
		strings.Title(string(itemType)),
		item.Created.Format("Jan 2, 2006 3:04 PM"),
		item.Modified.Format("Jan 2, 2006 3:04 PM"),
		tags)

	// Add type-specific fields to metadata table
	switch itemType {
	case models.TypeBookmark:
		metadataTable += fmt.Sprintf(`
		<tr>
			<th class="dark:text-gray-300">URL</th>
			<td class="dark:text-gray-200"><a href="%s" target="_blank" class="text-blue-600 dark:text-blue-400 hover:underline">%s</a></td>
		</tr>`,
			item.URL, item.URL)
	case models.TypeTask:
		statusClass := "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
		statusText := "Todo"
		if item.Status == models.TaskStatusDone {
			statusClass = "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
			statusText = "Done"
		}
		metadataTable += fmt.Sprintf(`
		<tr>
			<th class="dark:text-gray-300">Status</th>
			<td class="dark:text-gray-200">
				<span class="inline-block px-2 py-1 text-xs rounded %s">%s</span>
			</td>
		</tr>`,
			statusClass, statusText)
	case models.TypeFile:
		metadataTable += fmt.Sprintf(`
		<tr>
			<th class="dark:text-gray-300">Filename</th>
			<td class="dark:text-gray-200">%s</td>
		</tr>`,
			item.Filename)
	}

	// Close the table and container
	metadataTable += `
		</table>
	</div>
	`

	// Create actions sidebar section with Edit and Delete buttons
	actionsSidebar := fmt.Sprintf(`
	<div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700 mb-4">
		<div class="flex justify-between items-center mb-3">
			<h3 class="text-lg font-semibold dark:text-gray-200">Actions</h3>
		</div>
		<div class="flex space-x-2 class-item-actions">
			<button
				class="flex-1 px-3 py-2 bg-blue-100 text-blue-800 dark:bg-blue-800 dark:text-blue-100 rounded hover:bg-blue-200 dark:hover:bg-blue-700 class-item-edit"
				hx-get="/api/items/%s/%s/edit"
				hx-target="#content"
				hx-swap="innerHTML"
				hx-push-url="/items/%s/%s/edit"
			>
				Edit
			</button>
			<button 
				class="flex-1 px-3 py-2 bg-red-100 text-red-800 dark:bg-red-800 dark:text-red-100 rounded hover:bg-red-200 dark:hover:bg-red-700 class-item-delete"
				hx-delete="/api/items/%s/%s"
				hx-target="#content"
				hx-swap="innerHTML"
				hx-confirm="Are you sure you want to delete this item?"
			>
				Delete
			</button>
		</div>
	</div>
	`,
		itemType, item.ID, itemType, item.ID, itemType, item.ID)

	tmpl := `
	<div id="content-with-sidebar" class="flex flex-col lg:flex-row lg:space-x-6 min-h-full flex-1">
		<div class="w-full lg:w-2/3 flex flex-col flex-shrink min-h-0">
			<div class="space-y-6 class-item-detail flex-grow flex flex-col">
				<div class="prose max-w-none bg-white dark:bg-gray-800 p-6 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm class-item-content flex-grow">
					%s
				</div>
			</div>
		</div>
		
		<div class="w-full lg:w-1/3 mt-6 lg:mt-0 flex-shrink-0">
			%s
			%s
		</div>
	</div>`

	// Update breadcrumb via HTMX
	fmt.Fprintf(w, `<div hx-swap-oob="innerHTML:#breadcrumb" class="flex items-center gap-2">%s</div>`, breadcrumb)

	fmt.Fprintf(w, tmpl,
		contentHTML,
		actionsSidebar,
		metadataTable)
}

// listItems returns a list of items of a given type
func (h *ItemHandler) listItems(w http.ResponseWriter, r *http.Request) {
	itemType := models.ItemType(chi.URLParam(r, "type"))

	items, err := h.repo.ListItems(itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Breadcrumb for list view
	breadcrumb := fmt.Sprintf(`
		<a href="/" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0 inline-flex items-center" hx-boost="true">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"></path>
            </svg>
        </a>
		<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
		<span class="text-gray-600 dark:text-gray-300">%ss</span>
	`, strings.Title(string(itemType)))

	w.Header().Set("Content-Type", "text/html")

	// Update breadcrumb via HTMX
	fmt.Fprintf(w, `<div hx-swap-oob="innerHTML:#breadcrumb" class="flex items-center gap-2">%s</div>`, breadcrumb)

	// Table header that matches the design with title and create button
	fmt.Fprintf(w, `
	<div class="flex justify-between items-center mb-6">
		<h1 class="text-2xl font-bold class-page-title">%ss</h1>
		<button 
			class="px-3 py-1 bg-indigo-600 text-white rounded hover:bg-indigo-700 dark:bg-indigo-700 dark:hover:bg-indigo-800 class-create-item"
			hx-post="/api/items/%s"
			hx-target="#content"
		>
			Create %s
		</button>
	</div>
	<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden class-items-list">
		<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
			<thead class="bg-gray-50 dark:bg-gray-900">
				<tr>
					<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 60%%;">Title</th>
					<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 20%%;">Modified</th>
					<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 20%%;">Actions</th>
				</tr>
			</thead>
			<tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700 class-items-rows">
	`, strings.Title(string(itemType)), itemType, strings.Title(string(itemType)))

	if len(items) == 0 {
		fmt.Fprintf(w, `
		<tr>
			<td colspan="3" class="px-6 py-4 whitespace-nowrap text-sm text-center text-gray-500 dark:text-gray-400">
				No items found. Create your first %s to get started.
			</td>
		</tr>
		`, itemType)
	}

	for _, item := range items {
		title := item.Title

		// Load content only if we need to extract title
		if title == "" {
			_, rawContent, err := h.repo.LoadItem(item.ID, itemType)
			if err == nil {
				title = md.ExtractTitleFromContent(rawContent, string(itemType))
				// Update metadata if title was extracted
				if title != "" {
					item.Title = title
					h.repo.SaveItem(item, "")
				}
			}
		}

		// If still no title, use ID
		if title == "" {
			title = item.ID
		}

		fmt.Fprintf(w, `
		<tr class="hover:bg-gray-50 dark:hover:bg-gray-700 class-item-row">
			<td class="px-6 py-4 whitespace-nowrap">
				<a 
					href="/items/%s/%s"
					class="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 class-item-title"
					hx-get="/api/items/%s/%s"
					hx-target="#content"
					hx-swap="innerHTML"
					hx-push-url="/items/%s/%s"
				>%s</a>
			</td>
			<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 class-item-modified">
				%s
			</td>
			<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 space-x-2 class-item-actions">
				<button 
					class="text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 class-item-edit"
					hx-get="/api/items/%s/%s/edit"
					hx-target="#content"
					hx-swap="innerHTML"
					hx-push-url="/items/%s/%s/edit"
				>
					Edit
				</button>
				<button 
					class="text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 class-item-delete"
					hx-delete="/api/items/%s/%s"
					hx-target="#content"
					hx-swap="innerHTML"
					hx-confirm="Are you sure you want to delete this item?"
				>
					Delete
				</button>
			</td>
		</tr>`,
			itemType, item.ID,
			itemType, item.ID,
			itemType, item.ID,
			title,
			item.Modified.Format("Jan 2, 2006 3:04 PM"),
			itemType, item.ID,
			itemType, item.ID,
			itemType, item.ID,
		)
	}

	// Close table and container
	fmt.Fprint(w, `
			</tbody>
		</table>
	</div>
	`)
}

// createItem handles creation of new items
func (h *ItemHandler) createItem(w http.ResponseWriter, r *http.Request) {
	itemType := models.ItemType(chi.URLParam(r, "type"))

	// Generate ID based on timestamp
	id := time.Now().UTC().Format("20060102150405")

	item := models.NewItem(itemType, id)

	// Save the new item with empty content
	if err := h.repo.SaveItem(item, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to edit view for the new item
	w.Header().Set("HX-Redirect", fmt.Sprintf("/items/%s/%s/edit", itemType, id))
	w.WriteHeader(http.StatusOK)
}

// updateContent handles updating the content of an item
func (h *ItemHandler) updateContent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemType := models.ItemType(chi.URLParam(r, "type"))

	// Get item
	item, _, err := h.repo.LoadItem(id, itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var content string

	// Determine if this is a form submission or JSON request
	contentType := r.Header.Get("Content-Type")
	shouldRedirect := false

	if strings.HasPrefix(contentType, "application/json") {
		// Handle JSON payload (keeping backward compatibility)
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		content = req.Content
	} else {
		// Handle form data (new approach)
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		content = r.FormValue("content")
		shouldRedirect = r.FormValue("redirect") == "true"
	}

	// Extract hashtags from content
	previousTags := item.Tags
	extractedTags := h.tagService.ExtractTags(content)

	// Update item tags
	if len(extractedTags) > 0 || len(previousTags) > 0 {
		item.Tags = extractedTags

		// Update tag indices
		if err := h.tagService.UpdateItemTags(item, previousTags); err != nil {
			http.Error(w, "Failed to update tag indices: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Auto-update title from content if needed
	newTitle := md.ExtractTitleFromContent(content, string(itemType))
	if newTitle != "" && (item.Title == "" || item.Title == item.ID) {
		item.Title = newTitle
		h.repo.SaveItem(item, "")
	}

	// Update modification time
	item.Modified = time.Now().UTC()

	// Save item with new content
	if err := h.repo.SaveItem(item, content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond based on request type
	if shouldRedirect {
		// For form submissions with HTMX, use HX-Redirect header
		// This will trigger client-side redirection without content nesting
		w.Header().Set("HX-Redirect", fmt.Sprintf("/items/%s/%s", itemType, id))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Trigger tag update event if tags have changed
	if !stringSlicesEqual(previousTags, item.Tags) {
		// Tags have changed, but we no longer need to trigger an event
		// The user will need to refresh the page to see updated tags
	}

	// For JSON/HTMX requests, return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"id":"%s","title":"%s"}`, item.ID, item.Title)
}

// stringSlicesEqual checks if two string slices are equal
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for faster lookup
	mapA := make(map[string]bool)
	for _, val := range a {
		mapA[val] = true
	}

	// Check if all elements in b are in a
	for _, val := range b {
		if !mapA[val] {
			return false
		}
	}

	return true
}

// editItem shows the editor for an item
func (h *ItemHandler) editItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemType := models.ItemType(chi.URLParam(r, "type"))

	item, content, err := h.repo.LoadItem(id, itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Breadcrumb data
	breadcrumb := fmt.Sprintf(`
		<a href="/" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0 inline-flex items-center" hx-boost="true">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"></path>
            </svg>
        </a>
		<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
		<a href="/%ss" hx-get="/api/items/%s" hx-target="#content" hx-push-url="/%ss" hx-boost="true" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0 inline-flex items-center">%ss</a>
		<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
		<span class="text-gray-600 dark:text-gray-300 truncate">%s</span>
	`, itemType, itemType, itemType, strings.Title(string(itemType)), item.Title)

	w.Header().Set("Content-Type", "text/html")

	// Update breadcrumb via HTMX
	fmt.Fprintf(w, `<div hx-swap-oob="innerHTML:#breadcrumb" class="flex items-center gap-2">%s</div>`, breadcrumb)

	tmpl := `
	<div class="space-y-4 class-editor-container flex-1 flex flex-col">
		<div class="flex justify-between items-center">
			<h1 class="text-2xl font-bold class-editor-title">Editing %s</h1>
			<div class="space-x-2 class-editor-actions">
				<button 
					type="submit"
					form="editor-form"
					id="save-button"
					class="px-3 py-1 bg-blue-100 text-blue-800 dark:bg-blue-800 dark:text-blue-100 rounded hover:bg-blue-200 dark:hover:bg-blue-700 class-editor-save"
				>
					Save
				</button>
				<button 
					type="button"
					class="inline-block px-3 py-1 bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200 rounded hover:bg-gray-200 dark:hover:bg-gray-600 class-editor-cancel"
					onclick="window.history.back()"
				>
					Cancel
				</button>
			</div>
		</div>
		
		<form 
			id="editor-form"
			hx-put="/api/items/%s/%s/content"
			hx-trigger="submit"
			hx-indicator="#saving-indicator"
			hx-on::before-request="disableSaveButton()"
			class="bg-white dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm class-editor-form flex-1 flex flex-col"
		>
			<div class="flex-1 flex flex-col class-editor-preview">
				<label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2" for="content">Content</label>
				<textarea 
					id="content" 
					name="content"
					class="w-full flex-1 p-4 border rounded font-mono text-sm bg-white dark:bg-gray-700 dark:text-white dark:border-gray-600 class-editor-textarea" 
				>%s</textarea>
			</div>
			<input type="hidden" name="redirect" value="true">
		</form>
		
		<!-- Saving indicator -->
		<div id="saving-indicator" class="fixed bottom-4 left-4 bg-blue-500 text-white px-4 py-2 rounded shadow class-save-indicator htmx-indicator">
			Saving...
		</div>
	</div>
	
	<script>
		function disableSaveButton() {
			const saveButton = document.getElementById('save-button');
			saveButton.disabled = true;
			saveButton.classList.add('opacity-50');
		}
		
		document.addEventListener('htmx:beforeSwap', function(evt) {
			// Check if this is a response from our content save endpoint
			if (evt.detail.requestConfig && 
				evt.detail.requestConfig.path && 
				evt.detail.requestConfig.path.includes('/content')) {
				
				// Prevent the default content swap
				evt.detail.shouldSwap = false;
				
				// Show saved indicator
				const savedIndicator = document.createElement('div');
				savedIndicator.className = 'fixed bottom-4 left-4 bg-green-500 text-white px-4 py-2 rounded shadow class-save-indicator';
				savedIndicator.textContent = 'Saved';
				document.body.appendChild(savedIndicator);
				setTimeout(() => savedIndicator.remove(), 2000);
				
				// Enable save button
				const saveButton = document.getElementById('save-button');
				if (saveButton) {
					saveButton.disabled = false;
					saveButton.classList.remove('opacity-50');
				}
				
				// Return to the previous page
				window.history.back();
			}
		});
	</script>
	`

	fmt.Fprintf(w, tmpl,
		strings.Title(string(itemType)),
		itemType, item.ID,
		content,
	)
}

// listItemsByTag returns a list of items with a specific tag
func (h *ItemHandler) listItemsByTag(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		http.Error(w, "Tag is required", http.StatusBadRequest)
		return
	}

	// Get items for this tag
	items, err := h.tagService.GetItemsByTag(tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort items by modified date (newest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Modified.After(items[j].Modified)
	})

	// Breadcrumb for tag view
	breadcrumb := fmt.Sprintf(`
		<a href="/" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0 inline-flex items-center" hx-boost="true">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"></path>
            </svg>
        </a>
		<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
		<span class="text-gray-600 dark:text-gray-300">tag</span>
		<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
		<span class="text-gray-600 dark:text-gray-300">#%s</span>
	`, tag)

	w.Header().Set("Content-Type", "text/html")

	// Update breadcrumb via HTMX
	fmt.Fprintf(w, `<div hx-swap-oob="innerHTML:#breadcrumb" class="flex items-center gap-2">%s</div>`, breadcrumb)

	// Table header that matches the design with title
	fmt.Fprintf(w, `
	<div class="flex justify-between items-center mb-6">
		<h1 class="text-2xl font-bold class-page-title">Items tagged #%s</h1>
	</div>
	<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden class-items-list">
		<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
			<thead class="bg-gray-50 dark:bg-gray-900">
				<tr>
					<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 50%%;">Title</th>
					<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 20%%;">Type</th>
					<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 20%%;">Modified</th>
				</tr>
			</thead>
			<tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700 class-items-rows">
	`, tag)

	if len(items) == 0 {
		fmt.Fprintf(w, `
		<tr>
			<td colspan="3" class="px-6 py-4 whitespace-nowrap text-sm text-center text-gray-500 dark:text-gray-400">
				No items found with tag #%s.
			</td>
		</tr>
		`, tag)
	}

	for _, item := range items {
		title := item.Title
		if title == "" {
			title = item.ID
		}

		fmt.Fprintf(w, `
		<tr class="hover:bg-gray-50 dark:hover:bg-gray-700 class-item-row">
			<td class="px-6 py-4 whitespace-nowrap">
				<a 
					href="/items/%s/%s"
					class="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 class-item-title"
					hx-get="/api/items/%s/%s"
					hx-target="#content"
					hx-swap="innerHTML"
					hx-push-url="/items/%s/%s"
				>%s</a>
			</td>
			<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 class-item-type">
				%s
			</td>
			<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 class-item-modified">
				%s
			</td>
		</tr>`,
			item.Type, item.ID,
			item.Type, item.ID,
			item.Type, item.ID,
			title,
			strings.Title(string(item.Type)),
			item.Modified.Format("Jan 2, 2006 3:04 PM"),
		)
	}

	// Close table and container
	fmt.Fprint(w, `
			</tbody>
		</table>
	</div>
	`)
}
