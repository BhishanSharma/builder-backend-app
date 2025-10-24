// handlers/workflow.handler.go
package handlers

import (
    "bytes"
    "encoding/json"
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
    "builder.ai/src/utils"
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

// Use WorkflowConfig from utils package
// type WorkflowConfig = utils.WorkflowConfig (alias, not needed with direct import)

// RunCode handles the workflow execution and code concatenation
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

    projectDir, err := os.Getwd()
    if err != nil {
        fmt.Println("Error getting project directory:", err)
        projectDir = "."
    }

    // Save CSV if provided
    csvFile := filepath.Join(projectDir, fmt.Sprintf("temp/workflow_%d.csv", time.Now().Unix()))
    if request.Data.Schema != "" {
        err := os.MkdirAll(filepath.Dir(csvFile), 0755)
        if err != nil {
            fmt.Println("Error creating temp directory:", err)
        }
        err = os.WriteFile(csvFile, []byte(request.Data.Schema), 0644)
        if err != nil {
            fmt.Println("Error saving CSV:", err)
        } else {
            fmt.Println("CSV saved at:", csvFile)
        }
    }

    // Process code items
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

    c.JSON(http.StatusOK, gin.H{
        "message":           "Code concatenated successfully",
        "total_items":       len(request.Items),
        "concatenated_code": concatenatedCode,
        "components":        componentDetails,
        "csv_file":          csvFile,
    })
}

// GenerateExecutableScript generates a complete runnable Python script
func (h *WorkflowHandler) GenerateExecutableScript(c *gin.Context) {
    var request struct {
        WorkflowConfig string `json:"workflow_config" binding:"required"`
        ComponentCode  string `json:"component_code" binding:"required"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        fmt.Printf("Error binding JSON: %v\n", err)
        c.JSON(http.StatusBadRequest, gin.H{
            "error": fmt.Sprintf("Invalid request: %v", err),
            "details": "workflow_config and component_code are required fields",
        })
        return
    }

    fmt.Printf("Received workflow_config length: %d\n", len(request.WorkflowConfig))
    fmt.Printf("Received component_code length: %d\n", len(request.ComponentCode))

    // Parse workflow config
    var workflow utils.WorkflowConfig
    err := json.Unmarshal([]byte(request.WorkflowConfig), &workflow)
    if err != nil {
        fmt.Printf("Error parsing workflow config: %v\n", err)
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid workflow config JSON",
            "details": err.Error(),
        })
        return
    }

    fmt.Printf("Parsed workflow: version=%s, nodes=%d\n", workflow.Version, len(workflow.Nodes))

    // Generate executable script
    script, err := utils.GenerateExecutableScript(workflow, request.ComponentCode)
    if err != nil {
        fmt.Printf("Error generating script: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to generate script",
            "details": err.Error(),
        })
        return
    }

    fmt.Printf("Successfully generated script, length: %d\n", len(script))

    c.JSON(http.StatusOK, gin.H{
        "script":  script,
        "message": "Executable script generated successfully",
    })
}

// GenerateAndDownloadScript generates script from workflow items
func (h *WorkflowHandler) GenerateAndDownloadScript(c *gin.Context) {
    var request struct {
        Items []CodeItem `json:"items" binding:"required,min=1"`
        Data  struct {
            Schema string `json:"schema"`
        } `json:"data"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Build workflow config from items
    workflowConfig := utils.WorkflowConfig{
        Version:    "1.0",
        ExportedAt: time.Now().Format(time.RFC3339),
        Nodes:      []utils.Node{},
    }

    var codeBlocks []string
    stageNum := 1
    
    for i, item := range request.Items {
        if item.Code == "" {
            continue
        }

        // Process variables
        processedCode := item.Code
        variablesMap := make(map[string]interface{})
        
        for _, variable := range item.Variables {
            placeholder := fmt.Sprintf("{{%s}}", variable.Name)
            processedCode = strings.ReplaceAll(processedCode, placeholder, variable.Value)
            variablesMap[variable.Name] = variable.Value
        }

        codeBlocks = append(codeBlocks, processedCode)

        // Create node
        node := utils.Node{
            ID:        fmt.Sprintf("node_%d", i),
            Name:      extractFunctionName(item.Code),
            Stage:     stageNum,
            Code:      extractFunctionName(item.Code),
            Variables: variablesMap,
        }

        workflowConfig.Nodes = append(workflowConfig.Nodes, node)
        
        // Increment stage every 2 components (or use your logic)
        if (i+1)%2 == 0 && stageNum < 4 {
            stageNum++
        }
    }

    concatenatedCode := strings.Join(codeBlocks, "\n\n")

    // Generate executable script
    script, err := utils.GenerateExecutableScript(workflowConfig, concatenatedCode)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "script":            script,
        "concatenated_code": concatenatedCode,
        "message":           "Executable script generated successfully",
        "total_components":  len(request.Items),
    })
}

// extractFunctionName extracts function name from Python code
func extractFunctionName(code string) string {
    lines := strings.Split(code, "\n")
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "def ") {
            // Extract function name
            parts := strings.Split(trimmed, "(")
            if len(parts) > 0 {
                funcName := strings.TrimPrefix(parts[0], "def ")
                return strings.TrimSpace(funcName)
            }
        }
    }
    return "unknown_function"
}

// executeInDocker runs the Python code in a Docker container
func executeInDocker(code string) (string, error) {
    tempDir := "/tmp/code_execution"
    err := os.MkdirAll(tempDir, 0755)
    if err != nil {
        return "", fmt.Errorf("failed to create temp directory: %v", err)
    }

    timestamp := time.Now().Unix()
    filename := fmt.Sprintf("script_%d.py", timestamp)
    filepath := filepath.Join(tempDir, filename)

    err = os.WriteFile(filepath, []byte(code), 0644)
    if err != nil {
        return "", fmt.Errorf("failed to write code to file: %v", err)
    }

    defer os.Remove(filepath)

    dockerImage := os.Getenv("PYTHON_DOCKER_IMAGE")
    if dockerImage == "" {
        dockerImage = "python:3.11-slim"
    }

    cmd := exec.Command(
        "docker", "run",
        "--rm",
        "-v", fmt.Sprintf("%s:/code", tempDir),
        "--network", "none",
        "--memory", "2g",
        "--cpus", "2",
        dockerImage,
        "python", fmt.Sprintf("/code/%s", filename),
    )

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return stderr.String(), fmt.Errorf("execution error: %v", err)
    }

    output := stdout.String()
    if stderr.Len() > 0 {
        output += "\n[STDERR]\n" + stderr.String()
    }

    return output, nil
}