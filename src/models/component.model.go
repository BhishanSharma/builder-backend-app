package models

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Component struct {
    ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
    Name        string             `json:"name" bson:"name" binding:"required"`
    Description string             `json:"description" bson:"description"`
    Code        string             `json:"code" bson:"code" binding:"required"`
    Language    string             `json:"language" bson:"language" binding:"required"` // e.g., "go", "javascript", "python"
    Stage       string             `json:"stage" bson:"stage" binding:"required,oneof=stage1 stage2 stage3 stage4"`
    Tags        []string           `json:"tags" bson:"tags"`
    CreatedBy   primitive.ObjectID `json:"created_by,omitempty" bson:"created_by,omitempty"`
    CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
    UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// Stage constants
const (
    Stage1 = "stage1"
    Stage2 = "stage2"
    Stage3 = "stage3"
    Stage4 = "stage4"
)

// IsValidStage checks if the stage is valid
func (c *Component) IsValidStage() bool {
    validStages := []string{Stage1, Stage2, Stage3, Stage4}
    for _, stage := range validStages {
        if c.Stage == stage {
            return true
        }
    }
    return false
}

// GetStageNumber returns the stage number
func (c *Component) GetStageNumber() int {
    switch c.Stage {
    case Stage1:
        return 1
    case Stage2:
        return 2
    case Stage3:
        return 3
    case Stage4:
        return 4
    default:
        return 0
    }
}