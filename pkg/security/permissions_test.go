package security

import (
	"testing"
)

func TestPermissionLevel_String(t *testing.T) {
	tests := []struct {
		level PermissionLevel
		want  string
	}{
		{PermissionReadOnly, "read-only"},
		{PermissionShellExec, "shell-execution"},
		{PermissionFileWrite, "file-write"},
		{PermissionNetwork, "network"},
		{PermissionLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("PermissionLevel.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPermissionManager_ReadOnly(t *testing.T) {
	// Create a permission manager with a prompt that always denies
	// This should not matter for read-only operations
	pm, err := NewPermissionManager(DenyPrompt())
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	// Create a read-only permission request
	req := PermissionRequest{
		ToolName: "LS",
		ToolDesc: "List directory contents",
		Level:    PermissionReadOnly,
	}

	// Check permission - should be allowed even with deny prompt
	err = pm.CheckPermission(req)
	if err != nil {
		t.Errorf("CheckPermission() for read-only tool returned error: %v", err)
	}
}

func TestPermissionManager_AllowDisallowTool(t *testing.T) {
	// Create a permission manager with a prompt that always grants
	// to isolate the AllowTool/DisallowTool behavior
	pm, err := NewPermissionManager(NoPrompt())
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	toolName := "TestTool"

	// Disallow the tool
	pm.DisallowTool(toolName)

	// Create a permission request
	req := PermissionRequest{
		ToolName: toolName,
		ToolDesc: "Test tool",
		Level:    PermissionShellExec,
		Command:  "echo test",
	}

	// Check permission - should be denied
	err = pm.CheckPermission(req)
	if err == nil {
		t.Errorf("CheckPermission() for disallowed tool did not return error")
	}

	// Allow the tool
	pm.AllowTool(toolName)

	// Check permission again - should be allowed now
	err = pm.CheckPermission(req)
	if err != nil {
		t.Errorf("CheckPermission() for allowed tool returned error: %v", err)
	}
}

func TestPermissionManager_SetAllowedTools(t *testing.T) {
	// Create a permission manager with a prompt that always grants
	// This ensures we test only the SetAllowedTools logic, not the prompt
	pm, err := NewPermissionManager(NoPrompt())
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	// Set allowed tools
	allowedTools := []string{"Tool1", "Tool2"}
	pm.SetAllowedTools(allowedTools)

	// Check allowed tool
	req1 := PermissionRequest{
		ToolName: "Tool1",
		ToolDesc: "Test tool 1",
		Level:    PermissionShellExec,
		Command:  "echo test",
	}

	err = pm.CheckPermission(req1)
	if err != nil {
		t.Errorf("CheckPermission() for allowed tool returned error: %v", err)
	}

	// Check disallowed tool - we need to switch to DenyPrompt here
	// to test that the tool is actually denied
	pm.prompt = DenyPrompt()

	req2 := PermissionRequest{
		ToolName: "Tool3",
		ToolDesc: "Test tool 3",
		Level:    PermissionShellExec,
		Command:  "echo test",
	}

	err = pm.CheckPermission(req2)
	if err == nil {
		t.Errorf("CheckPermission() for disallowed tool did not return error")
	}
}

func TestPromptFunctions(t *testing.T) {
	// Test NoPrompt
	noPrompt := NoPrompt()
	resp, err := noPrompt(PermissionRequest{})
	if err != nil {
		t.Errorf("NoPrompt() returned error: %v", err)
	}
	if resp != PermissionGrantedOnce {
		t.Errorf("NoPrompt() = %v, want %v", resp, PermissionGrantedOnce)
	}

	// Test DenyPrompt
	denyPrompt := DenyPrompt()
	resp, err = denyPrompt(PermissionRequest{})
	if err != nil {
		t.Errorf("DenyPrompt() returned error: %v", err)
	}
	if resp != PermissionDenied {
		t.Errorf("DenyPrompt() = %v, want %v", resp, PermissionDenied)
	}

	// Test AllowListPrompt
	allowList := map[string]bool{
		"AllowedTool": true,
		"DeniedTool":  false,
	}

	allowListPrompt := AllowListPrompt(allowList)

	// Test allowed tool
	resp, err = allowListPrompt(PermissionRequest{ToolName: "AllowedTool"})
	if err != nil {
		t.Errorf("AllowListPrompt() returned error: %v", err)
	}
	if resp != PermissionGrantedAlways {
		t.Errorf("AllowListPrompt() for allowed tool = %v, want %v", resp, PermissionGrantedAlways)
	}

	// Test denied tool
	resp, err = allowListPrompt(PermissionRequest{ToolName: "DeniedTool"})
	if err != nil {
		t.Errorf("AllowListPrompt() returned error: %v", err)
	}
	if resp != PermissionDenied {
		t.Errorf("AllowListPrompt() for denied tool = %v, want %v", resp, PermissionDenied)
	}

	// Test unknown tool
	resp, err = allowListPrompt(PermissionRequest{ToolName: "UnknownTool"})
	if err != nil {
		t.Errorf("AllowListPrompt() returned error: %v", err)
	}
	if resp != PermissionDenied {
		t.Errorf("AllowListPrompt() for unknown tool = %v, want %v", resp, PermissionDenied)
	}
}
