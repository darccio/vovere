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
	port     = flag.Int("port", 9090, "Port to run the server on")
	repoPath = flag.String("repo", "", "Path to the repository")
)

func main() {
	flag.Parse()

	if *repoPath == "" {
		log.Fatal("Repository path is required")
	}

	// Ensure repository path exists
	if err := os.MkdirAll(*repoPath, 0755); err != nil {
		log.Fatalf("Failed to create repository directory: %v", err)
	}

	// Initialize services
	repo := services.NewRepository(*repoPath)

	// Initialize handlers
	itemHandler := handlers.NewItemHandler(repo)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Load templates
	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))

	// Static file server
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "web/static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(filesDir))))

	// Serve index page
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// API routes
	r.Mount("/api/items", itemHandler.Routes())

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
