package context

import (
	"testing"
)

func TestHierarchicalMemoryStore(t *testing.T) {
	// Setup
	store := NewHierarchicalMemoryStore()

	// Create a hierarchy of contexts
	rootCtx := &ContextData{
		ID:   "root",
		Type: GlobalContext,
		Name: "Root",
		Data: map[string]interface{}{"rootKey": "rootValue"},
		Tags: []string{"root"},
	}

	if err := store.Set(rootCtx); err != nil {
		t.Fatalf("Failed to set root context: %v", err)
	}

	child1Ctx := &ContextData{
		ID:       "child1",
		Type:     SessionContext,
		ParentID: "root",
		Name:     "Child1",
		Data:     map[string]interface{}{"child1Key": "child1Value"},
		Tags:     []string{"child", "level1"},
	}

	if err := store.Set(child1Ctx); err != nil {
		t.Fatalf("Failed to set child1 context: %v", err)
	}

	child2Ctx := &ContextData{
		ID:       "child2",
		Type:     SessionContext,
		ParentID: "root",
		Name:     "Child2",
		Data:     map[string]interface{}{"child2Key": "child2Value"},
		Tags:     []string{"child", "level1"},
	}

	if err := store.Set(child2Ctx); err != nil {
		t.Fatalf("Failed to set child2 context: %v", err)
	}

	grandchildCtx := &ContextData{
		ID:       "grandchild",
		Type:     ConversationContext,
		ParentID: "child1",
		Name:     "Grandchild",
		Data:     map[string]interface{}{"grandchildKey": "grandchildValue"},
		Tags:     []string{"grandchild", "level2"},
	}

	if err := store.Set(grandchildCtx); err != nil {
		t.Fatalf("Failed to set grandchild context: %v", err)
	}

	// Test GetAncestors
	ancestors, err := store.GetAncestors("grandchild")
	if err != nil {
		t.Fatalf("Failed to get ancestors: %v", err)
	}

	if len(ancestors) != 2 {
		t.Errorf("Expected 2 ancestors, got %d", len(ancestors))
	}

	if ancestors[0].ID != "child1" {
		t.Errorf("Expected first ancestor to be 'child1', got '%s'", ancestors[0].ID)
	}

	if ancestors[1].ID != "root" {
		t.Errorf("Expected second ancestor to be 'root', got '%s'", ancestors[1].ID)
	}

	// Test GetDescendants
	descendants, err := store.GetDescendants("root")
	if err != nil {
		t.Fatalf("Failed to get descendants: %v", err)
	}

	if len(descendants) != 3 {
		t.Errorf("Expected 3 descendants, got %d", len(descendants))
	}

	// Test Move
	if err := store.Move("child2", "child1"); err != nil {
		t.Fatalf("Failed to move context: %v", err)
	}

	// Check new parent
	moved, err := store.Get("child2")
	if err != nil {
		t.Fatalf("Failed to get moved context: %v", err)
	}

	if moved.ParentID != "child1" {
		t.Errorf("Expected parent ID to be 'child1', got '%s'", moved.ParentID)
	}

	// Test circular reference detection
	err = store.Move("child1", "grandchild")
	if err == nil {
		t.Errorf("Expected error for circular reference, got nil")
	}

	// Test GetDescendants after move
	descendants, err = store.GetDescendants("child1")
	if err != nil {
		t.Fatalf("Failed to get descendants after move: %v", err)
	}

	if len(descendants) != 2 {
		t.Errorf("Expected 2 descendants for 'child1', got %d", len(descendants))
	}

	// Test GetByPath after move
	byPath, err := store.GetByPath("/Root/Child1/Child2")
	if err != nil {
		t.Fatalf("Failed to get by path after move: %v", err)
	}

	if byPath.ID != "child2" {
		t.Errorf("Expected context with ID 'child2', got '%s'", byPath.ID)
	}

	// Test delete with hierarchical structure
	if err := store.Delete("child1"); err != nil {
		t.Fatalf("Failed to delete context: %v", err)
	}

	// Child1 should be gone
	_, err = store.Get("child1")
	if err == nil {
		t.Errorf("Context still exists after deletion")
	}

	// Child2 should be gone too (was child of child1)
	_, err = store.Get("child2")
	if err == nil {
		t.Errorf("Child context still exists after parent deletion")
	}

	// Grandchild should be gone too (was child of child1)
	_, err = store.Get("grandchild")
	if err == nil {
		t.Errorf("Grandchild context still exists after parent deletion")
	}
}
