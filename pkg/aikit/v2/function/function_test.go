package function

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestFunctionDefinitionValidation(t *testing.T) {
	tests := []struct {
		name    string
		def     FunctionDefinition
		wantErr bool
	}{
		{
			name: "valid definition",
			def: FunctionDefinition{
				Name:        "test_function",
				Description: "A test function",
				Parameters:  map[string]interface{}{"type": "object"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			def: FunctionDefinition{
				Description: "A test function",
				Parameters:  map[string]interface{}{"type": "object"},
			},
			wantErr: true,
		},
		{
			name: "missing description",
			def: FunctionDefinition{
				Name:       "test_function",
				Parameters: map[string]interface{}{"type": "object"},
			},
			wantErr: true,
		},
		{
			name: "missing parameters",
			def: FunctionDefinition{
				Name:        "test_function",
				Description: "A test function",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.def.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegistryThreadSafety(t *testing.T) {
	registry := NewRegistry()
	var wg sync.WaitGroup

	// Test concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			def := FunctionDefinition{
				Name:        "function_" + string(rune(id)),
				Description: "Test function",
				Parameters:  map[string]interface{}{"type": "object"},
			}
			registry.Register(def)
		}(i)
	}

	// Test concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registry.List()
			registry.Count()
			registry.Has("function_0")
		}()
	}

	wg.Wait()

	// Verify some functions were registered
	if registry.Count() == 0 {
		t.Error("No functions were registered")
	}
}

func TestRegistryOperations(t *testing.T) {
	registry := NewRegistry()

	// Test Register
	def1 := FunctionDefinition{
		Name:        "func1",
		Description: "First function",
		Parameters:  map[string]interface{}{"type": "object"},
	}

	err := registry.Register(def1)
	if err != nil {
		t.Errorf("Failed to register function: %v", err)
	}

	// Test duplicate registration
	err = registry.Register(def1)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}

	// Test Has
	if !registry.Has("func1") {
		t.Error("Has() returned false for registered function")
	}

	// Test Get
	retrieved, found := registry.Get("func1")
	if !found {
		t.Error("Get() failed to find registered function")
	}
	if retrieved.Name != def1.Name {
		t.Error("Retrieved function does not match original")
	}

	// Test Count
	if registry.Count() != 1 {
		t.Errorf("Count() returned %d, expected 1", registry.Count())
	}

	// Test RegisterMany
	defs := []FunctionDefinition{
		{
			Name:        "func2",
			Description: "Second function",
			Parameters:  map[string]interface{}{"type": "object"},
		},
		{
			Name:        "func3",
			Description: "Third function",
			Parameters:  map[string]interface{}{"type": "object"},
		},
	}

	err = registry.RegisterMany(defs)
	if err != nil {
		t.Errorf("RegisterMany failed: %v", err)
	}

	if registry.Count() != 3 {
		t.Errorf("Count() returned %d, expected 3", registry.Count())
	}

	// Test List
	list := registry.List()
	if len(list) != 3 {
		t.Errorf("List() returned %d items, expected 3", len(list))
	}

	// Test Unregister
	err = registry.Unregister("func1")
	if err != nil {
		t.Errorf("Unregister failed: %v", err)
	}

	if registry.Has("func1") {
		t.Error("Function still exists after unregister")
	}

	// Test unregister non-existent
	err = registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error for unregistering non-existent function")
	}

	// Test Clear
	registry.Clear()
	if registry.Count() != 0 {
		t.Errorf("Count() returned %d after Clear(), expected 0", registry.Count())
	}
}

func TestCreateExecutor(t *testing.T) {
	registry := NewRegistry()

	def := FunctionDefinition{
		Name:        "test_func",
		Description: "Test function",
		Parameters:  map[string]interface{}{"type": "object"},
	}
	registry.Register(def)

	// Create executor
	executor := registry.CreateExecutor(func(name string, params json.RawMessage) (interface{}, error) {
		return map[string]string{"result": "success", "function": name}, nil
	})

	// Test successful execution
	call := FunctionCall{
		Name:       "test_func",
		Parameters: json.RawMessage(`{"param": "value"}`),
	}

	response, err := executor(call)
	if err != nil {
		t.Errorf("Executor failed: %v", err)
	}

	if response.Error != "" {
		t.Errorf("Response contains error: %s", response.Error)
	}

	// Test non-existent function
	call.Name = "nonexistent"
	response, err = executor(call)
	if err == nil {
		t.Error("Expected error for non-existent function")
	}

	if response.Error == "" {
		t.Error("Response should contain error for non-existent function")
	}
}
