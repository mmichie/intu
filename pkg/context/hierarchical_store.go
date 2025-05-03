package context

import (
	"fmt"
)

// HierarchicalMemoryStore extends MemoryContextStore with hierarchical capabilities
type HierarchicalMemoryStore struct {
	*MemoryContextStore
}

// NewHierarchicalMemoryStore creates a new hierarchical context store
func NewHierarchicalMemoryStore() *HierarchicalMemoryStore {
	return &HierarchicalMemoryStore{
		MemoryContextStore: NewMemoryContextStore(),
	}
}

// GetAncestors retrieves all ancestor contexts of a given context ID
func (s *HierarchicalMemoryStore) GetAncestors(id string) ([]*ContextData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx, ok := s.contexts[id]
	if !ok {
		return nil, ErrContextNotFound
	}

	var ancestors []*ContextData
	currentID := ctx.ParentID

	// Traverse up the hierarchy
	for currentID != "" {
		parent, ok := s.contexts[currentID]
		if !ok {
			break
		}

		// Check if expired
		if parent.TTL > 0 && isExpired(parent) {
			break
		}

		ancestors = append(ancestors, cloneContext(parent))
		currentID = parent.ParentID
	}

	return ancestors, nil
}

// GetDescendants retrieves all descendant contexts of a given context ID
func (s *HierarchicalMemoryStore) GetDescendants(id string) ([]*ContextData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id != "" {
		if _, ok := s.contexts[id]; !ok {
			return nil, ErrContextNotFound
		}
	}

	var descendants []*ContextData
	queue := []string{id}
	visited := make(map[string]bool)

	// Breadth-first search of the hierarchy
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		// Add all children to the queue
		for childID, child := range s.contexts {
			if child.ParentID == currentID {
				// Check if expired
				if child.TTL > 0 && isExpired(child) {
					continue
				}
				descendants = append(descendants, cloneContext(child))
				queue = append(queue, childID)
			}
		}
	}

	return descendants, nil
}

// Move moves a context to a new parent
func (s *HierarchicalMemoryStore) Move(id string, newParentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get the context to move
	ctx, ok := s.contexts[id]
	if !ok {
		return ErrContextNotFound
	}

	// Check if the new parent exists
	if newParentID != "" {
		if _, ok := s.contexts[newParentID]; !ok {
			return fmt.Errorf("new parent context %s not found", newParentID)
		}

		// Check for circular references
		currentID := newParentID
		for currentID != "" {
			if currentID == id {
				return fmt.Errorf("circular reference detected")
			}
			parent, ok := s.contexts[currentID]
			if !ok {
				break
			}
			currentID = parent.ParentID
		}
	}

	// Update the parent ID
	ctx.ParentID = newParentID
	ctx.Updated = getNow()

	return nil
}

// Helper function to check if a context is expired
func isExpired(ctx *ContextData) bool {
	return ctx.TTL > 0 && getNow().Sub(ctx.Updated) > ctx.TTL
}
