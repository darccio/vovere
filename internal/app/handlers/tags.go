package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"vovere/internal/app/services"

	"github.com/go-chi/chi/v5"
)

// TagHandler handles HTTP requests for tags
type TagHandler struct {
	tagService *services.TagService
}

// NewTagHandler creates a new tag handler
func NewTagHandler(repo *services.Repository) *TagHandler {
	return &TagHandler{
		tagService: services.NewTagService(repo),
	}
}

// Tag represents a tag with its count for the API response
type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Routes returns the router for tag endpoints
func (h *TagHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.getAllTags)

	return r
}

// getAllTags returns all tags as JSON
func (h *TagHandler) getAllTags(w http.ResponseWriter, r *http.Request) {
	// Get tag statistics (tag -> count)
	stats, err := h.tagService.GetTagStatistics()
	if err != nil {
		http.Error(w, "Failed to get tag statistics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to a slice of Tag objects for the response
	tags := make([]Tag, 0, len(stats))
	for name, count := range stats {
		tags = append(tags, Tag{
			Name:  name,
			Count: count,
		})
	}

	// Sort by name (alphabetically)
	sort.Slice(tags, func(i, j int) bool {
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})

	// Return as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tags); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// plural returns "s" if n != 1, otherwise returns empty string
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// RenderTagList returns the HTML for the tag list as a string
func (h *TagHandler) RenderTagList() (string, error) {
	// Get tag statistics
	stats, err := h.tagService.GetTagStatistics()
	if err != nil {
		return "", fmt.Errorf("failed to get tag statistics: %w", err)
	}

	// Convert to a slice for sorting
	tags := make([]Tag, 0, len(stats))
	for name, count := range stats {
		tags = append(tags, Tag{
			Name:  name,
			Count: count,
		})
	}

	// Sort alphabetically by name
	sort.Slice(tags, func(i, j int) bool {
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})

	// Use a buffer to build the HTML
	var buf bytes.Buffer

	// Table header
	fmt.Fprintf(&buf, `
	<div class="flex justify-between items-center mb-6">
		<h1 class="text-2xl font-bold class-page-title">Tags</h1>
	</div>
	<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden class-items-list">
		<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
			<thead class="bg-gray-50 dark:bg-gray-900">
				<tr>
					<th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Tag</th>
					<th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Items</th>
				</tr>
			</thead>
			<tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
	`)

	// No tags message
	if len(tags) == 0 {
		fmt.Fprintf(&buf, `
			<tr>
				<td colspan="2" class="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 text-center">
					No tags found
				</td>
			</tr>
		`)
	}

	// List of tags
	for _, tag := range tags {
		fmt.Fprintf(&buf, `
			<tr class="hover:bg-gray-50 dark:hover:bg-gray-700">
				<td class="px-6 py-4 whitespace-nowrap">
					<a href="/tags/%s" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300">
						#%s
					</a>
				</td>
				<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
					%d item%s
				</td>
			</tr>
		`, tag.Name, tag.Name, tag.Count, plural(tag.Count))
	}

	// Close tags table
	fmt.Fprintf(&buf, `
			</tbody>
		</table>
	</div>
	`)

	return buf.String(), nil
}
