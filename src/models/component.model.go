package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ComponentInput represents an input parameter for a component
type ComponentInput struct {
    Name        string `json:"name" bson:"name" binding:"required"`
    Type        string `json:"type" bson:"type" binding:"required"` // e.g., "string", "int", "float", "bool", "list", "dict", "any"
    Description string `json:"description" bson:"description"`
    Required    bool   `json:"required" bson:"required"`
    DefaultValue interface{} `json:"default_value,omitempty" bson:"default_value,omitempty"`
}

// ComponentOutput represents the output of a component
type ComponentOutput struct {
    Type        string `json:"type" bson:"type" binding:"required"` // e.g., "string", "int", "float", "bool", "list", "dict", "any", "none"
    Description string `json:"description" bson:"description"`
}

type Component struct {
    ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
    Name        string             `json:"name" bson:"name" binding:"required"`
    Description string             `json:"description" bson:"description"`
    Code        string             `json:"code" bson:"code" binding:"required"`
    Language    string             `json:"language" bson:"language" binding:"required"` // e.g., "python", "go", "javascript"
    Stage       string             `json:"stage" bson:"stage" binding:"required,oneof=stage1 stage2 stage3 stage4"`
    Tags        []string           `json:"tags" bson:"tags"`
    Inputs      []ComponentInput   `json:"inputs" bson:"inputs"`           // Array of inputs (1 to n)
    Output      *ComponentOutput   `json:"output,omitempty" bson:"output,omitempty"` // Optional output (0 or 1)
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

// Data type constants
const (
    TypeString  = "string"
    TypeInt     = "int"
    TypeFloat   = "float"
    TypeBool    = "bool"
    TypeList    = "list"
    TypeDict    = "dict"
    TypeAny     = "any"
    TypeDataFrame = "DataFrame"
    TypeSeries   = "Series"
    TypeTuple   = "tuple"
    TypeArray   = "array"
    TypeObject  = "object"
    TypeIterable = "iterable"
    TypeDateTime = "datetime"
    TypeNdArray   = "ndarray"
    TypeTensor    = "tensor"
    TypeFunction  = "function"
    TypeKerasModel = "keras.model"
    TypeCallable = "callable"
    TypeNone    = "none"
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

// HasOutput checks if component has an output
func (c *Component) HasOutput() bool {
    return c.Output != nil && c.Output.Type != TypeNone
}

// GetRequiredInputs returns all required inputs
func (c *Component) GetRequiredInputs() []ComponentInput {
    var required []ComponentInput
    for _, input := range c.Inputs {
        if input.Required {
            required = append(required, input)
        }
    }
    return required
}

// GetOptionalInputs returns all optional inputs
func (c *Component) GetOptionalInputs() []ComponentInput {
    var optional []ComponentInput
    for _, input := range c.Inputs {
        if !input.Required {
            optional = append(optional, input)
        }
    }
    return optional
}

// ValidateInputTypes checks if input types are valid
func (c *Component) ValidateInputTypes() bool {
    validTypes := []string{TypeString, TypeInt, TypeFloat, TypeTensor, TypeBool, TypeList, TypeDict, TypeDataFrame, TypeSeries, TypeTuple, TypeArray, TypeObject, TypeIterable, TypeDateTime, TypeNdArray, TypeFunction, TypeKerasModel, TypeCallable, TypeAny}
    for _, input := range c.Inputs {
        isValid := false
        for _, validType := range validTypes {
            if input.Type == validType {
                isValid = true
                break
            }
        }
        if !isValid {
            fmt.Println("❌ Invalid Input Type:", input.Type)
            return false
        }
    }
    return true
}

// ValidateOutputType checks if output type is valid
func (c *Component) ValidateOutputType() bool {
    if c.Output == nil {
        return true
    }
    validTypes := []string{TypeString, TypeInt, TypeFloat, TypeTensor, TypeBool, TypeList, TypeDict, TypeAny, TypeDataFrame, TypeSeries, TypeTuple, TypeArray, TypeObject, TypeIterable, TypeDateTime, TypeNdArray, TypeNone}
    for _, validType := range validTypes {
        if c.Output.Type == validType {
            return true
        }
    }
    return false
}