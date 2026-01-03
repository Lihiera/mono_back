package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Lihiera/mono_back/database"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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

func main() {
	router := gin.Default()
	config := cors.Config{
		// AllowOrigins:     []string{"https://lihiera.github.io", "http://localhost:5173"},
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))
	router.POST("/result", getData)
	router.POST("/metadata", getMeta)
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port) // listens on 0.0.0.0:8080 by default
}

func getData(c *gin.Context) {
	var data QueryString
	c.Bind(&data)
	DTO := database.FetchPageData(c, data.Region, data.Page, data.Source)

	c.JSON(200, DTO)
}

func getMeta(c *gin.Context) {
	var data QueryString
	c.Bind(&data)
	DTO := database.FetchMetaData(c, data.Region, data.Source)
	fmt.Print(DTO.Count)
	c.JSON(200, DTO)
}
