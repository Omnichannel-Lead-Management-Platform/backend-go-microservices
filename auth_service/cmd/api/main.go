package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/aarondl/authboss/v3/defaults"
	_ "github.com/aarondl/authboss/v3/auth"
	_ "github.com/aarondl/authboss/v3/register"
	_ "github.com/aarondl/authboss/v3/recover"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/omnichannel/auth_service/internal/auth"
	"github.com/omnichannel/auth_service/internal/db"
)

func runMigrations(dbURL string) {
	fmt.Println("Running database migrations...")
	time.Sleep(2 * time.Second)

	m, err := migrate.New("file://migrations", dbURL)
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
	if dbURL == "" {
		dbURL = "postgresql://postgres:password@localhost:5433/postgres?sslmode=disable"
	}
	runMigrations(dbURL)

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("cannot connect to db: %v", err)
	}
	defer conn.Close()

	querier := db.New(conn)
	
	// Initialize Authboss
	ab := authboss.New()
	
	// ViewRenderer must be set before SetCore so Responder and Redirector can use it
	ab.Config.Core.ViewRenderer = defaults.JSONRenderer{}
	ab.Config.Core.MailRenderer = defaults.JSONRenderer{}
	
	defaults.SetCore(&ab.Config, true, false)
	
	// Set custom Responder and Redirector AFTER SetCore, so SetCore doesn't overwrite it
	customResponder := auth.NewAPIResponder()
	ab.Config.Core.Responder = customResponder
	ab.Config.Core.Redirector = customResponder
	
	// Ensure JSON body reader extracts our arbitrary fields during registration
	if reader, ok := ab.Config.Core.BodyReader.(*defaults.HTTPBodyReader); ok {
		reader.Whitelist = map[string][]string{
			"register": {"name", "company_name", "invite_token"},
		}
	}

	ab.Config.Core.Mailer = auth.NewFileMailer("./emails")
	ab.Config.Core.Logger = defaults.NewLogger(os.Stdout)

	// Define the mount path so Authboss internally registers the correct URLs
	ab.Config.Paths.Mount = "/api/auth"

	// Use API mode (JSON responses instead of HTML redirects)
	ab.Config.Modules.LogoutMethod = "POST"
	ab.Config.Modules.RecoverLoginAfterRecovery = true
	ab.Config.Modules.RegisterPreserveFields = []string{"name", "company_name", "invite_token"}
	ab.Config.Storage.Server = auth.NewServerStorer(querier)
	ab.Config.Storage.SessionState = auth.NewJWTReadWriter()
	ab.Config.Storage.CookieState = auth.NewJWTReadWriter()

	if err := ab.Init(); err != nil {
		log.Fatalf("Authboss init failed: %v", err)
	}
	
	// Force override in case ab.Init() overwrites it in API mode
	ab.Core.Responder = customResponder
	ab.Core.Redirector = customResponder

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	
	// Mount Authboss
	router.Use(ab.LoadClientStateMiddleware)
	// chi.Mount does not alter r.URL.Path for standard http.Handlers, so we MUST strip the prefix manually
	// before passing it to Authboss's internal defaults.Router (which expects exact matches like "/login")
	router.Mount("/api/auth", http.StripPrefix("/api/auth", ab.Config.Core.Router))

	// Protected routes
	router.Group(func(r chi.Router) {
		r.Use(authboss.Middleware(ab, true, false, false))
		r.Get("/api/auth/me", auth.MeHandler(ab))
		
		// Admin-only routes
		r.Group(func(adminRoutes chi.Router) {
			adminRoutes.Use(auth.RequireRole(ab, "admin"))
			adminRoutes.Post("/api/auth/invite", auth.GenerateInviteHandler(ab))
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Starting auth_service on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
