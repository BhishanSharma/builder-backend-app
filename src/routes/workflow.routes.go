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
            // Original endpoint - returns concatenated code
            workflow.POST("/run", workflowHandler.RunCode)
            
            // New endpoint - generates executable script from workflow config + code
            workflow.POST("/generate-script", workflowHandler.GenerateExecutableScript)
            
            // Convenience endpoint - generates script directly from items
            workflow.POST("/export", workflowHandler.GenerateAndDownloadScript)
        }
    }
}