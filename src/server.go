package main

import (
    "github.com/gin-gonic/gin"
    "builder.ai/src/routes"
    "builder.ai/config"
)

func main() {
    config.ConnectDB()
    config.CreateIndexes()
    
    r := gin.Default()
    
    routes.SetupUserRoutes(r)
    routes.SetupComponentRoutes(r)
    routes.SetupStageRoutes(r)

    r.Run("localhost:8080")
}