package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PersistentHierarchicalStore adds persistence to the hierarchical store
type PersistentHierarchicalStore struct {
	*HierarchicalMemoryStore
	storagePath   string
	autosave      bool
	autosaveTimer *time.Timer
	mu            sync.RWMutex
}

// PersistentStoreOptions configures the persistent store
type PersistentStoreOptions struct {
	StoragePath      string
	AutosaveInterval time.Duration
}

// DefaultPersistentStoreOptions returns default options
func DefaultPersistentStoreOptions() PersistentStoreOptions {
	return PersistentStoreOptions{
		StoragePath:      "", // Must be specified
		AutosaveInterval: 5 * time.Minute,
	}
}

// NewPersistentHierarchicalStore creates a new persistent hierarchical store
func NewPersistentHierarchicalStore(options PersistentStoreOptions) (*PersistentHierarchicalStore, error) {
	if options.StoragePath == "" {
		return nil, fmt.Errorf("storage path must be specified")
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(options.StoragePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	store := &PersistentHierarchicalStore{
		HierarchicalMemoryStore: NewHierarchicalMemoryStore(),
		storagePath:             options.StoragePath,
		autosave:                options.AutosaveInterval > 0,
	}

	// Try to load existing data
	_ = store.Load() // Ignore error, might be first run

	// Set up autosave if enabled
	if store.autosave && options.AutosaveInterval > 0 {
		store.autosaveTimer = time.AfterFunc(options.AutosaveInterval, func() {
			if err := store.Save(); err != nil {
				fmt.Printf("Autosave failed: %v\n", err)
			}
			// Reschedule next autosave
			store.autosaveTimer.Reset(options.AutosaveInterval)
		})
	}

	return store, nil
}

// SetStoragePath sets the path for persistent storage
func (s *PersistentHierarchicalStore) SetStoragePath(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if path == "" {
		return fmt.Errorf("storage path cannot be empty")
	}

	// Create the directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	s.storagePath = path
	return nil
}

// Load loads contexts from persistent storage
func (s *PersistentHierarchicalStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the file exists
	if _, err := os.Stat(s.storagePath); os.IsNotExist(err) {
		// File doesn't exist, not an error for first run
		return nil
	}

	// Read the file
	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		return fmt.Errorf("failed to read context file: %w", err)
	}

	// Parse the JSON
	var contexts map[string]*ContextData
	if err := json.Unmarshal(data, &contexts); err != nil {
		return fmt.Errorf("failed to parse context file: %w", err)
	}

	// Replace the current contexts
	s.MemoryContextStore.contexts = contexts

	// Cleanup expired contexts
	return s.Cleanup()
}

// Save saves contexts to persistent storage
func (s *PersistentHierarchicalStore) Save() error {
	// Perform cleanup first
	if err := s.Cleanup(); err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Skip if there are no contexts
	if len(s.contexts) == 0 {
		return nil
	}

	// Create a backup of the current file if it exists
	if _, err := os.Stat(s.storagePath); err == nil {
		backupPath := s.storagePath + ".bak"
		if err := os.Rename(s.storagePath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Marshal the contexts to JSON
	data, err := json.MarshalIndent(s.contexts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal contexts: %w", err)
	}

	// Write to a temporary file first
	tempFile := s.storagePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename the temporary file to the target file
	if err := os.Rename(tempFile, s.storagePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// Close gracefully shuts down the store, saving if needed
func (s *PersistentHierarchicalStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop the autosave timer
	if s.autosaveTimer != nil {
		s.autosaveTimer.Stop()
	}

	// Save the current state
	return s.Save()
}

// Override methods that modify the store to save automatically

// Set creates or updates a context and saves if autosave is enabled
func (s *PersistentHierarchicalStore) Set(context *ContextData) error {
	// Use the parent implementation
	err := s.HierarchicalMemoryStore.Set(context)
	if err != nil {
		return err
	}

	// Trigger save if autosave is off but we want to save on modification
	// Here we're not doing that, but this is where you would add that logic

	return nil
}

// Delete removes a context by ID and saves if autosave is enabled
func (s *PersistentHierarchicalStore) Delete(id string) error {
	// Use the parent implementation
	err := s.HierarchicalMemoryStore.Delete(id)
	if err != nil {
		return err
	}

	// Trigger save if autosave is off but we want to save on modification
	// Here we're not doing that, but this is where you would add that logic

	return nil
}

// Move moves a context to a new parent and saves if autosave is enabled
func (s *PersistentHierarchicalStore) Move(id string, newParentID string) error {
	// Use the parent implementation
	err := s.HierarchicalMemoryStore.Move(id, newParentID)
	if err != nil {
		return err
	}

	// Trigger save if autosave is off but we want to save on modification
	// Here we're not doing that, but this is where you would add that logic

	return nil
}
