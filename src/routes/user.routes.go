package routes

import (
    "github.com/gin-gonic/gin"
    "builder.ai/src/handlers"
)

func UserRoutes(r *gin.Engine) {
    userHandler := handlers.NewUserHandler()
    
    api := r.Group("/api/v1")
    {
        users := api.Group("/users")
        {
            users.GET("", userHandler.GetAll)
            users.GET("/:id", userHandler.GetByID)
            users.POST("", userHandler.Create)
            users.PUT("/:id", userHandler.Update)
            users.DELETE("/:id", userHandler.Delete)
            users.GET("/search", userHandler.SearchByName)
        }
    }
}