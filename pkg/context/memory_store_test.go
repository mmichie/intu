package context

import (
	"testing"
	"time"
)

// Mock time for testing
var (
	mockTime       = time.Now()
	originalGetNow = getNow
)

func init() {
	// Replace getNow with a mock function for testing
	getNow = func() time.Time {
		return mockTime
	}
}

func cleanup() {
	// Restore the original function
	getNow = originalGetNow
}

func TestMemoryContextStore(t *testing.T) {
	// Setup and cleanup
	defer cleanup()
	// Setup
	store := NewMemoryContextStore()

	// Test creating a context
	ctx := &ContextData{
		ID:      "test1",
		Type:    GlobalContext,
		Name:    "Test Context",
		Data:    map[string]interface{}{"key": "value"},
		Tags:    []string{"tag1", "tag2"},
		Created: time.Now(),
		Updated: time.Now(),
	}

	if err := store.Set(ctx); err != nil {
		t.Fatalf("Failed to set context: %v", err)
	}

	// Test retrieving the context
	retrieved, err := store.Get("test1")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if retrieved.ID != "test1" || retrieved.Name != "Test Context" {
		t.Errorf("Retrieved context doesn't match original: %+v", retrieved)
	}

	// Test listing contexts
	list, err := store.List("", nil)
	if err != nil {
		t.Fatalf("Failed to list contexts: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 context, got %d", len(list))
	}

	// Test filtering by type
	list, err = store.List(GlobalContext, nil)
	if err != nil {
		t.Fatalf("Failed to list contexts by type: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 context of type GlobalContext, got %d", len(list))
	}

	list, err = store.List(SessionContext, nil)
	if err != nil {
		t.Fatalf("Failed to list contexts by type: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("Expected 0 contexts of type SessionContext, got %d", len(list))
	}

	// Test filtering by tags
	list, err = store.List("", []string{"tag1"})
	if err != nil {
		t.Fatalf("Failed to list contexts by tag: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 context with tag 'tag1', got %d", len(list))
	}

	list, err = store.List("", []string{"tag1", "tag2"})
	if err != nil {
		t.Fatalf("Failed to list contexts by multiple tags: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 context with tags 'tag1' and 'tag2', got %d", len(list))
	}

	list, err = store.List("", []string{"tag3"})
	if err != nil {
		t.Fatalf("Failed to list contexts by non-existent tag: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("Expected 0 contexts with tag 'tag3', got %d", len(list))
	}

	// Test creating a child context
	childCtx := &ContextData{
		ID:       "child1",
		Type:     SessionContext,
		ParentID: "test1",
		Name:     "Child Context",
		Data:     map[string]interface{}{"childKey": "childValue"},
		Tags:     []string{"childTag"},
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	if err := store.Set(childCtx); err != nil {
		t.Fatalf("Failed to set child context: %v", err)
	}

	// Test getting children
	children, err := store.GetChildren("test1")
	if err != nil {
		t.Fatalf("Failed to get children: %v", err)
	}

	if len(children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(children))
	}

	// Test getting by path
	byPath, err := store.GetByPath("/Test Context/Child Context")
	if err != nil {
		t.Fatalf("Failed to get by path: %v", err)
	}

	if byPath.ID != "child1" {
		t.Errorf("Expected context with ID 'child1', got '%s'", byPath.ID)
	}

	// Save current mockTime
	originalMockTime := mockTime

	// Test TTL expiration
	expiredCtx := &ContextData{
		ID:      "expired",
		Type:    SessionContext,
		Name:    "Expired Context",
		Data:    map[string]interface{}{"key": "value"},
		Created: mockTime.Add(-2 * time.Hour),
		Updated: mockTime.Add(-2 * time.Hour),
		TTL:     time.Hour, // Expired an hour ago
	}

	if err := store.Set(expiredCtx); err != nil {
		t.Fatalf("Failed to set expired context: %v", err)
	}

	// Advance time by 2 hours to ensure it's expired
	mockTime = mockTime.Add(2 * time.Hour)

	// Now the context should not be retrievable
	_, err = store.Get("expired")
	if err == nil {
		t.Errorf("Expected expired context to not be retrievable")
	}

	// Run cleanup to actually remove it
	if err := store.Cleanup(); err != nil {
		t.Fatalf("Failed to cleanup expired contexts: %v", err)
	}

	// Reset mockTime for later tests
	mockTime = originalMockTime

	// Test cleanup
	if err := store.Cleanup(); err != nil {
		t.Fatalf("Failed to run cleanup: %v", err)
	}

	// Delete context
	if err := store.Delete("test1"); err != nil {
		t.Fatalf("Failed to delete context: %v", err)
	}

	// Should be gone
	_, err = store.Get("test1")
	if err == nil {
		t.Errorf("Context still exists after deletion")
	}

	// Child should be gone too
	_, err = store.Get("child1")
	if err == nil {
		t.Errorf("Child context still exists after parent deletion")
	}
}
