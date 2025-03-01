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
	"runtime/debug"

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
					// Print the stack trace of the recovered panic
					debug.PrintStack()
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

		// Mount the TagHandler for our new tag API
		r.Mount("/api/tags", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			tagHandler := handlers.NewTagHandler(repo)
			tagHandler.Routes().ServeHTTP(w, r)
		}))

		// API tag route for HTMX
		r.Get("/api/tags/{tag}", func(w http.ResponseWriter, r *http.Request) {
			// Get repository and create an item handler
			repo := services.RepositoryFromContext(r.Context())
			tag := chi.URLParam(r, "tag")

			// Create a new request with adjusted path to match the ItemHandler's route pattern
			newURL := fmt.Sprintf("/tags/%s", tag)
			newReq, _ := http.NewRequest(r.Method, newURL, r.Body)
			// Copy headers and other properties
			newReq.Header = r.Header
			newReq = newReq.WithContext(r.Context())

			// Use the item handler directly
			itemHandler := handlers.NewItemHandler(repo)
			itemHandler.Routes().ServeHTTP(w, newReq)
		})

		// Tags route - Main tags page
		r.Get("/tags", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())

			// Create tag handler and get the list HTML directly
			tagHandler := handlers.NewTagHandler(repo)
			tagListHTML, err := tagHandler.RenderTagList()
			if err != nil {
				http.Error(w, "Failed to render tag list: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Create breadcrumb HTML for the tags page
			breadcrumbHTML := `
			<a href="/" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0 inline-flex items-center" hx-boost="true">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"></path>
				</svg>
			</a>
			<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
			<span class="text-gray-600 dark:text-gray-300">Tags</span>
			`

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"PageTitle":      "Tags",
				"ViewType":       "tags",
				"TagListHTML":    template.HTML(tagListHTML),    // Pre-rendered HTML
				"BreadcrumbHTML": template.HTML(breadcrumbHTML), // Pre-rendered breadcrumb
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		// Tags route
		r.Get("/tags/{tag}", func(w http.ResponseWriter, r *http.Request) {
			repo := services.RepositoryFromContext(r.Context())
			repoName := getRepositoryName(repo.BasePath())
			tag := chi.URLParam(r, "tag")

			// Create breadcrumb HTML for tag detail page
			breadcrumbHTML := fmt.Sprintf(`
			<a href="/" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0 inline-flex items-center" hx-boost="true">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"></path>
				</svg>
			</a>
			<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
			<a href="/tags" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 flex-shrink-0" hx-boost="true">Tags</a>
			<span class="text-gray-500 dark:text-gray-400 flex-shrink-0">/</span>
			<span class="text-gray-600 dark:text-gray-300">%s</span>
			`, tag)

			data := map[string]interface{}{
				"RepositoryName": repoName,
				"PageTitle":      "Tag: #" + tag,
				"ViewType":       "list",
				"Tag":            tag,
				"BreadcrumbHTML": template.HTML(breadcrumbHTML),
			}

			if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})
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
