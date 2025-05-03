package context

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryContextStore is an in-memory implementation of ContextStore
type MemoryContextStore struct {
	mu       sync.RWMutex
	contexts map[string]*ContextData
}

// NewMemoryContextStore creates a new in-memory context store
func NewMemoryContextStore() *MemoryContextStore {
	return &MemoryContextStore{
		contexts: make(map[string]*ContextData),
	}
}

// Get retrieves a context by ID
func (s *MemoryContextStore) Get(id string) (*ContextData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx, ok := s.contexts[id]
	if !ok {
		return nil, ErrContextNotFound
	}

	// Check if the context has expired
	if ctx.TTL > 0 && getNow().Sub(ctx.Updated) > ctx.TTL {
		// Don't delete here to avoid write lock, will be cleaned up by Cleanup
		return nil, ErrContextNotFound
	}

	// Return a copy to avoid concurrent modification
	return cloneContext(ctx), nil
}

// Set creates or updates a context
func (s *MemoryContextStore) Set(context *ContextData) error {
	if context.ID == "" {
		return fmt.Errorf("context ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if parent exists if specified
	if context.ParentID != "" {
		if _, ok := s.contexts[context.ParentID]; !ok && context.ParentID != "" {
			return fmt.Errorf("parent context %s not found", context.ParentID)
		}
	}

	// Set creation time if not set
	if context.Created.IsZero() {
		context.Created = time.Now()
	}

	// Update the last updated time
	context.Updated = time.Now()

	// Store the context
	s.contexts[context.ID] = cloneContext(context)
	return nil
}

// Delete removes a context by ID
func (s *MemoryContextStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.contexts[id]; !ok {
		return ErrContextNotFound
	}

	// Also delete any children
	for childID, child := range s.contexts {
		if child.ParentID == id {
			delete(s.contexts, childID)
		}
	}

	delete(s.contexts, id)
	return nil
}

// List retrieves all contexts, optionally filtered by type and tags
func (s *MemoryContextStore) List(contextType ContextType, tags []string) ([]*ContextData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ContextData

	for _, ctx := range s.contexts {
		// Check if expired
		if ctx.TTL > 0 && time.Since(ctx.Updated) > ctx.TTL {
			continue
		}

		// Filter by type if specified
		if contextType != "" && ctx.Type != contextType {
			continue
		}

		// Filter by tags if specified
		if len(tags) > 0 && !containsAllTags(ctx.Tags, tags) {
			continue
		}

		result = append(result, cloneContext(ctx))
	}

	return result, nil
}

// GetByPath retrieves a context by a path-like string
func (s *MemoryContextStore) GetByPath(path string) (*ContextData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return nil, ErrInvalidContext
	}

	// First find a context with the given name and no parent
	var currentID string
	for id, ctx := range s.contexts {
		if ctx.Name == parts[0] && ctx.ParentID == "" {
			currentID = id
			break
		}
	}

	if currentID == "" {
		return nil, ErrContextNotFound
	}

	// Now traverse the path
	for i := 1; i < len(parts); i++ {
		found := false
		for id, ctx := range s.contexts {
			if ctx.Name == parts[i] && ctx.ParentID == currentID {
				currentID = id
				found = true
				break
			}
		}
		if !found {
			return nil, ErrContextNotFound
		}
	}

	ctx, ok := s.contexts[currentID]
	if !ok {
		return nil, ErrContextNotFound
	}

	// Check if expired
	if ctx.TTL > 0 && time.Since(ctx.Updated) > ctx.TTL {
		return nil, ErrContextNotFound
	}

	return cloneContext(ctx), nil
}

// GetChildren retrieves all child contexts of a given parent ID
func (s *MemoryContextStore) GetChildren(parentID string) ([]*ContextData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if parentID != "" {
		if _, ok := s.contexts[parentID]; !ok {
			return nil, ErrContextNotFound
		}
	}

	var result []*ContextData
	for _, ctx := range s.contexts {
		if ctx.ParentID == parentID {
			// Check if expired
			if ctx.TTL > 0 && time.Since(ctx.Updated) > ctx.TTL {
				continue
			}
			result = append(result, cloneContext(ctx))
		}
	}

	return result, nil
}

// Cleanup removes expired contexts
func (s *MemoryContextStore) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := getNow()
	for id, ctx := range s.contexts {
		if ctx.TTL > 0 && now.Sub(ctx.Updated) > ctx.TTL {
			delete(s.contexts, id)
		}
	}

	return nil
}

// Helper functions

// cloneContext creates a deep copy of a context
func cloneContext(ctx *ContextData) *ContextData {
	// Using JSON marshaling for deep copy
	data, _ := json.Marshal(ctx)
	var clone ContextData
	_ = json.Unmarshal(data, &clone)
	return &clone
}

// containsAllTags checks if a slice contains all the specified tags
func containsAllTags(haystack, needles []string) bool {
	if len(needles) == 0 {
		return true
	}
	if len(haystack) == 0 {
		return false
	}

	// Create a map for O(1) lookups
	tagMap := make(map[string]bool)
	for _, tag := range haystack {
		tagMap[tag] = true
	}

	// Check if all needles are in the map
	for _, needle := range needles {
		if !tagMap[needle] {
			return false
		}
	}
	return true
}
