package handlers

import (
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

// RepositoryHandler handles repository selection and management
type RepositoryHandler struct {
	tmpl *template.Template
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(tmpl *template.Template) *RepositoryHandler {
	return &RepositoryHandler{
		tmpl: tmpl,
	}
}

// Routes returns the router for repository endpoints
func (h *RepositoryHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.showSelection)
	r.Post("/select", h.selectRepository)
	r.Get("/select", h.selectRepository) // For recent repos

	return r
}

// showSelection shows the repository selection screen
func (h *RepositoryHandler) showSelection(w http.ResponseWriter, r *http.Request) {
	// Check if repository is already selected
	if cookie, err := r.Cookie("repository"); err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := h.tmpl.ExecuteTemplate(w, "repo-select.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// selectRepository handles repository selection
func (h *RepositoryHandler) selectRepository(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	if path == "" {
		http.Redirect(w, r, "/api/repository?error="+url.QueryEscape("Repository path is required"), http.StatusSeeOther)
		return
	}

	// Ensure path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory if it doesn't exist
			if err := os.MkdirAll(path, 0755); err != nil {
				http.Redirect(w, r, "/api/repository?error="+url.QueryEscape("Failed to create repository directory"), http.StatusSeeOther)
				return
			}
		} else {
			http.Redirect(w, r, "/api/repository?error="+url.QueryEscape("Invalid repository path"), http.StatusSeeOther)
			return
		}
	} else if !info.IsDir() {
		http.Redirect(w, r, "/api/repository?error="+url.QueryEscape("Path must be a directory"), http.StatusSeeOther)
		return
	}

	// Create required subdirectories
	dirs := []string{
		filepath.Join(path, ".meta", "notes"),
		filepath.Join(path, ".meta", "bookmarks"),
		filepath.Join(path, ".meta", "tasks"),
		filepath.Join(path, ".meta", "workstreams"),
		filepath.Join(path, "notes"),
		filepath.Join(path, "bookmarks"),
		filepath.Join(path, "tasks"),
		filepath.Join(path, "files"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			http.Redirect(w, r, "/api/repository?error="+url.QueryEscape("Failed to create repository structure"), http.StatusSeeOther)
			return
		}
	}

	// Set repository cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "repository",
		Value:    path,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to main application
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
