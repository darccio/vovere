package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

// RepositoryConfig represents configuration for a repository
type RepositoryConfig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

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
	r.Get("/config", h.getConfig)
	r.Get("/close", h.closeRepository) // Add endpoint for closing repository

	return r
}

// getConfig returns repository configuration
func (h *RepositoryHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	// Get repository from cookie
	cookie, err := r.Cookie("repository")
	if err != nil {
		http.Error(w, "Repository not selected", http.StatusBadRequest)
		return
	}

	repoPath := cookie.Value
	if repoPath == "" {
		http.Error(w, "Repository path is empty", http.StatusBadRequest)
		return
	}

	// Default config uses the last directory name
	config := RepositoryConfig{
		Name: filepath.Base(repoPath),
	}

	// Try to load config file
	configPath := filepath.Join(repoPath, "config.json")
	if configFile, err := os.Open(configPath); err == nil {
		defer configFile.Close()
		if err := json.NewDecoder(configFile).Decode(&config); err != nil {
			// Use default on error
		}
	}

	// Return config as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// showSelection shows the repository selection screen
func (h *RepositoryHandler) showSelection(w http.ResponseWriter, r *http.Request) {
	// Check if repository is already selected
	if cookie, err := r.Cookie("repository"); err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := h.tmpl.ExecuteTemplate(w, "repo-select.html", nil); err != nil {
		// Use the error page template
		w.WriteHeader(http.StatusInternalServerError)
		h.tmpl.ExecuteTemplate(w, "errors/500.html", nil)
		return
	}
}

// selectRepository handles repository selection
func (h *RepositoryHandler) selectRepository(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	if path == "" {
		// TODO: pass errors through another way instead of query parameters.
		http.Redirect(w, r, "/api/repository?error="+url.QueryEscape("Repository path is required"), http.StatusSeeOther)
		return
	}

	// Ensure path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, return error instead of creating it
			http.Redirect(w, r, "/api/repository?error="+url.QueryEscape("Repository directory does not exist"), http.StatusSeeOther)
			return
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

	// Create default config.json if it doesn't exist
	// TODO: review this, it isn't really needed by default.
	configPath := filepath.Join(path, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := RepositoryConfig{
			Name:        filepath.Base(path),
			Description: "Vovere knowledge repository",
			Tags:        []string{},
		}

		configFile, err := os.Create(configPath)
		if err == nil {
			defer configFile.Close()
			json.NewEncoder(configFile).Encode(config)
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

	// Save to localStorage via script (for recent repositories)
	w.Header().Set("Content-Type", "text/html")
	script := `
	<script>
		// Get existing repos from localStorage or initialize empty array
		const recentRepos = JSON.parse(localStorage.getItem('recentRepos') || '[]');
		
		// Create new entry
		const newRepo = {
			path: "%s",
			lastAccessed: new Date().toISOString()
		};
		
		// Remove existing entry with same path
		const filteredRepos = recentRepos.filter(repo => repo.path !== newRepo.path);
		
		// Add to front of array and limit to 5 entries
		filteredRepos.unshift(newRepo);
		if (filteredRepos.length > 5) filteredRepos.pop();
		
		// Save back to localStorage
		localStorage.setItem('recentRepos', JSON.stringify(filteredRepos));
		localStorage.setItem('currentRepo', "%s");
		
		// Redirect to main page with theme preservation
		window.location.href = '/';
	</script>
	`
	fmt.Fprintf(w, script, path, path)
}

// closeRepository handles closing the current repository
func (h *RepositoryHandler) closeRepository(w http.ResponseWriter, r *http.Request) {
	// Clear the repository cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "repository",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete the cookie
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to repository selection page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
