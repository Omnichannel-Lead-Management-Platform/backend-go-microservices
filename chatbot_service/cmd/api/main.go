package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(dbURL string) {
	fmt.Println("Running database migrations...")
	
	// Wait a bit for postgres to be fully ready in docker compose
	time.Sleep(2 * time.Second)

	m, err := migrate.New(
		"file://migrations",
		dbURL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize migrations: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	fmt.Println("Migrations applied successfully!")
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		runMigrations(dbURL)
	} else {
		fmt.Println("WARNING: DATABASE_URL not set, skipping migrations.")
	}
	fmt.Println("Starting chatbot_service...")
	
	// Block forever for demo purposes
	select {}
}
