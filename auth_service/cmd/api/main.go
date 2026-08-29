package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

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
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

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

	ab.Config.Core.Mailer = auth.NewSMTPMailer()
	ab.Config.Core.Logger = defaults.NewLogger(os.Stdout)

	// Define the mount path so Authboss internally registers the correct URLs
	ab.Config.Paths.Mount = "/api/auth"

	// Use API mode (JSON responses instead of HTML redirects)
	ab.Config.Modules.LogoutMethod = "POST"
	ab.Config.Modules.RecoverLoginAfterRecovery = true
	ab.Config.Modules.RegisterPreserveFields = []string{"name", "company_name", "invite_token"}
	// Initialize Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	blacklister := &RedisTokenBlacklister{client: rdb}

	ab.Config.Storage.Server = auth.NewServerStorer(querier)
	ab.Config.Storage.SessionState = auth.NewJWTReadWriter(blacklister)
	ab.Config.Storage.CookieState = auth.NewJWTReadWriter(blacklister)

	if err := ab.Init(); err != nil {
		log.Fatalf("Authboss init failed: %v", err)
	}
	
	// Force override in case ab.Init() overwrites it in API mode
	ab.Core.Responder = customResponder
	ab.Core.Redirector = customResponder

	// Inject User Permissions and Workspace info into the JWT Session upon Login/Registration
	injectSessionClaims := func(w http.ResponseWriter, r *http.Request, handled bool) (bool, error) {
		u, err := ab.CurrentUser(r)
		if err == nil && u != nil {
			if user, ok := u.(*auth.User); ok {
				authboss.PutSession(w, "workspace_id", user.WorkspaceID.String())
				if user.RoleID.Valid {
					authboss.PutSession(w, "role_id", user.RoleID.UUID.String())
				}
				if len(user.Permissions) > 0 {
					authboss.PutSession(w, "permissions", strings.Join(user.Permissions, ","))
				}
			}
		}
		return false, nil
	}
	ab.Events.After(authboss.EventAuth, injectSessionClaims)
	ab.Events.After(authboss.EventRegister, injectSessionClaims)

	// Blacklist JWT on logout
	ab.Events.After(authboss.EventLogout, func(w http.ResponseWriter, r *http.Request, handled bool) (bool, error) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			// We blacklist it for 24 hours (the max TTL of our JWTs)
			blacklister.Blacklist(r.Context(), tokenString, 24*time.Hour)
		}
		return false, nil
	})

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	
	// CORS middleware
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Rate limiting: 100 requests per minute per IP
	router.Use(httprate.LimitByIP(100, 1*time.Minute))
	
	// Mount Authboss
	router.Use(ab.LoadClientStateMiddleware)
	// chi.Mount does not alter r.URL.Path for standard http.Handlers, so we MUST strip the prefix manually
	// before passing it to Authboss's internal defaults.Router (which expects exact matches like "/login")
	router.Mount("/api/auth", http.StripPrefix("/api/auth", ab.Config.Core.Router))

	// Protected routes
	router.Group(func(r chi.Router) {
		r.Use(authboss.Middleware(ab, true, false, false))
		r.Get("/api/auth/me", auth.MeHandler(ab))
		r.Get("/api/auth/introspect", auth.IntrospectHandler(ab))
		
		// Admin-only routes
		r.Group(func(adminRoutes chi.Router) {
			adminRoutes.Use(auth.RequirePermission(ab, "users:manage"))
			adminRoutes.Post("/api/auth/invite", auth.GenerateInviteHandler(ab))
			adminRoutes.Get("/api/auth/roles", auth.GetRolesHandler(ab, querier))
			adminRoutes.Post("/api/auth/roles", auth.CreateRoleHandler(ab, querier))
			adminRoutes.Get("/api/auth/permissions", auth.GetPermissionsHandler(ab, querier))
			adminRoutes.Get("/api/auth/roles/{id}/permissions", auth.GetRolePermissionsHandler(ab, querier))
			adminRoutes.Put("/api/auth/roles/{id}/permissions", auth.UpdateRolePermissionsHandler(ab, querier))
			adminRoutes.Delete("/api/auth/roles/{id}", auth.DeleteRoleHandler(ab, querier))
			adminRoutes.Get("/api/auth/users", auth.ListUsersHandler(ab, querier))
			adminRoutes.Put("/api/auth/users/{id}/role", auth.UpdateUserRoleHandler(ab, querier))
			adminRoutes.Get("/api/auth/workspace", auth.GetWorkspaceHandler(ab, querier))
			adminRoutes.Put("/api/auth/workspace", auth.UpdateWorkspaceHandler(ab, querier))
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Starting auth_service on port %s...\n", port)
	
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

type RedisTokenBlacklister struct {
	client *redis.Client
}

func (r *RedisTokenBlacklister) Blacklist(ctx context.Context, token string, expiration time.Duration) error {
	return r.client.Set(ctx, token, "blacklisted", expiration).Err()
}

func (r *RedisTokenBlacklister) IsBlacklisted(ctx context.Context, token string) bool {
	res, err := r.client.Get(ctx, token).Result()
	return err == nil && res == "blacklisted"
}
