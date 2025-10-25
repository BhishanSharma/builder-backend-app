// src/utils/script_generator.go
package utils

import (
	"fmt"
	"strconv"
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
	fmt.Println(workflow.Nodes)

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
import warnings
warnings.filterwarnings('ignore', category=FutureWarning)

# ============================================================
# COMPONENT FUNCTIONS
# ============================================================

%s

# ============================================================
# PIPELINE EXECUTION
# ============================================================

def execute_pipeline(data_file, target_column='target', output_file='output.csv', skip_split_warning=False):
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
    
    # Initialize pipeline variables
    current_data = X
    model = None
    le = None
    X_train, X_test, y_train, y_test = None, None, None, None
    split_performed = False
    
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

	// Add validation check
	sb.WriteString(`    # ============================================================
    # VALIDATION CHECK
    # ============================================================
    if model is not None and not split_performed and not skip_split_warning:
        print(f"\n⚠ WARNING: Model was trained but no train/test split was performed!")
        print(f"  Metrics shown are from training data and may be overly optimistic.")
        print(f"  Consider adding a train/test split component to Stage 1.")
    
`)

	// Add output saving
	sb.WriteString(`    # ============================================================
    # SAVE OUTPUT
    # ============================================================
    print(f"\n[SAVING OUTPUT]")
    
    # Save processed features
    if isinstance(current_data, pd.DataFrame):
        current_data.to_csv(output_file, index=False)
        print(f"✓ Processed features saved to: {output_file}")
    else:
        print(f"⚠ Could not save output (unsupported data type)")
    
    # Save test set if available
    if X_test is not None:
        test_file = output_file.replace('.csv', '_test.csv')
        if isinstance(X_test, pd.DataFrame):
            X_test.to_csv(test_file, index=False)
            print(f"✓ Test features saved to: {test_file}")
    
    print(f"\n{'='*60}")
    print("PIPELINE COMPLETED")
    print(f"{'='*60}")
    
    return {
        'data': current_data,
        'model': model,
        'label_encoder': le,
        'X_train': X_train,
        'X_test': X_test,
        'y_train': y_train,
        'y_test': y_test,
        'split_performed': split_performed
    }

# ============================================================
# MAIN ENTRY POINT
# ============================================================

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Execute ML pipeline')
    parser.add_argument('--data', required=True, help='Input CSV file')
    parser.add_argument('--target', default='target', help='Target column name (default: target)')
    parser.add_argument('--output', default='output.csv', help='Output file (default: output.csv)')
    parser.add_argument('--skip-split-warning', action='store_true', help='Skip train/test split warning')
    
    args = parser.parse_args()
    
    try:
        result = execute_pipeline(args.data, args.target, args.output, args.skip_split_warning)
        print(f"\n✓ Pipeline executed successfully!")
        
        if result['model'] is not None:
            print(f"✓ Model trained and ready to use")
        
        if result['split_performed']:
            print(f"✓ Train/test split performed")
            if result['X_test'] is not None:
                print(f"  - Training samples: {len(result['X_train'])}")
                print(f"  - Test samples: {len(result['X_test'])}")
        
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

// Helper function to detect split components
func isSplitComponent(funcName string) bool {
	lowerFunc := strings.ToLower(funcName)
	return strings.Contains(lowerFunc, "split") || strings.Contains(lowerFunc, "stratified")
}

// Helper function to detect cross-validation components
func isCrossValidationComponent(funcName string) bool {
	lowerFunc := strings.ToLower(funcName)
	cvKeywords := []string{"cross", "kfold", "k_fold", "_cv", "crossval"}

	for _, keyword := range cvKeywords {
		if strings.Contains(lowerFunc, keyword) {
			return true
		}
	}
	return false
}

// Helper function to check if component removes rows
func isRowFilteringComponent(funcName string) bool {
	lowerFunc := strings.ToLower(funcName)
	filterKeywords := []string{"outlier", "remove", "drop", "filter"}

	for _, keyword := range filterKeywords {
		if strings.Contains(lowerFunc, keyword) {
			return true
		}
	}
	return false
}

// Helper function to filter out specific parameters from variable string
func filterOutParameter(varStr string, paramToRemove string) string {
	if varStr == "" {
		return ""
	}

	// Split by comma
	parts := strings.Split(varStr, ", ")
	var filtered []string

	for _, part := range parts {
		// Check if this part starts with the parameter name
		if !strings.HasPrefix(strings.TrimSpace(part), paramToRemove+"=") {
			filtered = append(filtered, part)
		}
	}

	return strings.Join(filtered, ", ")
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

		// Check if component needs target_column (like train_test_split)
		needsTarget := isSplitComponent(funcName)
		canFilterRows := isRowFilteringComponent(funcName)

		if needsTarget {
			// Filter out target_column from varStr since it's passed as positional arg
			filteredVarStr := filterOutParameter(varStr, "target_column")

			// Components that split data return (X_train, X_test, y_train, y_test)
			if filteredVarStr != "" {
				sb.WriteString(fmt.Sprintf(`        if y is not None:
            X_train, X_test, y_train, y_test = %s(df, target_column, %s)
            current_data = X_train
            split_performed = True
            print(f"    ✓ Split into train ({len(X_train)}) and test ({len(X_test)}) sets")
        else:
            print(f"    ⚠ No target column, skipping train/test split")
`, funcName, filteredVarStr))
			} else {
				sb.WriteString(fmt.Sprintf(`        if y is not None:
            X_train, X_test, y_train, y_test = %s(df, target_column)
            current_data = X_train
            split_performed = True
            print(f"    ✓ Split into train ({len(X_train)}) and test ({len(X_test)}) sets")
        else:
            print(f"    ⚠ No target column, skipping train/test split")
`, funcName))
			}
		} else {
			// Regular preprocessing components
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
`)

			// Add index synchronization for row-filtering components
			if canFilterRows {
				sb.WriteString(`        
        # Synchronize target variable if rows were removed
        if y is not None and not split_performed:
            if isinstance(current_data, pd.DataFrame) and len(current_data) != len(y):
                y = y.loc[current_data.index]
                print(f"    ⚠ Synced target variable: {len(y)} samples remaining")
`)
			}
		}

		sb.WriteString(`        print(f"    ✓ Completed")
    except Exception as e:
        print(f"    ⚠ Error: {e}")
        print(f"    Skipping component...")
    
`)
	}

	// Stage 3: Model Training
	if stage == 3 {
		sb.WriteString(fmt.Sprintf(`    print(f"  [%d/%d] Training: %s")
    if y is not None or y_train is not None:
        try:
            # Prepare data for training
            if split_performed and X_train is not None:
                # Use split data
                if isinstance(X_train, pd.DataFrame):
                    X_for_training = X_train.values
                else:
                    X_for_training = X_train
                y_for_training = y_train
                print(f"    ℹ Using training split: {len(X_for_training)} samples")
            else:
                # Use all data
                if isinstance(current_data, pd.DataFrame):
                    X_for_training = current_data.values
                else:
                    X_for_training = current_data
                y_for_training = y
                print(f"    ℹ Using all data: {len(X_for_training)} samples")
            
            # Encode labels if needed
            if hasattr(y_for_training, 'dtype') and y_for_training.dtype == 'object':
                from sklearn.preprocessing import LabelEncoder
                le = LabelEncoder()
                y_encoded = le.fit_transform(y_for_training)
                print(f"    ✓ Encoded {len(le.classes_)} classes: {list(le.classes_)}")
            else:
                y_encoded = y_for_training.values if hasattr(y_for_training, 'values') else y_for_training
            
            # Train model
`, index, total, compName))

		if varStr != "" {
			sb.WriteString(fmt.Sprintf(`            model = %s(X_for_training, y_encoded, %s)
`, funcName, varStr))
		} else {
			sb.WriteString(fmt.Sprintf(`            model = %s(X_for_training, y_encoded)
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
		if isCrossValidationComponent(funcName) {
			// Cross-validation: uses model and data
			sb.WriteString(fmt.Sprintf(`    print(f"  [%d/%d] Evaluating: %s")
    if model is not None and (y is not None or y_train is not None):
        try:
            # Prepare data for CV
            if split_performed and X_train is not None:
                X_for_cv = X_train.values if isinstance(X_train, pd.DataFrame) else X_train
                y_for_cv = y_train
            else:
                X_for_cv = current_data.values if isinstance(current_data, pd.DataFrame) else current_data
                y_for_cv = y
            
            # Encode if needed
            if hasattr(y_for_cv, 'dtype') and y_for_cv.dtype == 'object':
                if le is None:
                    from sklearn.preprocessing import LabelEncoder
                    le = LabelEncoder()
                    y_encoded = le.fit_transform(y_for_cv)
                else:
                    y_encoded = le.transform(y_for_cv)
            else:
                y_encoded = y_for_cv.values if hasattr(y_for_cv, 'values') else y_for_cv
            
            # Perform cross-validation
`, index, total, compName))

			if varStr != "" {
				sb.WriteString(fmt.Sprintf(`            cv_results = %s(model, X_for_cv, y_encoded, %s)
`, funcName, varStr))
			} else {
				sb.WriteString(fmt.Sprintf(`            cv_results = %s(model, X_for_cv, y_encoded)
`, funcName))
			}

			sb.WriteString(`            
            # Print CV results
            if isinstance(cv_results, dict):
                print(f"\n    Cross-Validation Results:")
                if 'mean_test_score' in cv_results:
                    print(f"      Mean Test Score: {cv_results['mean_test_score']:.4f} (+/- {cv_results.get('std_test_score', 0):.4f})")
                if 'mean_train_score' in cv_results:
                    print(f"      Mean Train Score: {cv_results['mean_train_score']:.4f} (+/- {cv_results.get('std_train_score', 0):.4f})")
                if 'test_scores' in cv_results:
                    print(f"      Individual Fold Scores: {[f'{score:.4f}' for score in cv_results['test_scores']]}")
            
            print(f"    ✓ Cross-validation completed")
        except Exception as e:
            print(f"    ⚠ Cross-validation failed: {e}")
            import traceback
            traceback.print_exc()
    else:
        print(f"    ⚠ No model or target, skipping cross-validation")
    
`)
		} else {
			// Regular metrics evaluation
			sb.WriteString(fmt.Sprintf(`    print(f"  [%d/%d] Evaluating: %s")
    if model is not None and (y is not None or y_train is not None):
        try:
            # Determine which data to use for evaluation
            if split_performed and X_test is not None and y_test is not None:
                # Use test set
                X_eval = X_test.values if isinstance(X_test, pd.DataFrame) else X_test
                y_for_eval = y_test
                eval_type = "test"
                print(f"    ℹ Evaluating on test set: {len(X_eval)} samples")
            elif split_performed and X_train is not None:
                # Use training set (no test available)
                X_eval = X_train.values if isinstance(X_train, pd.DataFrame) else X_train
                y_for_eval = y_train
                eval_type = "training"
                print(f"    ⚠ Evaluating on training set: {len(X_eval)} samples")
            else:
                # Use all data
                X_eval = current_data.values if isinstance(current_data, pd.DataFrame) else current_data
                y_for_eval = y
                eval_type = "all data"
                print(f"    ⚠ Evaluating on all data: {len(X_eval)} samples")
            
            # Encode labels if needed
            if hasattr(y_for_eval, 'dtype') and y_for_eval.dtype == 'object':
                if le is None:
                    from sklearn.preprocessing import LabelEncoder
                    le = LabelEncoder()
                    y_encoded = le.fit_transform(y_for_eval)
                else:
                    y_encoded = le.transform(y_for_eval)
            else:
                y_encoded = y_for_eval.values if hasattr(y_for_eval, 'values') else y_for_eval
            
            # Make predictions
            y_pred = model.predict(X_eval)
            
            # Get probabilities if available
            try:
                y_pred_proba = model.predict_proba(X_eval)
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
                print(f"\n    Metrics ({eval_type} set):")
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
	}

	return sb.String()
}

func buildVariablesString(variables map[string]interface{}) string {
	if len(variables) == 0 {
		return ""
	}

	var parts []string
	for key, value := range variables {
		// Skip empty string values
		if strVal, ok := value.(string); ok && strVal == "" {
			continue
		}

		var formatted string
		switch v := value.(type) {
		case string:
			// Handle special Python literals
			if v == "None" || v == "True" || v == "False" {
				// Python keywords - no quotes
				formatted = fmt.Sprintf("%s=%s", key, v)
			} else if isNumeric(v) {
				// Numeric strings - no quotes
				formatted = fmt.Sprintf("%s=%s", key, v)
			} else if strings.HasPrefix(v, "(") && strings.HasSuffix(v, ")") {
				// Tuples - no quotes
				formatted = fmt.Sprintf("%s=%s", key, v)
			} else if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
				// Lists - no quotes
				formatted = fmt.Sprintf("%s=%s", key, v)
			} else if strings.HasPrefix(v, "{") && strings.HasSuffix(v, "}") {
				// Dicts - no quotes
				formatted = fmt.Sprintf("%s=%s", key, v)
			} else {
				// Regular strings - add quotes
				formatted = fmt.Sprintf("%s='%s'", key, v)
			}
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

// isNumeric checks if a string represents a number
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	// Check if it's a valid number (int or float)
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
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