package routes

import (
    "github.com/gin-gonic/gin"
    "builder.ai/src/handlers"
)

func SetupComponentRoutes(r *gin.Engine) {
    componentHandler := handlers.NewComponentHandler()
    
    api := r.Group("/api/v1")
    {
        components := api.Group("/components")
        {
            components.GET("", componentHandler.GetAll)              // Get all with optional filters
            components.GET("/:id", componentHandler.GetByID)         // Get by ID
            components.POST("", componentHandler.Create)             // Create new
            components.PUT("/:id", componentHandler.Update)          // Update
            components.DELETE("/:id", componentHandler.Delete)       // Delete
            components.GET("/search", componentHandler.SearchByName) // Search
            components.GET("/stats", componentHandler.GetStageStats) // Get stats
        }    
    }
}