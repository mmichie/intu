package context

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ContextManager provides a high-level interface for context operations
type ContextManager struct {
	store      PersistentContextStore
	mu         sync.RWMutex
	activePath string
}

// GetStore returns the underlying context store
func (m *ContextManager) GetStore() ContextStore {
	return m.store
}

// NewContextManager creates a new context manager
func NewContextManager(store PersistentContextStore) *ContextManager {
	return &ContextManager{
		store:      store,
		activePath: "/",
	}
}

// SetActivePath sets the current active context path
func (m *ContextManager) SetActivePath(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate the path exists unless it's the root
	if path != "/" {
		_, err := m.store.GetByPath(path)
		if err != nil {
			return err
		}
	}

	m.activePath = path
	return nil
}

// GetActivePath returns the current active context path
func (m *ContextManager) GetActivePath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activePath
}

// CreateContext creates a new context at the given path
func (m *ContextManager) CreateContext(name string, contextType ContextType, data map[string]interface{}, ttl time.Duration) (*ContextData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a unique ID
	id := GenerateID()

	// Determine the parent ID
	var parentID string
	if m.activePath != "/" {
		parent, err := m.store.GetByPath(m.activePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent context: %w", err)
		}
		parentID = parent.ID
	}

	// Create the context
	ctx := &ContextData{
		ID:       id,
		Type:     contextType,
		ParentID: parentID,
		Name:     name,
		Data:     data,
		Created:  getNow(),
		Updated:  getNow(),
		TTL:      ttl,
	}

	// Store it
	if err := m.store.Set(ctx); err != nil {
		return nil, fmt.Errorf("failed to store context: %w", err)
	}

	return ctx, nil
}

// GetContext retrieves a context by ID or path
func (m *ContextManager) GetContext(idOrPath string) (*ContextData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if it's a path
	if strings.Contains(idOrPath, "/") {
		return m.store.GetByPath(idOrPath)
	}

	// Otherwise, treat as ID
	return m.store.Get(idOrPath)
}

// UpdateContext updates an existing context
func (m *ContextManager) UpdateContext(id string, data map[string]interface{}, mergeData bool) (*ContextData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the existing context
	ctx, err := m.store.Get(id)
	if err != nil {
		return nil, err
	}

	// Update the context data
	if mergeData {
		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		for k, v := range data {
			ctx.Data[k] = v
		}
	} else {
		ctx.Data = data
	}

	ctx.Updated = getNow()

	// Store the updated context
	if err := m.store.Set(ctx); err != nil {
		return nil, fmt.Errorf("failed to update context: %w", err)
	}

	return ctx, nil
}

// DeleteContext removes a context by ID or path
func (m *ContextManager) DeleteContext(idOrPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var id string

	// Check if it's a path
	if strings.Contains(idOrPath, "/") {
		ctx, err := m.store.GetByPath(idOrPath)
		if err != nil {
			return err
		}
		id = ctx.ID
	} else {
		id = idOrPath
	}

	return m.store.Delete(id)
}

// ListContexts lists contexts filtered by type and tags
func (m *ContextManager) ListContexts(contextType ContextType, tags []string) ([]*ContextData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.store.List(contextType, tags)
}

// GetActiveContextData gets all context data in the current active path, including ancestors
func (m *ContextManager) GetActiveContextData() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Start with an empty map
	result := make(map[string]interface{})

	// First, always include global contexts regardless of the active path
	globals, err := m.store.List(GlobalContext, nil)
	if err != nil {
		return nil, err
	}

	// Merge all global contexts
	for _, c := range globals {
		for k, v := range c.Data {
			result[k] = v
		}
	}

	// If we're at the root path, just return the global contexts
	if m.activePath == "/" {
		return result, nil
	}

	// Get the active context
	ctx, err := m.store.GetByPath(m.activePath)
	if err != nil {
		return nil, err
	}

	// Merge its data (overriding globals if needed)
	for k, v := range ctx.Data {
		result[k] = v
	}

	// Get and merge parent contexts
	if store, ok := m.store.(HierarchicalContextStore); ok {
		ancestors, err := store.GetAncestors(ctx.ID)
		if err != nil {
			return nil, err
		}

		// Merge ancestor data (closest parents override)
		for _, ancestor := range ancestors {
			for k, v := range ancestor.Data {
				// Only add if not already present
				if _, exists := result[k]; !exists {
					result[k] = v
				}
			}
		}
	}

	return result, nil
}

// FindContextByName finds a context by name under a specific parent
func (m *ContextManager) FindContextByName(name string, parentID string) (*ContextData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	children, err := m.store.GetChildren(parentID)
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		if child.Name == name {
			return child, nil
		}
	}

	return nil, ErrContextNotFound
}

// SaveToStorage forces saving the context store
func (m *ContextManager) SaveToStorage() error {
	return m.store.Save()
}

// LoadFromStorage forces loading from the context store
func (m *ContextManager) LoadFromStorage() error {
	return m.store.Load()
}

// CleanupExpiredContexts removes expired contexts
func (m *ContextManager) CleanupExpiredContexts() error {
	return m.store.Cleanup()
}
