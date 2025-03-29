package main

import (
	"log"
	"reward-service/cmd/api"
)

func main() {
	log.Println("Starting reward service")

	server, err := api.NewServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Настройка роутов (можно вынести в отдельный метод)
	setupRoutes(server)

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
