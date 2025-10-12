// handlers/workflow.handler.go
package handlers

import (
    "bytes"
    "context"
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"

    "builder.ai/config"
    "builder.ai/src/models"
)

type WorkflowHandler struct {
    collection *mongo.Collection
}

func NewWorkflowHandler() *WorkflowHandler {
    return &WorkflowHandler{
        collection: config.GetCollection("components"),
    }
}

type WorkflowItem struct {
    Type  string `json:"type" binding:"required,oneof=id code"` 
    Value string `json:"value" binding:"required"`              
}

type ConcatenateRequest struct {
    Items []WorkflowItem `json:"items" binding:"required,min=1"`
}

func (h *WorkflowHandler) RunCode(c *gin.Context) {
    var request ConcatenateRequest
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // No timeout for database queries - let them run as long as needed
    ctx := context.Background()

    var codeBlocks []string
    var componentDetails []map[string]interface{}

    // Process each item in order
    for i, item := range request.Items {
        if item.Type == "id" {
            // It's a component ID - fetch from database
            objectID, err := primitive.ObjectIDFromHex(item.Value)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": fmt.Sprintf("Invalid ID format at index %d: %s", i, item.Value),
                })
                return
            }

            var component models.Component
            err = h.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&component)
            if err != nil {
                if err == mongo.ErrNoDocuments {
                    c.JSON(http.StatusNotFound, gin.H{
                        "error": fmt.Sprintf("Component not found at index %d: %s", i, item.Value),
                    })
                    return
                }
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            // Add component code to blocks
            codeBlocks = append(codeBlocks, component.Code)
            
            // Store component details
            componentDetails = append(componentDetails, map[string]interface{}{
                "index":       i,
                "type":        "component",
                "id":          component.ID.Hex(),
                "name":        component.Name,
                "description": component.Description,
                "stage":       component.Stage,
                "language":    component.Language,
                "inputs":      component.Inputs,
                "output":      component.Output,
            })

        } else {
            // It's raw code
            codeBlocks = append(codeBlocks, item.Value)
            
            // Store raw code details
            componentDetails = append(componentDetails, map[string]interface{}{
                "index": i,
                "type":  "raw_code",
                "code":  item.Value,
            })
        }
    }

    // Concatenate all code blocks
    concatenatedCode := strings.Join(codeBlocks, "\n\n")

    // Print to console
    fmt.Println("========== CONCATENATED CODE ==========")
    fmt.Println(concatenatedCode)
    fmt.Println("========================================")

    // Execute code in Docker container
    output, executionError := executeInDocker(concatenatedCode)

    // Return response
    response := gin.H{
        "message":           "Code executed successfully",
        "total_items":       len(request.Items),
        "concatenated_code": concatenatedCode,
        "components":        componentDetails,
        "execution": gin.H{
            "output": output,
        },
    }

    if executionError != nil {
        response["execution"].(gin.H)["error"] = executionError.Error()
        response["message"] = "Code execution failed"
        c.JSON(http.StatusOK, response) // Still return 200 with error details
        return
    }

    c.JSON(http.StatusOK, response)
}

// executeInDocker runs the Python code in a Docker container without time constraints
func executeInDocker(code string) (string, error) {
    // Create temporary directory for code files
    tempDir := "/tmp/code_execution"
    err := os.MkdirAll(tempDir, 0755)
    if err != nil {
        return "", fmt.Errorf("failed to create temp directory: %v", err)
    }

    // Create unique filename with timestamp
    timestamp := time.Now().Unix()
    filename := fmt.Sprintf("script_%d.py", timestamp)
    filepath := filepath.Join(tempDir, filename)

    // Write code to file
    err = os.WriteFile(filepath, []byte(code), 0644)
    if err != nil {
        return "", fmt.Errorf("failed to write code to file: %v", err)
    }

    // Clean up file after execution
    defer os.Remove(filepath)

    // Get Docker image name from environment variable or use default
    dockerImage := os.Getenv("PYTHON_DOCKER_IMAGE")
    if dockerImage == "" {
        dockerImage = "python:3.11-slim" // Default image
    }

    cmd := exec.Command(
        "docker", "run",
        "--rm",                                    // Remove container after execution
        "-v", fmt.Sprintf("%s:/code", tempDir),   // Mount volume
        "--network", "none",                       // Disable network for security
        "--memory", "2g",                          // Increased memory limit
        "--cpus", "2",                             // Increased CPU limit
        dockerImage,
        "python", fmt.Sprintf("/code/%s", filename),
    )

    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    // Start and wait for the command to complete (no timeout)
    if err := cmd.Run(); err != nil {
        return stderr.String(), fmt.Errorf("execution error: %v", err)
    }

    // Combine stdout and stderr
    output := stdout.String()
    if stderr.Len() > 0 {
        output += "\n[STDERR]\n" + stderr.String()
    }

    return output, nil
}
