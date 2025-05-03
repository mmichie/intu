package aikit

import (
	"encoding/json"
	"testing"
)

func TestFunctionDefinition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		fd      FunctionDefinition
		wantErr bool
	}{
		{
			name: "valid definition",
			fd: FunctionDefinition{
				Name:        "test_function",
				Description: "A test function",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "The name parameter",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			fd: FunctionDefinition{
				Description: "A test function",
				Parameters: map[string]interface{}{
					"type": "object",
				},
			},
			wantErr: true,
		},
		{
			name: "missing description",
			fd: FunctionDefinition{
				Name: "test_function",
				Parameters: map[string]interface{}{
					"type": "object",
				},
			},
			wantErr: true,
		},
		{
			name: "nil parameters",
			fd: FunctionDefinition{
				Name:        "test_function",
				Description: "A test function",
			},
			wantErr: true,
		},
		{
			name: "empty parameters are valid",
			fd: FunctionDefinition{
				Name:        "test_function",
				Description: "A test function",
				Parameters:  map[string]interface{}{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fd.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFunctionDefinition_ToMap(t *testing.T) {
	fd := FunctionDefinition{
		Name:        "test_function",
		Description: "A test function",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "The name parameter",
				},
			},
		},
	}

	result := fd.ToMap()

	// Check name
	if name, ok := result["name"].(string); !ok || name != "test_function" {
		t.Errorf("ToMap() name = %v, want %v", result["name"], "test_function")
	}

	// Check description
	if desc, ok := result["description"].(string); !ok || desc != "A test function" {
		t.Errorf("ToMap() description = %v, want %v", result["description"], "A test function")
	}

	// Check parameters (simple check that it exists and is the same reference)
	if params, ok := result["parameters"].(map[string]interface{}); !ok {
		t.Errorf("ToMap() parameters not of expected type")
	} else if params["type"] != "object" {
		t.Errorf("ToMap() parameters content not preserved correctly")
	}
}

func TestFunctionResponse_Serialization(t *testing.T) {
	// Test that FunctionResponse can be properly serialized to JSON
	resp := FunctionResponse{
		Name:     "test_function",
		Content:  map[string]interface{}{"result": "value"},
		Error:    "",
		Metadata: map[string]string{"time": "now"},
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal FunctionResponse: %v", err)
	}

	var decoded FunctionResponse
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal FunctionResponse: %v", err)
	}

	if decoded.Name != "test_function" {
		t.Errorf("Name not preserved in serialization, got %s", decoded.Name)
	}

	// Check Content field (simple type assertion check)
	if content, ok := decoded.Content.(map[string]interface{}); !ok {
		t.Errorf("Content not properly deserialized")
	} else if val, ok := content["result"].(string); !ok || val != "value" {
		t.Errorf("Content value not preserved in serialization")
	}

	if decoded.Error != "" {
		t.Errorf("Error field not preserved, got %s", decoded.Error)
	}

	// Check Metadata field
	if metadata, ok := decoded.Metadata.(map[string]interface{}); !ok {
		t.Errorf("Metadata not properly deserialized")
	} else if val, ok := metadata["time"].(string); !ok || val != "now" {
		t.Errorf("Metadata value not preserved in serialization")
	}
}
