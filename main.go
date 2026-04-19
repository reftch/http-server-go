package main

import (
	"log"
	"time"

	"reftch.com/http-server/handler"
	"reftch.com/http-server/server"
)

func main() {
	before := time.Now()

	// Create router
	router := server.NewSimpleRouter()

	// Register routes
	router.GET("/", handler.HomeHandler)
	router.GET("/health", handler.HealthHandler)

	// Create server
	s := server.NewServer(router)

	// Start server
	if err := s.Start(":8080"); err != nil {
		log.Fatal(err)
	}

	after := time.Now()
	startupMicros := after.Sub(before).Microseconds()
	log.Printf("server started at on http://localhost:8080, startup time %d µs", startupMicros)

	// Keep main goroutine alive
	select {}
}
