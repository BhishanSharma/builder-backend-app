package main

import (
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
    "time"
    "log"
    "context"
    "builder.ai/src/handlers"
    "builder.ai/src/routes"
    "builder.ai/config"
)

func main() {
    config.ConnectDB()
    config.CreateIndexes()

    // Store the handler first
    componentHandler := handlers.NewComponentHandler()
    err := componentHandler.CreateSearchIndexes(context.Background())
    if err != nil {
        log.Printf("Warning: Failed to create indexes: %v", err)
    }
    
    r := gin.Default()
    
    // Configure CORS
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))
    
    routes.SetupUserRoutes(r)
    routes.SetupComponentRoutes(r)
    routes.SetupStageRoutes(r)
    routes.SetupWorkflowRoutes(r)
    
    r.Run("localhost:8080")
}