package tools

import (
	"github.com/mmichie/intu/pkg/security"
)

// MockPermissionManager provides a permissive permission manager for testing
type MockPermissionManager struct {
	permLevel security.PermissionLevel
}

// NewMockPermissionManager creates a new mock permission manager that approves everything
func NewMockPermissionManager() (*security.PermissionManager, error) {
	// Use the NoPrompt function which always returns PermissionGrantedAlways
	return security.NewPermissionManager(security.NoPrompt())
}

// NoPermissionManager creates a mock that doesn't do permission checks
func NoPermissionManager() *MockPermissionManager {
	return &MockPermissionManager{
		permLevel: 999, // High level to approve everything
	}
}

// CheckPermission always returns nil (approved) for testing
func (m *MockPermissionManager) CheckPermission(req security.PermissionRequest) error {
	return nil
}
