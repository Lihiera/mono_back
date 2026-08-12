package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Lihiera/mono_back/database"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type QueryString struct {
	Region    string   `form:"region"`
	Cuisines  []string `form:"cuisines"`
	PriceLow  int      `form:"priceLow"`
	PriceHigh int      `form:"priceHigh"`
	Source    string   `form:"source"`
	Page      int      `form:"page"`
}

type cache struct {
	sync.RWMutex
	data     map[string]map[string]interface{}
	category map[string]map[string]int
}

var dataCache = struct {
	MiCache   cache
	TabeCache cache
}{
	MiCache: cache{
		data:     make(map[string]map[string]interface{}),
		category: make(map[string]map[string]int),
	},
	TabeCache: cache{
		data:     make(map[string]map[string]interface{}),
		category: make(map[string]map[string]int),
	},
}

type Server struct {
	db *pgxpool.Pool
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	dbURL := os.Getenv("SUPABASE_DB_URL")
	if dbURL == "" {
		log.Fatal("SUPABASE_DB_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := newDBPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to initialize database pool: %v", err)
	}
	defer dbPool.Close()

	srv := &Server{db: dbPool}

	router := gin.Default()
	config := cors.Config{
		AllowOrigins: []string{"https://lihiera.github.io", "http://localhost:5173"},
		// AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))
	router.GET("/healthz", srv.healthz)
	router.GET("/readyz", srv.readyz)
	router.POST("/result", srv.getData)
	router.POST("/metadata", srv.getMeta)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func newDBPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	poolConfig.MinConns = 0
	poolConfig.MaxConns = 5
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.HealthCheckPeriod = time.Minute

	return pgxpool.NewWithConfig(ctx, poolConfig)
}

func (s *Server) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "mono-back",
	})
}

func (s *Server) readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var ok int
	if err := s.db.QueryRow(ctx, "SELECT 1").Scan(&ok); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unavailable",
			"database": "unavailable",
			"message":  "Database is temporarily unavailable.",
		})
		return
	}
	if ok != 1 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unavailable",
			"database": "unavailable",
			"message":  "Database readiness check returned an unexpected result.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"database": "ok",
	})
}

func (s *Server) getData(c *gin.Context) {
	var data QueryString
	if err := c.ShouldBind(&data); err != nil {
		writeInvalidRequest(c)
		return
	}
	if !isValidQuery(data) {
		writeInvalidRequest(c)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	DTO, err := database.FetchPageData(ctx, s.db, data.Region, data.Page, data.Source)
	if err != nil {
		log.Printf("failed to fetch page data: %v", err)
		writeDatabaseUnavailable(c)
		return
	}

	c.JSON(http.StatusOK, DTO)
}

func (s *Server) getMeta(c *gin.Context) {
	var data QueryString
	if err := c.ShouldBind(&data); err != nil {
		writeInvalidRequest(c)
		return
	}
	if !isValidMetadataQuery(data) {
		writeInvalidRequest(c)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	DTO, err := database.FetchMetaData(ctx, s.db, data.Region, data.Source)
	if err != nil {
		log.Printf("failed to fetch metadata: %v", err)
		writeDatabaseUnavailable(c)
		return
	}

	c.JSON(http.StatusOK, DTO)
}

func isValidQuery(data QueryString) bool {
	return data.Region != "" && data.Page >= 0 && isValidSource(data.Source)
}

func isValidMetadataQuery(data QueryString) bool {
	return data.Region != "" && isValidSource(data.Source)
}

func isValidSource(source string) bool {
	return source == "tabelog" || source == "michelin"
}

func writeInvalidRequest(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{
		"code":    "INVALID_REQUEST",
		"message": "The request parameters are invalid.",
	})
}

func writeDatabaseUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"code":    "DATABASE_UNAVAILABLE",
		"message": "Restaurant data is temporarily unavailable. Please try again later.",
	})
}
