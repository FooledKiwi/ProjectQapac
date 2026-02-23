package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dom1nux/qapac-api/internal/config"
	"github.com/dom1nux/qapac-api/internal/handler"
	"github.com/dom1nux/qapac-api/internal/middleware"
	"github.com/dom1nux/qapac-api/internal/routing"
	"github.com/dom1nux/qapac-api/internal/service"
	"github.com/dom1nux/qapac-api/internal/storage"
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

	api := router.Group("/api/v1")
	{
		api.GET("/stops/nearby", h.ListStopsNear)
		api.GET("/stops/:id", h.GetStop)
		api.GET("/routes/to-stop", h.GetRouteToStop)
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
