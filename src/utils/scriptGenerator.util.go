// src/utils/script_generator.go
package utils

import (
    "fmt"
    "strings"
    "time"
)

// Node represents a workflow component
type Node struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Stage       int                    `json:"stage"`
    Description string                 `json:"description,omitempty"`
    Code        string                 `json:"code"`
    Inputs      []Input                `json:"inputs,omitempty"`
    Output      map[string]interface{} `json:"output,omitempty"`
    Variables   map[string]interface{} `json:"variables,omitempty"`
}

type Input struct {
    Name string `json:"name"`
    Type string `json:"type"`
}

type WorkflowConfig struct {
    Version    string `json:"version"`
    ExportedAt string `json:"exported_at"`
    Nodes      []Node `json:"nodes"`
}

// GenerateExecutableScript generates a complete runnable Python script
func GenerateExecutableScript(workflow WorkflowConfig, componentCode string) (string, error) {
    // Organize nodes by stage
    stages := organizeByStage(workflow.Nodes)

    var sb strings.Builder

    // Header
    sb.WriteString(fmt.Sprintf(`#!/usr/bin/env python3
"""
Auto-generated Pipeline Script
Generated at: %s
Version: %s
Total Components: %d
"""

import pandas as pd
import numpy as np
import sys
import argparse

# ============================================================
# COMPONENT FUNCTIONS
# ============================================================

%s

# ============================================================
# PIPELINE EXECUTION
# ============================================================

def execute_pipeline(data_file, target_column='target', output_file='output.csv'):
    """Execute the complete pipeline"""
    
    print("="*60)
    print("PIPELINE EXECUTION")
    print("="*60)
    
    # Load data
    print(f"\n[LOADING DATA]")
    df = pd.read_csv(data_file)
    print(f"✓ Loaded {len(df)} samples")
    print(f"✓ Columns: {list(df.columns)}")
    
    # Separate features and target
    if target_column in df.columns:
        X = df.drop(columns=[target_column])
        y = df[target_column]
        print(f"✓ Target column: {target_column}")
    else:
        X = df
        y = None
        print(f"⚠ No target column found, processing features only")
    
    current_data = X
    model = None
    le = None
    
`, time.Now().Format(time.RFC3339), workflow.Version, len(workflow.Nodes), componentCode))

    // Generate stage execution code
    for stageNum := 1; stageNum <= 4; stageNum++ {
        nodes := stages[stageNum]
        if len(nodes) == 0 {
            continue
        }

        sb.WriteString(fmt.Sprintf(`    # ============================================================
    # STAGE %d
    # ============================================================
    print(f"\n[STAGE %d]")
    
`, stageNum, stageNum))

        for i, node := range nodes {
            sb.WriteString(generateComponentExecution(node, i+1, len(nodes), stageNum))
        }
    }

    // Add output saving and metrics display
    sb.WriteString(`    # ============================================================
    # SAVE OUTPUT
    # ============================================================
    print(f"\n[SAVING OUTPUT]")
    
    if isinstance(current_data, pd.DataFrame):
        current_data.to_csv(output_file, index=False)
        print(f"✓ Output saved to: {output_file}")
    else:
        print(f"⚠ Could not save output (unsupported data type)")
    
    print(f"\n{'='*60}")
    print("PIPELINE COMPLETED")
    print(f"{'='*60}")
    
    return {
        'data': current_data,
        'model': model,
        'label_encoder': le
    }

# ============================================================
# MAIN ENTRY POINT
# ============================================================

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Execute ML pipeline')
    parser.add_argument('--data', required=True, help='Input CSV file')
    parser.add_argument('--target', default='target', help='Target column name (default: target)')
    parser.add_argument('--output', default='output.csv', help='Output file (default: output.csv)')
    
    args = parser.parse_args()
    
    try:
        result = execute_pipeline(args.data, args.target, args.output)
        print(f"\n✓ Pipeline executed successfully!")
        
        if result['model'] is not None:
            print(f"✓ Model trained and ready to use")
        
    except FileNotFoundError as e:
        print(f"\n❌ Error: File not found - {e}")
        print(f"Make sure the file '{args.data}' exists")
        sys.exit(1)
    except KeyError as e:
        print(f"\n❌ Error: Column not found - {e}")
        print(f"Make sure the target column '{args.target}' exists in your CSV")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
`)

    return sb.String(), nil
}

func organizeByStage(nodes []Node) map[int][]Node {
    stages := make(map[int][]Node)
    for i := 1; i <= 4; i++ {
        stages[i] = []Node{}
    }

    for _, node := range nodes {
        stage := node.Stage
        if stage < 1 || stage > 4 {
            stage = 1
        }
        stages[stage] = append(stages[stage], node)
    }

    return stages
}

func generateComponentExecution(node Node, index, total, stage int) string {
    var sb strings.Builder

    compName := node.Name
    if compName == "" {
        compName = node.Code
    }

    funcName := node.Code
    if funcName == "" {
        funcName = strings.ToLower(strings.ReplaceAll(compName, " ", "_"))
    }

    // Build variables string
    varStr := buildVariablesString(node.Variables)

    // Stage 1 & 2: Preprocessing and Feature Engineering
    if stage == 1 || stage == 2 {
        sb.WriteString(fmt.Sprintf(`    print(f"  [%d/%d] Executing: %s")
    try:
`, index, total, compName))

        if varStr != "" {
            sb.WriteString(fmt.Sprintf(`        result = %s(current_data, %s)
`, funcName, varStr))
        } else {
            sb.WriteString(fmt.Sprintf(`        result = %s(current_data)
`, funcName))
        }

        sb.WriteString(`        if isinstance(result, tuple):
            current_data = result[0]
        else:
            current_data = result
        print(f"    ✓ Completed")
    except Exception as e:
        print(f"    ⚠ Error: {e}")
        print(f"    Skipping component...")
    
`)
    }

    // Stage 3: Model Training
    if stage == 3 {
        sb.WriteString(fmt.Sprintf(`    print(f"  [%d/%d] Training: %s")
    if y is not None:
        try:
            # Prepare data for training
            if isinstance(current_data, pd.DataFrame):
                X_train = current_data.values
            else:
                X_train = current_data
            
            # Encode labels if needed
            if y.dtype == 'object':
                from sklearn.preprocessing import LabelEncoder
                le = LabelEncoder()
                y_encoded = le.fit_transform(y)
                print(f"    ✓ Encoded {len(le.classes_)} classes: {list(le.classes_)}")
            else:
                y_encoded = y.values
            
            # Train model
`, index, total, compName))

        if varStr != "" {
            sb.WriteString(fmt.Sprintf(`            model = %s(X_train, y_encoded, %s)
`, funcName, varStr))
        } else {
            sb.WriteString(fmt.Sprintf(`            model = %s(X_train, y_encoded)
`, funcName))
        }

        sb.WriteString(`            print(f"    ✓ Model trained successfully")
        except Exception as e:
            print(f"    ⚠ Training failed: {e}")
            import traceback
            traceback.print_exc()
            model = None
    else:
        print(f"    ⚠ No target column, skipping training")
        model = None
    
`)
    }

    // Stage 4: Evaluation
    if stage == 4 {
        sb.WriteString(fmt.Sprintf(`    print(f"  [%d/%d] Evaluating: %s")
    if model is not None and y is not None:
        try:
            # Make predictions
            y_pred = model.predict(X_train)
            
            # Get probabilities if available
            try:
                y_pred_proba = model.predict_proba(X_train)
            except:
                y_pred_proba = None
            
            # Calculate metrics
`, index, total, compName))

        if varStr != "" {
            sb.WriteString(fmt.Sprintf(`            metrics = %s(y_encoded, y_pred, y_pred_proba, %s)
`, funcName, varStr))
        } else {
            sb.WriteString(fmt.Sprintf(`            metrics = %s(y_encoded, y_pred, y_pred_proba)
`, funcName))
        }

        sb.WriteString(`            
            # Print metrics
            if isinstance(metrics, dict):
                print(f"\n    Metrics:")
                for key, value in metrics.items():
                    if isinstance(value, (int, float)):
                        print(f"      {key}: {value:.4f}")
                    elif key == 'confusion_matrix':
                        print(f"      {key}:")
                        for row in value:
                            print(f"        {row}")
            
            print(f"    ✓ Evaluation completed")
        except Exception as e:
            print(f"    ⚠ Evaluation failed: {e}")
            import traceback
            traceback.print_exc()
    else:
        print(f"    ⚠ No model or target, skipping evaluation")
    
`)
    }

    return sb.String()
}

func buildVariablesString(variables map[string]interface{}) string {
    if len(variables) == 0 {
        return ""
    }

    var parts []string
    for key, value := range variables {
        var formatted string
        switch v := value.(type) {
        case string:
            formatted = fmt.Sprintf("%s='%s'", key, v)
        case float64:
            if v == float64(int(v)) {
                formatted = fmt.Sprintf("%s=%d", key, int(v))
            } else {
                formatted = fmt.Sprintf("%s=%f", key, v)
            }
        case bool:
            if v {
                formatted = fmt.Sprintf("%s=True", key)
            } else {
                formatted = fmt.Sprintf("%s=False", key)
            }
        case []interface{}:
            formatted = fmt.Sprintf("%s=%s", key, formatList(v))
        default:
            formatted = fmt.Sprintf("%s=%v", key, v)
        }
        parts = append(parts, formatted)
    }

    return strings.Join(parts, ", ")
}

func formatList(list []interface{}) string {
    var parts []string
    for _, item := range list {
        switch v := item.(type) {
        case string:
            parts = append(parts, fmt.Sprintf("'%s'", v))
        case float64:
            if v == float64(int(v)) {
                parts = append(parts, fmt.Sprintf("%d", int(v)))
            } else {
                parts = append(parts, fmt.Sprintf("%f", v))
            }
        default:
            parts = append(parts, fmt.Sprintf("%v", v))
        }
    }
    return "[" + strings.Join(parts, ", ") + "]"
}