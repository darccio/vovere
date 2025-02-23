package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"vovere/internal/app/models"
	"vovere/internal/app/services"

	"github.com/go-chi/chi/v5"
)

// ItemHandler handles HTTP requests for items
type ItemHandler struct {
	repo *services.Repository
}

// NewItemHandler creates a new item handler
func NewItemHandler(repo *services.Repository) *ItemHandler {
	return &ItemHandler{
		repo: repo,
	}
}

// Routes returns the router for item endpoints
func (h *ItemHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/{type}", h.createItem)
	r.Get("/{type}", h.listItems)
	r.Get("/{type}/{id}", h.getItem)
	r.Get("/{type}/{id}/edit", h.editItem)
	r.Put("/{type}/{id}/content", h.updateContent)
	r.Delete("/{type}/{id}", h.deleteItem)

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

	if err := h.repo.DeleteItem(item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return empty response with 200 status
	w.WriteHeader(http.StatusOK)
}

// listItems returns a list of items of a given type
func (h *ItemHandler) listItems(w http.ResponseWriter, r *http.Request) {
	itemType := models.ItemType(chi.URLParam(r, "type"))

	items, err := h.repo.ListItems(itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	for _, item := range items {
		fmt.Fprintf(w, `
		<div class="flex items-center justify-between p-2 bg-white rounded border hover:bg-gray-50">
			<span class="text-sm text-gray-600">%s</span>
			<div class="flex space-x-2">
				<button 
					class="text-blue-500 hover:text-blue-700"
					hx-get="/api/items/%s/%s/edit"
					hx-target="#editor-container"
					hx-swap="innerHTML"
				>
					Edit
				</button>
				<button 
					class="text-red-500 hover:text-red-700"
					hx-delete="/api/items/%s/%s"
					hx-target="closest div"
					hx-swap="outerHTML"
					hx-confirm="Are you sure you want to delete this item?"
				>
					Delete
				</button>
			</div>
		</div>`,
			item.ID,
			item.Type,
			item.ID,
			item.Type,
			item.ID,
		)
	}
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

	// Return the editor interface
	w.Header().Set("Content-Type", "text/html")
	tmpl := `
	<div class="p-4 bg-white rounded-lg shadow">
		<div class="flex justify-between items-center mb-4">
			<h3 class="text-lg font-semibold">New %s</h3>
			<span class="text-sm text-gray-500">ID: %s</span>
		</div>
		<form
			hx-put="/api/items/%s/%s/content"
			hx-trigger="input changed delay:1s from:textarea"
			hx-swap="none"
		>
			<textarea
				class="w-full h-64 p-2 border rounded font-mono"
				placeholder="Start writing in markdown..."
				name="content"
			></textarea>
		</form>
	</div>`

	fmt.Fprintf(w, tmpl, itemType, item.ID, itemType, item.ID)
}

// getItem handles retrieval of existing items
func (h *ItemHandler) getItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemType := models.ItemType(chi.URLParam(r, "type"))

	item, content, err := h.repo.LoadItem(id, itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := struct {
		*models.Item
		Content string `json:"content,omitempty"`
	}{
		Item:    item,
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// editItem returns the editor interface for an item
func (h *ItemHandler) editItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemType := models.ItemType(chi.URLParam(r, "type"))

	item, content, err := h.repo.LoadItem(id, itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := `
	<div class="p-4 bg-white rounded-lg shadow">
		<div class="flex justify-between items-center mb-4">
			<h3 class="text-lg font-semibold">Edit %s</h3>
			<span class="text-sm text-gray-500">ID: %s</span>
		</div>
		<form
			hx-put="/api/items/%s/%s/content"
			hx-trigger="input changed delay:1s from:textarea"
			hx-swap="none"
		>
			<textarea
				class="w-full h-64 p-2 border rounded font-mono"
				name="content"
			>%s</textarea>
		</form>
	</div>`

	fmt.Fprintf(w, tmpl, itemType, item.ID, itemType, item.ID, content)
}

// updateContent handles content updates for an item
func (h *ItemHandler) updateContent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemType := models.ItemType(chi.URLParam(r, "type"))

	item, _, err := h.repo.LoadItem(id, itemType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if err := h.repo.UpdateContent(item, content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
