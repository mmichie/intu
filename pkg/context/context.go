// Package context provides a system for managing conversational context and memory
package context

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrContextNotFound = errors.New("context not found")
	ErrInvalidContext  = errors.New("invalid context")
)

// IsContextNotFound checks if an error is a "context not found" error
func IsContextNotFound(err error) bool {
	return err == ErrContextNotFound || err != nil && err.Error() == ErrContextNotFound.Error()
}

// ContextType represents the type of a context
type ContextType string

// Context types
const (
	// Global context applies to the entire application
	GlobalContext ContextType = "global"

	// SessionContext applies to a single user session
	SessionContext ContextType = "session"

	// ConversationContext applies to a single conversation
	ConversationContext ContextType = "conversation"

	// ToolContext applies to a specific tool execution
	ToolContext ContextType = "tool"
)

// ContextData represents a context entry with metadata
type ContextData struct {
	// ID is a unique identifier for this context entry
	ID string `json:"id"`

	// Type is the type of context (global, session, conversation, tool)
	Type ContextType `json:"type"`

	// ParentID is the ID of the parent context (for hierarchical storage)
	ParentID string `json:"parent_id,omitempty"`

	// Name is a human-readable name for this context
	Name string `json:"name"`

	// Data contains the actual context information
	Data map[string]interface{} `json:"data"`

	// Tags allow for flexible categorization and searching
	Tags []string `json:"tags,omitempty"`

	// Created is the time when this context was created
	Created time.Time `json:"created"`

	// Updated is the time when this context was last updated
	Updated time.Time `json:"updated"`

	// TTL is an optional expiration duration for this context (0 means no expiration)
	TTL time.Duration `json:"ttl,omitempty"`
}

// ContextStore defines the interface for storing and retrieving context data
type ContextStore interface {
	// Get retrieves a context by ID
	Get(id string) (*ContextData, error)

	// Set creates or updates a context
	Set(context *ContextData) error

	// Delete removes a context by ID
	Delete(id string) error

	// List retrieves all contexts, optionally filtered by type and tags
	List(contextType ContextType, tags []string) ([]*ContextData, error)

	// GetByPath retrieves a context by a path-like string (e.g., "/global/session/conversation")
	GetByPath(path string) (*ContextData, error)

	// GetChildren retrieves all child contexts of a given parent ID
	GetChildren(parentID string) ([]*ContextData, error)

	// Cleanup removes expired contexts
	Cleanup() error
}

// HierarchicalContextStore extends ContextStore with hierarchical capabilities
type HierarchicalContextStore interface {
	ContextStore

	// GetAncestors retrieves all ancestor contexts of a given context ID
	GetAncestors(id string) ([]*ContextData, error)

	// GetDescendants retrieves all descendant contexts of a given context ID
	GetDescendants(id string) ([]*ContextData, error)

	// Move moves a context to a new parent
	Move(id string, newParentID string) error
}

// PersistentContextStore extends ContextStore with persistence capabilities
type PersistentContextStore interface {
	ContextStore

	// Load loads contexts from persistent storage
	Load() error

	// Save saves contexts to persistent storage
	Save() error

	// SetStoragePath sets the path for persistent storage
	SetStoragePath(path string) error
}
