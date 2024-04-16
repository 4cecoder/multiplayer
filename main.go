package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/4cecoder/multiplayer/handlers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(LoggingMiddleware)

	r.Get("/", handlers.HandleRoot)
	r.Get("/ws", handlers.ServeWebSocket)

	// Serve static files
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	log.Printf("Server started on :%s", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start timer
		start := time.Now()

		// Process request
		next.ServeHTTP(w, r)

		// Stop timer
		elapsed := time.Since(start)

		// Log details of the request
		log.Printf("%s %s %s", r.Method, r.RequestURI, elapsed)
	})
}
