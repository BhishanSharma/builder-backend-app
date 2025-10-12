package routes

import (
    "github.com/gin-gonic/gin"
    "builder.ai/src/handlers"
)

func SetupWorkflowRoutes(r *gin.Engine) {
    workflowHandler := handlers.NewWorkflowHandler()
    
    api := r.Group("/api/v1")
    {
        workflow := api.Group("/workflow")
        {
            workflow.POST("/run", workflowHandler.RunCode)
        }
    }
}