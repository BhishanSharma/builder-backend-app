package routes

import (
    "github.com/gin-gonic/gin"
    "builder.ai/src/handlers"
)

func SetupStageRoutes(r *gin.Engine) {
    componentHandler := handlers.NewComponentHandler()
    
    api := r.Group("/api/v1")
    {
        stages := api.Group("/stages")
        {
            stages.GET("/:stage/components", componentHandler.GetByStage)
        }    
    }
}