// handlers/workflow.handler.go
package handlers

import (
    "bytes"
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/mongo"

    "builder.ai/config"
)

type WorkflowHandler struct {
    collection *mongo.Collection
}

func NewWorkflowHandler() *WorkflowHandler {
    return &WorkflowHandler{
        collection: config.GetCollection("components"),
    }
}

// Variable represents a single variable with name and value
type Variable struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

// CodeItem represents a code block with its variables
type CodeItem struct {
    Code      string     `json:"code" binding:"required"`
    Variables []Variable `json:"variables"`
}

// ConcatenateRequest is the request payload for running workflow code
type ConcatenateRequest struct {
    Items []CodeItem `json:"items" binding:"required,min=1"`
}

func (h *WorkflowHandler) RunCode(c *gin.Context) {
    var request struct {
        Items []CodeItem `json:"items" binding:"required,min=1"`
        Data  struct {
            Schema string `json:"schema"` // raw CSV string
        } `json:"data"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    projectDir, err := os.Getwd() // get current project directory
    if err != nil {
        fmt.Println("Error getting project directory:", err)
        projectDir = "." // fallback to current directory
    }

    csvFile := filepath.Join(projectDir, fmt.Sprintf("temp/workflow_%d.csv", time.Now().Unix()))
    if request.Data.Schema != "" {
        err := os.WriteFile(csvFile, []byte(request.Data.Schema), 0644)
        if err != nil {
            fmt.Println("Error saving CSV:", err)
        } else {
            fmt.Println("CSV saved at:", csvFile)
        }
    }

    // Process code items as before...
    var codeBlocks []string
    var componentDetails []map[string]interface{}
    for i, item := range request.Items {
        if item.Code == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Empty code at index %d", i)})
            return
        }

        processedCode := item.Code
        for _, variable := range item.Variables {
            placeholder := fmt.Sprintf("{{%s}}", variable.Name)
            processedCode = strings.ReplaceAll(processedCode, placeholder, variable.Value)
        }

        codeBlocks = append(codeBlocks, processedCode)
        componentDetails = append(componentDetails, map[string]interface{}{
            "index":     i,
            "code":      item.Code,
            "variables": item.Variables,
            "processed": processedCode,
        })
    }

    concatenatedCode := strings.Join(codeBlocks, "\n\n")
    output, executionError := executeInDocker(concatenatedCode)

    response := gin.H{
        "message":           "Code executed successfully",
        "total_items":       len(request.Items),
        "concatenated_code": concatenatedCode,
        "components":        componentDetails,
        "csv_file":          csvFile,
        "execution": gin.H{
            "output": output,
        },
    }

    if executionError != nil {
        response["execution"].(gin.H)["error"] = executionError.Error()
        response["message"] = "Code execution failed"
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
