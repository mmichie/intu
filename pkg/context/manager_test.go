package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContextManager(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping context manager test in short mode")
	}
	// Create a temporary directory for storage
	tempDir, err := os.MkdirTemp("", "context-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storagePath := filepath.Join(tempDir, "contexts.json")

	// Create store with test options
	options := PersistentStoreOptions{
		StoragePath:      storagePath,
		AutosaveInterval: 0, // Disable autosave for testing
	}

	store, err := NewPersistentHierarchicalStore(options)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create manager
	manager := NewContextManager(store)

	// Initial active path should be root
	if path := manager.GetActivePath(); path != "/" {
		t.Errorf("Expected initial active path to be '/', got '%s'", path)
	}

	// Create a global context
	_, err = manager.CreateContext("Global", GlobalContext, map[string]interface{}{
		"globalKey": "globalValue",
	}, 0)
	if err != nil {
		t.Fatalf("Failed to create global context: %v", err)
	}

	// Create a session context
	sessionCtx, err := manager.CreateContext("Session", SessionContext, map[string]interface{}{
		"sessionKey": "sessionValue",
	}, 0)
	if err != nil {
		t.Fatalf("Failed to create session context: %v", err)
	}

	// Set active path to the session
	if err := manager.SetActivePath("/" + sessionCtx.Name); err != nil {
		t.Fatalf("Failed to set active path: %v", err)
	}

	// Create a conversation context under the session
	convCtx, err := manager.CreateContext("Conversation", ConversationContext, map[string]interface{}{
		"convKey": "convValue",
	}, 0)
	if err != nil {
		t.Fatalf("Failed to create conversation context: %v", err)
	}

	// Set active path to the conversation
	convPath := "/" + sessionCtx.Name + "/" + convCtx.Name
	if err := manager.SetActivePath(convPath); err != nil {
		t.Fatalf("Failed to set active path to conversation: %v", err)
	}

	// Check active path
	if path := manager.GetActivePath(); path != convPath {
		t.Errorf("Expected active path to be '%s', got '%s'", convPath, path)
	}

	// Get active context data (should include ancestors)
	data, err := manager.GetActiveContextData()
	if err != nil {
		t.Fatalf("Failed to get active context data: %v", err)
	}

	// Should have values from all contexts
	if data["globalKey"] != "globalValue" {
		t.Errorf("Global context data not included in active context data")
	}

	if data["sessionKey"] != "sessionValue" {
		t.Errorf("Session context data not included in active context data")
	}

	if data["convKey"] != "convValue" {
		t.Errorf("Conversation context data not included in active context data")
	}

	// Update a context
	updatedData := map[string]interface{}{
		"convKey":       "updatedValue",
		"additionalKey": "additionalValue",
	}

	_, err = manager.UpdateContext(convCtx.ID, updatedData, true)
	if err != nil {
		t.Fatalf("Failed to update context: %v", err)
	}

	// Check that the update was applied
	updatedCtx, err := manager.GetContext(convCtx.ID)
	if err != nil {
		t.Fatalf("Failed to get updated context: %v", err)
	}

	if updatedCtx.Data["convKey"] != "updatedValue" {
		t.Errorf("Update didn't apply correctly, expected 'updatedValue', got '%v'", updatedCtx.Data["convKey"])
	}

	if updatedCtx.Data["additionalKey"] != "additionalValue" {
		t.Errorf("Merge didn't work correctly, missing additional key")
	}

	// List contexts by type
	globals, err := manager.ListContexts(GlobalContext, nil)
	if err != nil {
		t.Fatalf("Failed to list contexts by type: %v", err)
	}

	if len(globals) != 1 {
		t.Errorf("Expected 1 global context, got %d", len(globals))
	}

	// Find context by name
	found, err := manager.FindContextByName("Conversation", sessionCtx.ID)
	if err != nil {
		t.Fatalf("Failed to find context by name: %v", err)
	}

	if found.ID != convCtx.ID {
		t.Errorf("Found wrong context, expected ID '%s', got '%s'", convCtx.ID, found.ID)
	}

	// Delete a context
	if err := manager.DeleteContext(sessionCtx.ID); err != nil {
		t.Fatalf("Failed to delete context: %v", err)
	}

	// Session should be gone
	_, err = manager.GetContext(sessionCtx.ID)
	if err == nil {
		t.Errorf("Context still exists after deletion")
	}

	// Conversation should be gone too (was child of session)
	_, err = manager.GetContext(convCtx.ID)
	if err == nil {
		t.Errorf("Child context still exists after parent deletion")
	}

	// Save to storage
	if err := manager.SaveToStorage(); err != nil {
		t.Fatalf("Failed to save to storage: %v", err)
	}

	// Create a new manager to load from storage
	store2, err := NewPersistentHierarchicalStore(options)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}

	manager2 := NewContextManager(store2)

	// Check if data was loaded correctly
	loadedGlobals, err := manager2.ListContexts(GlobalContext, nil)
	if err != nil {
		t.Fatalf("Failed to list contexts after loading: %v", err)
	}

	if len(loadedGlobals) != 1 {
		t.Errorf("Expected 1 global context after loading, got %d", len(loadedGlobals))
	}

	// Sessions should still be gone
	loadedSessions, err := manager2.ListContexts(SessionContext, nil)
	if err != nil {
		t.Fatalf("Failed to list sessions after loading: %v", err)
	}

	if len(loadedSessions) != 0 {
		t.Errorf("Expected 0 session contexts after loading, got %d", len(loadedSessions))
	}
}
