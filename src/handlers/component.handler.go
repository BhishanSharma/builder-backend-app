// handlers/component.handler.go
package handlers

import (
    "context"
    "net/http"
    "time"
    "fmt"
    "strconv"
    "math"
    "io"
    "sync"
    "encoding/json"
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

// Create creates one or more new components
func (h *ComponentHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try binding either a single object or an array
	var components []models.Component

	// Peek first byte to check if it's an array or object
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Determine whether input is an array or single object
	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty request body"})
		return
	}

	if body[0] == '{' {
		// Single component
		var single models.Component
		if err := json.Unmarshal(body, &single); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		components = append(components, single)
	} else if body[0] == '[' {
		// Array of components
		if err := json.Unmarshal(body, &components); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Validation and insertion
	var inserted []models.Component
	for _, component := range components {
		if !component.IsValidStage() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stage. Must be stage1, stage2, stage3, or stage4"})
			return
		}
		if !component.ValidateInputTypes() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input type. Must be string, int, float, bool, list, dict, or any"})
			return
		}
		if !component.ValidateOutputType() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid output type. Must be string, int, float, bool, list, dict, any, or none"})
			return
		}
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
		inserted = append(inserted, component)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    fmt.Sprintf("%d component(s) created successfully", len(inserted)),
		"components": inserted,
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

// SearchByName searches components by name with pagination and optimizations
func (h *ComponentHandler) SearchByName(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Get query parameters
    name := c.Query("name")
    if name == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Name query parameter is required"})
        return
    }

    // Get optional stage filter
    stage := c.Query("stage")

    // Pagination parameters with defaults
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
    
    // Validate pagination
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 50
    }
    
    skip := (page - 1) * limit

    // Sort parameter
    sortBy := c.DefaultQuery("sort", "name")
    sortOrder := 1
    if c.Query("order") == "desc" {
        sortOrder = -1
    }

    // Build filter - use prefix match for better index usage
    filter := bson.M{"name": bson.M{"$regex": "^" + name, "$options": "i"}}
    
    // Add stage filter if provided
    if stage != "" {
        filter["stage"] = stage
    }

    // Get total count and results in parallel using goroutines
    var totalCount int64
    var components []models.Component
    var countErr, findErr error

    // Use WaitGroup for parallel execution
    var wg sync.WaitGroup
    wg.Add(2)

    // Get count in goroutine
    go func() {
        defer wg.Done()
        totalCount, countErr = h.collection.CountDocuments(ctx, filter)
    }()

    // Get results in goroutine
    go func() {
        defer wg.Done()
        findOptions := options.Find().
            SetSkip(int64(skip)).
            SetLimit(int64(limit)).
            SetSort(bson.D{{Key: sortBy, Value: sortOrder}}).
            SetProjection(bson.M{
                "name":        1,
                "description": 1,
                "stage":       1,
                "inputs":      1,
                "output":      1,
                "code":        1,
            })

        cursor, err := h.collection.Find(ctx, filter, findOptions)
        if err != nil {
            findErr = err
            return
        }
        defer cursor.Close(ctx)

        findErr = cursor.All(ctx, &components)
    }()

    // Wait for both operations to complete
    wg.Wait()

    // Check for errors
    if countErr != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count documents"})
        return
    }
    if findErr != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute search"})
        return
    }

    // Handle empty results
    if components == nil {
        components = []models.Component{}
    }

    // Calculate pagination metadata
    totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
    hasNext := page < totalPages
    hasPrev := page > 1
    
    // Return paginated response
    c.JSON(http.StatusOK, gin.H{
        "data": components,
        "pagination": gin.H{
            "page":       page,
            "limit":      limit,
            "total":      totalCount,
            "totalPages": totalPages,
            "hasNext":    hasNext,
            "hasPrev":    hasPrev,
        },
    })
}

// CreateSearchIndexes creates optimized indexes for search
func (h *ComponentHandler) CreateSearchIndexes(ctx context.Context) error {
    indexes := []mongo.IndexModel{
        {
            Keys: bson.D{
                {Key: "name", Value: 1},
            },
            Options: options.Index().
                SetName("name_1").
                SetBackground(true),
        },
        {
            Keys: bson.D{
                {Key: "stage", Value: 1},
                {Key: "name", Value: 1},
            },
            Options: options.Index().
                SetName("stage_1_name_1").
                SetBackground(true),
        },
    }
    
    _, err := h.collection.Indexes().CreateMany(ctx, indexes)
    return err
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