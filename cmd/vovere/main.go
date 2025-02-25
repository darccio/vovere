package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"vovere/internal/app/handlers"
	"vovere/internal/app/models"
	"vovere/internal/app/services"
)

var (
	port = flag.Int("port", 9090, "Port to run the server on")
)

// customErrorHandler wraps the notFound handler to use custom error pages
func customErrorHandler(tmpl *template.Template) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				// Recover from panics and display error page
				if err := recover(); err != nil {
					log.Printf("PANIC: %+v", err)
					ww.WriteHeader(http.StatusInternalServerError)
					tmpl.ExecuteTemplate(ww, "errors/500.html", nil)
				}

				// After the request is done, check status code and render error pages for 404/500
				switch ww.Status() {
				case http.StatusNotFound:
					tmpl.ExecuteTemplate(w, "errors/404.html", nil)
				case http.StatusInternalServerError:
					tmpl.ExecuteTemplate(w, "errors/500.html", nil)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// repositoryMiddleware ensures a repository is selected
func repositoryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("repository")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/api/repository", http.StatusSeeOther)
			return
		}

		// Initialize repository service
		repo := services.NewRepository(cookie.Value)

		// Store repository service in context
		ctx := services.WithRepository(r.Context(), repo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getRepositoryName gets the name of the repository from config or path
func getRepositoryName(repoPath string) string {
	// Default repository name is the last part of the path
	repoName := filepath.Base(repoPath)

	// Try to load config file for custom name
	configPath := filepath.Join(repoPath, "config.json")
	if configFile, err := os.Open(configPath); err == nil {
		defer configFile.Close()

		// Parse config to get name
		var config handlers.RepositoryConfig
		if err := json.NewDecoder(configFile).Decode(&config); err == nil {
			if config.Name != "" {
				repoName = config.Name
			}
		}
	}

	return repoName
}

// itemHandler wraps the item handler with repository context
type itemHandler struct {
	tmpl *template.Template
}

func (h *itemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	repo := services.RepositoryFromContext(r.Context())
	handler := handlers.NewItemHandler(repo)
	handler.Routes().ServeHTTP(w, r)
}

// dashboardHandler wraps the dashboard handler with repository context
type dashboardHandler struct {
	tmpl *template.Template
}

func (h *dashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	repo := services.RepositoryFromContext(r.Context())
	handler := handlers.NewDashboardHandler(repo)
	handler.Routes().ServeHTTP(w, r)
}

func main() {
	flag.Parse()

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Load templates
	tmpl := template.Must(template.ParseFiles(
		"web/templates/index.html",
		"web/templates/repo-select.html",
		"web/templates/errors/404.html",
		"web/templates/errors/500.html",
	))

	// Add custom error handling
	r.Use(customErrorHandler(tmpl))

	// Static file server
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "web/static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(filesDir))))

	// Repository selection handler
	repoHandler := handlers.NewRepositoryHandler(tmpl)
	r.Mount("/api/repository", repoHandler.Routes())

	// Main application routes
	r.Group(func(r chi.Router) {
		// Add repository middleware
		r.Use(repositoryMiddleware)

		// Dashboard - shows recent items of all types
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"PageTitle":      "Home",
				"ViewType":       "dashboard",
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		// Type-specific listing pages
		r.Get("/notes", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"PageTitle":      "Notes",
				"ViewType":       "list",
				"ItemType":       models.TypeNote,
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		r.Get("/bookmarks", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"PageTitle":      "Bookmarks",
				"ViewType":       "list",
				"ItemType":       models.TypeBookmark,
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		r.Get("/tasks", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"PageTitle":      "Tasks",
				"ViewType":       "list",
				"ItemType":       models.TypeTask,
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		r.Get("/workstreams", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"PageTitle":      "Workstreams",
				"ViewType":       "list",
				"ItemType":       models.TypeWorkstream,
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		// Item detail routes
		r.Get("/items/{type}/{id}", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"ViewType":       "detail",
				"ItemID":         chi.URLParam(r, "id"),
				"ItemType":       chi.URLParam(r, "type"),
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		r.Get("/items/{type}/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"ViewType":       "edit",
				"ItemID":         chi.URLParam(r, "id"),
				"ItemType":       chi.URLParam(r, "type"),
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		// API routes
		r.Mount("/api/items", &itemHandler{tmpl: tmpl})
		r.Mount("/api/dashboard", &dashboardHandler{tmpl: tmpl})
	})

	// Handle 404 for undefined routes
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		tmpl.ExecuteTemplate(w, "errors/404.html", nil)
	})

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
