package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/config"
	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/handler"
	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/middleware"
	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/routing"
	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/service"
	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBError represents a database-related error.
type DBError struct {
	Op  string
	Err error
}

func (e *DBError) Error() string {
	return fmt.Sprintf("db error during %q: %v", e.Op, e.Err)
}

func (e *DBError) Unwrap() error { return e.Err }

// App holds the application-level dependencies.
type App struct {
	DB     *pgxpool.Pool
	Router *gin.Engine
	cfg    *config.Config
}

// New initializes the application: connects to PostGIS, runs migrations,
// wires all domain dependencies, and configures the HTTP engine with routes.
func New(cfg *config.Config) (*App, error) {
	// --- Database pool ---
	poolCfg, err := pgxpool.ParseConfig(cfg.DBDSN)
	if err != nil {
		return nil, &DBError{Op: "parse_dsn", Err: err}
	}

	poolCfg.MaxConns = 20
	poolCfg.MaxConnLifetime = 30 * time.Second
	poolCfg.MaxConnIdleTime = 10 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, &DBError{Op: "connect", Err: err}
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, &DBError{Op: "ping", Err: err}
	}

	log.Println("database connection pool established")

	// --- Migrations ---
	if err := storage.RunMigrations(context.Background(), pool); err != nil {
		return nil, fmt.Errorf("app: run migrations: %w", err)
	}

	log.Println("database schema up to date")

	// --- Domain dependencies ---
	stopsRepo := storage.NewStopsRepository(pool)

	googleRouter := routing.NewGoogleRouter(cfg.GoogleAPIKey)
	cachedRouter := routing.NewCachedRouter(
		googleRouter,
		routing.NewPgCacheStore(pool),
		routing.WithLogger(log.Printf),
	)

	routingService := service.NewRoutingService(cachedRouter, stopsRepo)

	etaProvider := service.NewSimpleETAProvider()
	etaStore := service.NewPgETACacheStore(pool)
	etaService := service.NewETAService(etaProvider, etaStore)

	// Auth dependencies.
	usersRepo := storage.NewUsersRepository(pool)
	tokensRepo := storage.NewRefreshTokensRepository(pool)
	authService := service.NewAuthService(
		usersRepo, tokensRepo,
		cfg.JWTSecret,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)

	// Admin dependencies.
	vehiclesRepo := storage.NewVehiclesRepository(pool)
	alertsRepo := storage.NewAlertsRepository(pool)

	// Public dependencies.
	publicRoutesRepo := storage.NewPublicRoutesRepository(pool)
	positionsRepo := storage.NewVehiclePositionsRepository(pool)
	ratingsRepo := storage.NewRatingsRepository(pool)
	favoritesRepo := storage.NewFavoritesRepository(pool)

	// --- HTTP engine ---
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.Timeout(10 * time.Second))

	// Health check.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes.
	h := handler.New(stopsRepo, etaService, routingService)
	ah := handler.NewAuthHandler(authService)
	adminH := handler.NewAdminHandler(usersRepo, vehiclesRepo, alertsRepo)
	pubH := handler.NewPublicHandler(publicRoutesRepo, positionsRepo, alertsRepo, ratingsRepo, favoritesRepo)

	api := router.Group("/api/v1")
	{
		// Public endpoints (no auth required).
		api.GET("/stops/nearby", h.ListStopsNear)
		api.GET("/stops/:id", h.GetStop)
		api.GET("/routes/to-stop", h.GetRouteToStop)

		// Routes (public).
		api.GET("/routes", pubH.ListRoutes)
		api.GET("/routes/:id", pubH.GetRoute)
		api.GET("/routes/:id/vehicles", pubH.GetRouteVehicles)

		// Vehicle positions (public).
		api.GET("/vehicles/nearby", pubH.NearbyVehicles)
		api.GET("/vehicles/:id/position", pubH.GetVehiclePosition)

		// Alerts (public read).
		api.GET("/alerts", pubH.ListAlerts)
		api.GET("/alerts/:id", pubH.GetAlert)

		// Ratings (anonymous).
		api.POST("/ratings", pubH.CreateRating)

		// Favorites (anonymous, by device_id).
		api.GET("/favorites", pubH.ListFavorites)
		api.POST("/favorites", pubH.AddFavorite)
		api.DELETE("/favorites", pubH.RemoveFavorite)

		// Auth endpoints (no auth required to call these).
		auth := api.Group("/auth")
		{
			auth.POST("/login", ah.Login)
			auth.POST("/refresh", ah.Refresh)
			auth.POST("/logout", ah.Logout)
		}

		// Protected endpoints: driver role.
		driver := api.Group("/driver")
		driver.Use(middleware.JWTAuth(authService))
		driver.Use(middleware.RequireRole("driver", "admin"))
		{
			// Driver-specific endpoints will be registered here in Phase 5.
		}

		// Protected endpoints: admin role.
		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth(authService))
		admin.Use(middleware.RequireRole("admin"))
		{
			// User management.
			admin.POST("/users", adminH.CreateUser)
			admin.GET("/users", adminH.ListUsers)
			admin.GET("/users/:id", adminH.GetUser)
			admin.PUT("/users/:id", adminH.UpdateUser)
			admin.DELETE("/users/:id", adminH.DeactivateUser)

			// Vehicle management.
			admin.POST("/vehicles", adminH.CreateVehicle)
			admin.GET("/vehicles", adminH.ListVehicles)
			admin.GET("/vehicles/:id", adminH.GetVehicle)
			admin.PUT("/vehicles/:id", adminH.UpdateVehicle)
			admin.POST("/vehicles/:id/assign", adminH.AssignVehicle)

			// Alert management.
			admin.POST("/alerts", adminH.CreateAlert)
			admin.DELETE("/alerts/:id", adminH.DeleteAlert)
		}
	}

	return &App{
		DB:     pool,
		Router: router,
		cfg:    cfg,
	}, nil
}

// Shutdown gracefully closes the database pool.
func (a *App) Shutdown() {
	if a.DB != nil {
		a.DB.Close()
		log.Println("database connection pool closed")
	}
}
