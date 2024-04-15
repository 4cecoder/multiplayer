package main

import (
	"log"
	"net/http"
	"os"

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

	r.Get("/", handlers.HandleRoot)
	r.Get("/ws", handlers.HandleWebSocket)

	// Serve static files
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	port := os.Getenv("PORT")
	if port == "" {
		log.Println("PORT environment variable not set")
		log.Println("Using default port 8080")
		port = "8080" // Default port if not specified
	}

	log.Printf("Server started on :%s", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatal(err)
		return
	}
}
