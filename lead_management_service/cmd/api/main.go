package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq" // Postgres driver

	"github.com/omnichannel/lead_management_service/internal/domain"
	handler "github.com/omnichannel/lead_management_service/internal/handler/http"
	"github.com/omnichannel/lead_management_service/internal/repository/postgres"
	"github.com/omnichannel/lead_management_service/internal/service"
)

// DummyEventPublisher just prints to the console instead of real Redis (for now)
type DummyEventPublisher struct{}
func (p *DummyEventPublisher) Publish(ctx context.Context, topic string, event domain.Event) error {
	fmt.Printf("[REDIS MOCK] Successfully published event '%s' to topic '%s'\n", event.Type, topic)
	return nil
}

func runMigrations(dbURL string) {
	fmt.Println("Running database migrations...")
	time.Sleep(2 * time.Second)

	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		log.Printf("Failed to initialize migrations: %v", err)
		return
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Printf("Failed to run migrations: %v", err)
	} else {
		fmt.Println("Migrations applied successfully!")
	}
}

func main() {
	// 1. Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default to a local dev DB if not provided by Docker
		dbURL = "postgres://user:password@localhost:5433/postgres?sslmode=disable"
	}
	runMigrations(dbURL)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 2. Wire up the Clean Architecture Layers
	repo := postgres.NewLeadRepository(db)
	publisher := &DummyEventPublisher{}
	leadSvc := service.NewLeadService(repo, publisher)
	leadHandler := handler.NewLeadHandler(leadSvc)

	// 3. Set up the Chi Router
	r := chi.NewRouter()
	r.Use(middleware.Logger) // Logs every incoming web request to the terminal
	
	// Register our API endpoints
	leadHandler.RegisterRoutes(r)

	// 4. Start the Web Server
	port := ":8082"
	fmt.Printf("Lead Management Service is running on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
