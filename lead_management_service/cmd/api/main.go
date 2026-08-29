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

	handler "github.com/omnichannel/lead_management_service/internal/handler/http"
	redis_messaging "github.com/omnichannel/lead_management_service/internal/messaging/redis"
	"github.com/omnichannel/lead_management_service/internal/repository/postgres"
	"github.com/omnichannel/lead_management_service/internal/service"
	"github.com/omnichannel/lead_management_service/internal/worker"
	"github.com/redis/go-redis/v9"
)

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

	// Pillar 5: Event Bus (Real Redis Streams!)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	// Test the Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Successfully connected to Redis Streams on localhost:6379!")

	eventBus := redis_messaging.NewRedisEventBus(redisClient)

	leadSvc := service.NewLeadService(repo, eventBus)
	leadHandler := handler.NewLeadHandler(leadSvc)

	// Pillar 5: Start the Background Workers
	aiWorker := worker.NewAISummaryWorker(eventBus)
	go func() {
		if err := aiWorker.Start(context.Background()); err != nil {
			log.Printf("AI Summary Worker crashed: %v", err)
		}
	}()

	activityWorker := worker.NewActivityWorker(eventBus, repo)
	go func() {
		if err := activityWorker.Start(context.Background()); err != nil {
			log.Printf("Activity Worker crashed: %v", err)
		}
	}()

	// 3. Set up the Chi Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Register our API endpoints
	leadHandler.RegisterRoutes(r)

	// 4. Start the Web Server
	port := ":8082"
	fmt.Printf("Lead Management Service is running on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
