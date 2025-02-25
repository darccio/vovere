package handlers

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"vovere/internal/app/models"
	"vovere/internal/app/services"
)

// DashboardHandler handles dashboard-related functionality
type DashboardHandler struct {
	repo *services.Repository
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(repo *services.Repository) *DashboardHandler {
	return &DashboardHandler{
		repo: repo,
	}
}

// RecentItem represents an item displayed on the dashboard
type RecentItem struct {
	*models.Item
	Label string
}

// Routes returns the router for dashboard endpoints
func (h *DashboardHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/recent", h.getRecentItems)

	return r
}

// getRecentItems returns the most recent items across all types
func (h *DashboardHandler) getRecentItems(w http.ResponseWriter, r *http.Request) {
	// Get items of each type
	notes, _ := h.repo.ListItems(models.TypeNote)
	bookmarks, _ := h.repo.ListItems(models.TypeBookmark)
	tasks, _ := h.repo.ListItems(models.TypeTask)
	workstreams, _ := h.repo.ListItems(models.TypeWorkstream)

	// Create combined list with labels
	var allItems []RecentItem
	for _, item := range notes {
		allItems = append(allItems, RecentItem{Item: item, Label: "Note"})
	}
	for _, item := range bookmarks {
		allItems = append(allItems, RecentItem{Item: item, Label: "Bookmark"})
	}
	for _, item := range tasks {
		allItems = append(allItems, RecentItem{Item: item, Label: "Task"})
	}
	for _, item := range workstreams {
		allItems = append(allItems, RecentItem{Item: item, Label: "Workstream"})
	}

	// Sort by modified time, most recent first
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Modified.After(allItems[j].Modified)
	})

	// Limit to 20 most recent items
	if len(allItems) > 20 {
		allItems = allItems[:20]
	}

	w.Header().Set("Content-Type", "text/html")

	// Dashboard header
	fmt.Fprint(w, `
	<div class="class-dashboard">
		<h1 class="text-2xl font-bold mb-6 class-dashboard-title">All Items</h1>
		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden class-dashboard-recent">
			<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
				<thead class="bg-gray-50 dark:bg-gray-900">
					<tr>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 10%;">Type</th>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 50%;">Title</th>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 20%;">Last Modified</th>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider" style="width: 20%;">Actions</th>
					</tr>
				</thead>
				<tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
	`)

	// If no items, show message
	if len(allItems) == 0 {
		fmt.Fprint(w, `
			<tr>
				<td colspan="4" class="px-6 py-4 text-center text-sm text-gray-500 dark:text-gray-400">
					No items found. Create your first note, bookmark, task, or workstream to get started.
				</td>
			</tr>
		`)
	}

	// Display each item
	for _, item := range allItems {
		title := item.Title
		if title == "" {
			_, content, err := h.repo.LoadItem(item.ID, item.Type)
			if err == nil {
				title = extractTitleFromContent(content, item.Type)
				// Save the extracted title
				if title != "" {
					item.Title = title
					h.repo.SaveItem(item.Item, "")
				}
			}
		}

		// If still no title, use ID
		if title == "" {
			title = item.ID
		}

		// Type-specific styling
		typeClass := "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200"
		switch item.Type {
		case models.TypeNote:
			typeClass = "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
		case models.TypeBookmark:
			typeClass = "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200"
		case models.TypeTask:
			typeClass = "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
		case models.TypeWorkstream:
			typeClass = "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
		}

		fmt.Fprintf(w, `
		<tr class="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td class="px-6 py-4 whitespace-nowrap">
				<span class="inline-block px-2 py-1 text-xs rounded %s">%s</span>
			</td>
			<td class="px-6 py-4 whitespace-nowrap">
				<a 
					href="/items/%s/%s"
					class="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
					hx-get="/api/items/%s/%s"
					hx-target="#content"
					hx-swap="innerHTML"
					hx-push-url="/items/%s/%s"
				>%s</a>
			</td>
			<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
				%s
			</td>
			<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 space-x-2">
				<button 
					class="text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
					hx-get="/api/items/%s/%s/edit"
					hx-target="#content"
					hx-swap="innerHTML"
					hx-push-url="/items/%s/%s/edit"
				>
					Edit
				</button>
				<button 
					class="text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
					hx-delete="/api/items/%s/%s"
					hx-target="#content"
					hx-swap="innerHTML"
					hx-confirm="Are you sure you want to delete this item?"
				>
					Delete
				</button>
			</td>
		</tr>`,
			typeClass, item.Label,
			item.Type, item.ID,
			item.Type, item.ID,
			item.Type, item.ID,
			title,
			item.Modified.Format("Jan 2, 2006 3:04 PM"),
			item.Type, item.ID,
			item.Type, item.ID,
			item.Type, item.ID,
		)
	}

	// Close table and container
	fmt.Fprint(w, `
				</tbody>
			</table>
		</div>
	</div>
	`)
}
