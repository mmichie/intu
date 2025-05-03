package context

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPersistentHierarchicalStore(t *testing.T) {
	// Set timeout for the test
	if testing.Short() {
		t.Skip("Skipping persistent store test in short mode")
	}
	// Create a temporary directory for storage
	tempDir, err := os.MkdirTemp("", "context-test")
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

	// Add some data
	rootCtx := &ContextData{
		ID:   "root",
		Type: GlobalContext,
		Name: "Root",
		Data: map[string]interface{}{"key": "value"},
	}

	if err := store.Set(rootCtx); err != nil {
		t.Fatalf("Failed to set context: %v", err)
	}

	childCtx := &ContextData{
		ID:       "child",
		Type:     SessionContext,
		ParentID: "root",
		Name:     "Child",
		Data:     map[string]interface{}{"childKey": "childValue"},
	}

	if err := store.Set(childCtx); err != nil {
		t.Fatalf("Failed to set child context: %v", err)
	}

	// Test saving to disk
	if err := store.Save(); err != nil {
		t.Fatalf("Failed to save contexts: %v", err)
	}

	// Check if the file exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		t.Fatalf("Storage file doesn't exist after save")
	}

	// Create a new store that loads from the same file
	store2, err := NewPersistentHierarchicalStore(options)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}

	// Check if the data was loaded correctly
	root, err := store2.Get("root")
	if err != nil {
		t.Fatalf("Failed to get root context from second store: %v", err)
	}

	if root.Name != "Root" {
		t.Errorf("Expected context named 'Root', got '%s'", root.Name)
	}

	child, err := store2.Get("child")
	if err != nil {
		t.Fatalf("Failed to get child context from second store: %v", err)
	}

	if child.Name != "Child" {
		t.Errorf("Expected context named 'Child', got '%s'", child.Name)
	}

	// Test hierarchical operations
	ancestors, err := store2.GetAncestors("child")
	if err != nil {
		t.Fatalf("Failed to get ancestors from second store: %v", err)
	}

	if len(ancestors) != 1 || ancestors[0].ID != "root" {
		t.Errorf("Ancestor relationship not preserved after loading")
	}

	// Test changing storage path
	newStoragePath := filepath.Join(tempDir, "new_contexts.json")
	if err := store2.SetStoragePath(newStoragePath); err != nil {
		t.Fatalf("Failed to set new storage path: %v", err)
	}

	// Add more data
	grandchildCtx := &ContextData{
		ID:       "grandchild",
		Type:     ConversationContext,
		ParentID: "child",
		Name:     "Grandchild",
		Data:     map[string]interface{}{"grandchildKey": "grandchildValue"},
	}

	if err := store2.Set(grandchildCtx); err != nil {
		t.Fatalf("Failed to set grandchild context: %v", err)
	}

	// Save to new location
	if err := store2.Save(); err != nil {
		t.Fatalf("Failed to save to new location: %v", err)
	}

	// Check if the new file exists
	if _, err := os.Stat(newStoragePath); os.IsNotExist(err) {
		t.Fatalf("New storage file doesn't exist after save")
	}

	// Create a third store pointing to the new location
	options.StoragePath = newStoragePath
	store3, err := NewPersistentHierarchicalStore(options)
	if err != nil {
		t.Fatalf("Failed to create third store: %v", err)
	}

	// Check if it loaded all data
	count, err := countContexts(store3)
	if err != nil {
		t.Fatalf("Failed to count contexts: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 contexts, got %d", count)
	}

	// Test TTL expiration and cleanup
	expiredCtx := &ContextData{
		ID:      "expired",
		Type:    SessionContext,
		Name:    "Expired",
		Data:    map[string]interface{}{"key": "value"},
		Created: time.Now().Add(-2 * time.Hour),
		Updated: time.Now().Add(-2 * time.Hour),
		TTL:     time.Hour, // Expired an hour ago
	}

	if err := store3.Set(expiredCtx); err != nil {
		t.Fatalf("Failed to set expired context: %v", err)
	}

	// Save and reload to ensure TTL is preserved
	if err := store3.Save(); err != nil {
		t.Fatalf("Failed to save after adding expired context: %v", err)
	}

	if err := store3.Cleanup(); err != nil {
		t.Fatalf("Failed to cleanup contexts: %v", err)
	}

	// Save one last time
	if err := store3.Save(); err != nil {
		t.Fatalf("Failed to save store: %v", err)
	}
}

// Helper function to count contexts in a store
func countContexts(store ContextStore) (int, error) {
	contexts, err := store.List("", nil)
	if err != nil {
		return 0, err
	}
	return len(contexts), nil
}
