// handlers/component.handler.go
package handlers

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "builder.ai/config"
    "builder.ai/src/models"
)

type ComponentHandler struct {
    collection *mongo.Collection
}

func NewComponentHandler() *ComponentHandler {
    return &ComponentHandler{
        collection: config.GetCollection("components"),
    }
}

// GetAll retrieves all components with optional filtering
func (h *ComponentHandler) GetAll(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var components []models.Component

    // Optional filters
    filter := bson.M{}
    
    if stage := c.Query("stage"); stage != "" {
        filter["stage"] = stage
    }
    
    if language := c.Query("language"); language != "" {
        filter["language"] = language
    }
    
    // Filter by output type
    if outputType := c.Query("output_type"); outputType != "" {
        filter["output.type"] = outputType
    }
    
    // Filter components with/without output
    if hasOutput := c.Query("has_output"); hasOutput == "true" {
        filter["output"] = bson.M{"$ne": nil}
    } else if hasOutput == "false" {
        filter["output"] = nil
    }

    opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

    cursor, err := h.collection.Find(ctx, filter, opts)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer cursor.Close(ctx)

    if err = cursor.All(ctx, &components); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "count":      len(components),
        "components": components,
    })
}

// GetByID retrieves a component by ID
func (h *ComponentHandler) GetByID(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    id := c.Param("id")
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
        return
    }

    var component models.Component
    err = h.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&component)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            c.JSON(http.StatusNotFound, gin.H{"error": "Component not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, component)
}

// GetByStage retrieves all components for a specific stage
func (h *ComponentHandler) GetByStage(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    stage := c.Param("stage")
    
    validStages := []string{"stage1", "stage2", "stage3", "stage4"}
    isValid := false
    for _, s := range validStages {
        if stage == s {
            isValid = true
            break
        }
    }
    if !isValid {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stage. Must be stage1, stage2, stage3, or stage4"})
        return
    }

    var components []models.Component

    cursor, err := h.collection.Find(ctx, bson.M{"stage": stage})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer cursor.Close(ctx)

    if err = cursor.All(ctx, &components); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "stage":      stage,
        "count":      len(components),
        "components": components,
    })
}

// Create creates a new component
func (h *ComponentHandler) Create(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var component models.Component
    if err := c.ShouldBindJSON(&component); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate stage
    if !component.IsValidStage() {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stage. Must be stage1, stage2, stage3, or stage4"})
        return
    }

    // Validate input types
    if !component.ValidateInputTypes() {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input type. Must be string, int, float, bool, list, dict, or any"})
        return
    }

    // Validate output type
    if !component.ValidateOutputType() {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid output type. Must be string, int, float, bool, list, dict, any, or none"})
        return
    }

    // Validate at least one input
    if len(component.Inputs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Component must have at least one input"})
        return
    }

    component.CreatedAt = time.Now()
    component.UpdatedAt = time.Now()

    result, err := h.collection.InsertOne(ctx, component)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    component.ID = result.InsertedID.(primitive.ObjectID)

    c.JSON(http.StatusCreated, gin.H{
        "message":   "Component created successfully",
        "component": component,
    })
}

// Update updates a component by ID
func (h *ComponentHandler) Update(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    id := c.Param("id")
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
        return
    }

    var component models.Component
    if err := c.ShouldBindJSON(&component); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if !component.IsValidStage() {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stage"})
        return
    }

    if !component.ValidateInputTypes() {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input type"})
        return
    }

    if !component.ValidateOutputType() {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid output type"})
        return
    }

    if len(component.Inputs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Component must have at least one input"})
        return
    }

    component.UpdatedAt = time.Now()

    update := bson.M{
        "$set": bson.M{
            "name":        component.Name,
            "description": component.Description,
            "code":        component.Code,
            "language":    component.Language,
            "stage":       component.Stage,
            "tags":        component.Tags,
            "inputs":      component.Inputs,
            "output":      component.Output,
            "updated_at":  component.UpdatedAt,
        },
    }

    result, err := h.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if result.MatchedCount == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Component not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Component updated successfully",
        "id":      id,
    })
}

// Delete deletes a component by ID
func (h *ComponentHandler) Delete(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    id := c.Param("id")
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
        return
    }

    result, err := h.collection.DeleteOne(ctx, bson.M{"_id": objectID})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if result.DeletedCount == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Component not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Component deleted successfully",
        "id":      id,
    })
}

// SearchByName searches components by name
func (h *ComponentHandler) SearchByName(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    name := c.Query("name")
    if name == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Name query parameter is required"})
        return
    }

    var components []models.Component

    filter := bson.M{"name": bson.M{"$regex": name, "$options": "i"}}
    cursor, err := h.collection.Find(ctx, filter)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer cursor.Close(ctx)

    if err = cursor.All(ctx, &components); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "count":      len(components),
        "components": components,
    })
}

// GetStageStats returns statistics for each stage
func (h *ComponentHandler) GetStageStats(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pipeline := []bson.M{
        {
            "$group": bson.M{
                "_id":   "$stage",
                "count": bson.M{"$sum": 1},
            },
        },
        {
            "$sort": bson.M{"_id": 1},
        },
    }

    cursor, err := h.collection.Aggregate(ctx, pipeline)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer cursor.Close(ctx)

    var results []bson.M
    if err = cursor.All(ctx, &results); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "stats": results,
    })
}

// GetByInputType finds components that accept a specific input type
func (h *ComponentHandler) GetByInputType(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    inputType := c.Query("type")
    if inputType == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Type query parameter is required"})
        return
    }

    var components []models.Component

    filter := bson.M{"inputs.type": inputType}
    cursor, err := h.collection.Find(ctx, filter)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer cursor.Close(ctx)

    if err = cursor.All(ctx, &components); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "input_type": inputType,
        "count":      len(components),
        "components": components,
    })
}

// GetByOutputType finds components with a specific output type
func (h *ComponentHandler) GetByOutputType(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    outputType := c.Query("type")
    if outputType == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Type query parameter is required"})
        return
    }

    var components []models.Component

    filter := bson.M{"output.type": outputType}
    cursor, err := h.collection.Find(ctx, filter)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer cursor.Close(ctx)

    if err = cursor.All(ctx, &components); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "output_type": outputType,
        "count":       len(components),
        "components":  components,
    })
}