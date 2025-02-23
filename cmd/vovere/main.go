package main

import (
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
	"vovere/internal/app/services"
)

var (
	port = flag.Int("port", 9090, "Port to run the server on")
)

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

// itemHandler wraps the item handler with repository context
type itemHandler struct {
	tmpl *template.Template
}

func (h *itemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	repo := services.RepositoryFromContext(r.Context())
	handler := handlers.NewItemHandler(repo)
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
	))

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

		// Serve index page
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})

		// API routes
		r.Mount("/api/items", &itemHandler{tmpl: tmpl})
	})

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
