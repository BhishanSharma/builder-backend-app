package main

import (
    "github.com/gin-gonic/gin"
    "builder.ai/src/routes"
    "builder.ai/config"
)

func main() {
    config.ConnectDB()
    r := gin.Default()
    routes.UserRoutes(r)
    r.Run(":8080")
}