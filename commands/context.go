package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mmichie/intu/pkg/context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InitContextCommand initializes and adds the context command to the root command
func InitContextCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(registerContextCommand())
}

// registerContextCommand registers the context command and its subcommands
func registerContextCommand() *cobra.Command {
	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Manage conversation contexts",
		Long:  `Manage, create, and manipulate conversation contexts for AI interactions.`,
	}

	// Add subcommands
	contextCmd.AddCommand(registerContextListCommand())
	contextCmd.AddCommand(registerContextCreateCommand())
	contextCmd.AddCommand(registerContextGetCommand())
	contextCmd.AddCommand(registerContextDeleteCommand())
	contextCmd.AddCommand(registerContextUpdateCommand())
	contextCmd.AddCommand(registerContextSetActiveCommand())
	contextCmd.AddCommand(registerContextShowActiveCommand())

	return contextCmd
}

// registerContextListCommand registers the 'context list' command
func registerContextListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all contexts",
		Long:  `List all available contexts, optionally filtered by type and tags.`,
		RunE:  runContextListCommand,
	}

	cmd.Flags().String("type", "", "Filter by context type (global, session, conversation, tool)")
	cmd.Flags().StringSlice("tags", nil, "Filter by tags (comma-separated)")
	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

// registerContextCreateCommand registers the 'context create' command
func registerContextCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new context",
		Long:  `Create a new context with the specified name and properties.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runContextCreateCommand,
	}

	cmd.Flags().String("type", "global", "Context type (global, session, conversation, tool)")
	cmd.Flags().String("parent", "", "Parent context ID or path")
	cmd.Flags().StringSlice("tags", nil, "Tags (comma-separated)")
	cmd.Flags().StringToString("data", nil, "Data key-value pairs (format: key=value)")
	cmd.Flags().Duration("ttl", 0, "Time-to-live duration (e.g., 24h, 30m)")

	return cmd
}

// registerContextGetCommand registers the 'context get' command
func registerContextGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [id_or_path]",
		Short: "Get context details",
		Long:  `Get detailed information about a specific context.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runContextGetCommand,
	}

	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("with-ancestors", false, "Include ancestor contexts")
	cmd.Flags().Bool("with-children", false, "Include child contexts")

	return cmd
}

// registerContextDeleteCommand registers the 'context delete' command
func registerContextDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [id_or_path]",
		Short: "Delete a context",
		Long:  `Delete a context and all its children.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runContextDeleteCommand,
	}

	cmd.Flags().Bool("force", false, "Force deletion without confirmation")

	return cmd
}

// registerContextUpdateCommand registers the 'context update' command
func registerContextUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [id_or_path]",
		Short: "Update a context",
		Long:  `Update an existing context with new data.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runContextUpdateCommand,
	}

	cmd.Flags().StringToString("data", nil, "Data key-value pairs (format: key=value)")
	cmd.Flags().StringSlice("add-tags", nil, "Tags to add (comma-separated)")
	cmd.Flags().StringSlice("remove-tags", nil, "Tags to remove (comma-separated)")
	cmd.Flags().Bool("replace", false, "Replace data instead of merging")
	cmd.Flags().String("rename", "", "New name for the context")
	cmd.Flags().String("move-to", "", "Move to a new parent (ID or path)")

	return cmd
}

// registerContextSetActiveCommand registers the 'context active set' command
func registerContextSetActiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-active [path]",
		Short: "Set the active context path",
		Long:  `Set the currently active context path for AI interactions.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runContextSetActiveCommand,
	}

	return cmd
}

// registerContextShowActiveCommand registers the 'context active show' command
func registerContextShowActiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active",
		Short: "Show active context information",
		Long:  `Display information about the currently active context path and its data.`,
		RunE:  runContextShowActiveCommand,
	}

	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("data-only", false, "Show only the merged data")

	return cmd
}

// Command implementations

// runContextListCommand implements the 'context list' command
func runContextListCommand(cmd *cobra.Command, args []string) error {
	manager, err := getContextManager()
	if err != nil {
		return err
	}

	typeStr, _ := cmd.Flags().GetString("type")
	tagsSlice, _ := cmd.Flags().GetStringSlice("tags")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	var contextType context.ContextType
	if typeStr != "" {
		contextType = context.ContextType(typeStr)
	}

	contexts, err := manager.ListContexts(contextType, tagsSlice)
	if err != nil {
		return fmt.Errorf("failed to list contexts: %w", err)
	}

	if jsonOutput {
		jsonData, err := json.MarshalIndent(contexts, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal contexts to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Table output
	if len(contexts) == 0 {
		fmt.Println("No contexts found.")
		return nil
	}

	fmt.Printf("%-36s %-10s %-20s %-20s %s\n", "ID", "TYPE", "NAME", "UPDATED", "TAGS")
	fmt.Println(strings.Repeat("-", 100))

	for _, ctx := range contexts {
		fmt.Printf("%-36s %-10s %-20s %-20s %s\n",
			ctx.ID,
			ctx.Type,
			ctx.Name,
			ctx.Updated.Format("2006-01-02 15:04:05"),
			strings.Join(ctx.Tags, ", "),
		)
	}

	return nil
}

// runContextCreateCommand implements the 'context create' command
func runContextCreateCommand(cmd *cobra.Command, args []string) error {
	manager, err := getContextManager()
	if err != nil {
		return err
	}

	name := args[0]
	typeStr, _ := cmd.Flags().GetString("type")
	parentPath, _ := cmd.Flags().GetString("parent")
	tags, _ := cmd.Flags().GetStringSlice("tags")
	dataMap, _ := cmd.Flags().GetStringToString("data")
	ttl, _ := cmd.Flags().GetDuration("ttl")

	contextType := context.ContextType(typeStr)
	if contextType == "" {
		contextType = context.GlobalContext
	}

	// Convert string map to interface map
	data := make(map[string]interface{})
	for k, v := range dataMap {
		data[k] = v
	}

	// If parent is specified, set active path to parent
	if parentPath != "" {
		if err := manager.SetActivePath(parentPath); err != nil {
			return fmt.Errorf("invalid parent path: %w", err)
		}
	}

	// Create the context
	ctx, err := manager.CreateContext(name, contextType, data, ttl)
	if err != nil {
		return fmt.Errorf("failed to create context: %w", err)
	}

	// Add tags if specified
	if len(tags) > 0 {
		context.AddTagsToContext(ctx, tags...)
		if _, err := manager.UpdateContext(ctx.ID, nil, true); err != nil {
			return fmt.Errorf("failed to add tags: %w", err)
		}
	}

	// Save to storage
	if err := manager.SaveToStorage(); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	fmt.Printf("Context created with ID: %s\n", ctx.ID)
	return nil
}

// runContextGetCommand implements the 'context get' command
func runContextGetCommand(cmd *cobra.Command, args []string) error {
	manager, err := getContextManager()
	if err != nil {
		return err
	}

	idOrPath := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")
	withAncestors, _ := cmd.Flags().GetBool("with-ancestors")
	withChildren, _ := cmd.Flags().GetBool("with-children")

	// Get the context
	ctx, err := manager.GetContext(idOrPath)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	if jsonOutput {
		// Build a response object
		response := map[string]interface{}{
			"context": ctx,
		}

		// Add ancestors if requested
		if withAncestors {
			if hstore, ok := manager.GetStore().(context.HierarchicalContextStore); ok {
				ancestors, err := hstore.GetAncestors(ctx.ID)
				if err == nil {
					response["ancestors"] = ancestors
				}
			}
		}

		// Add children if requested
		if withChildren {
			children, err := manager.GetStore().GetChildren(ctx.ID)
			if err == nil {
				response["children"] = children
			}
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal context to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Pretty print
	fmt.Printf("ID:        %s\n", ctx.ID)
	fmt.Printf("Type:      %s\n", ctx.Type)
	fmt.Printf("Name:      %s\n", ctx.Name)
	fmt.Printf("Parent:    %s\n", ctx.ParentID)
	fmt.Printf("Created:   %s\n", ctx.Created.Format(time.RFC3339))
	fmt.Printf("Updated:   %s\n", ctx.Updated.Format(time.RFC3339))
	fmt.Printf("TTL:       %v\n", ctx.TTL)
	fmt.Printf("Tags:      %s\n", strings.Join(ctx.Tags, ", "))
	fmt.Println("Data:")
	jsonData, _ := json.MarshalIndent(ctx.Data, "  ", "  ")
	fmt.Println("  " + strings.ReplaceAll(string(jsonData), "\n", "\n  "))

	// Print ancestors if requested
	if withAncestors {
		if hstore, ok := manager.GetStore().(context.HierarchicalContextStore); ok {
			ancestors, err := hstore.GetAncestors(ctx.ID)
			if err == nil && len(ancestors) > 0 {
				fmt.Println("\nAncestors:")
				for i, ancestor := range ancestors {
					fmt.Printf("  %d. %s (%s)\n", i+1, ancestor.Name, ancestor.ID)
				}
			}
		}
	}

	// Print children if requested
	if withChildren {
		children, err := manager.GetStore().GetChildren(ctx.ID)
		if err == nil && len(children) > 0 {
			fmt.Println("\nChildren:")
			for i, child := range children {
				fmt.Printf("  %d. %s (%s)\n", i+1, child.Name, child.ID)
			}
		}
	}

	return nil
}

// runContextDeleteCommand implements the 'context delete' command
func runContextDeleteCommand(cmd *cobra.Command, args []string) error {
	manager, err := getContextManager()
	if err != nil {
		return err
	}

	idOrPath := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Get the context first to show what we're deleting
	ctx, err := manager.GetContext(idOrPath)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Get children to warn about cascading deletion
	children, err := manager.GetStore().GetChildren(ctx.ID)
	if err != nil {
		return fmt.Errorf("failed to get children: %w", err)
	}

	if !force && len(children) > 0 {
		fmt.Printf("Warning: Deleting context '%s' will also delete %d child contexts.\n", ctx.Name, len(children))
		fmt.Print("Continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Delete the context
	if err := manager.DeleteContext(idOrPath); err != nil {
		return fmt.Errorf("failed to delete context: %w", err)
	}

	// Save changes
	if err := manager.SaveToStorage(); err != nil {
		return fmt.Errorf("failed to save changes: %w", err)
	}

	fmt.Printf("Context '%s' deleted successfully.\n", ctx.Name)
	if len(children) > 0 {
		fmt.Printf("%d child contexts were also deleted.\n", len(children))
	}

	return nil
}

// runContextUpdateCommand implements the 'context update' command
func runContextUpdateCommand(cmd *cobra.Command, args []string) error {
	manager, err := getContextManager()
	if err != nil {
		return err
	}

	idOrPath := args[0]
	dataMap, _ := cmd.Flags().GetStringToString("data")
	addTags, _ := cmd.Flags().GetStringSlice("add-tags")
	removeTags, _ := cmd.Flags().GetStringSlice("remove-tags")
	replace, _ := cmd.Flags().GetBool("replace")
	rename, _ := cmd.Flags().GetString("rename")
	moveTo, _ := cmd.Flags().GetString("move-to")

	// Get the context
	ctx, err := manager.GetContext(idOrPath)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Update name if requested
	if rename != "" {
		ctx.Name = rename
	}

	// Convert string map to interface map
	data := make(map[string]interface{})
	for k, v := range dataMap {
		data[k] = v
	}

	// Add tags if requested
	if len(addTags) > 0 {
		context.AddTagsToContext(ctx, addTags...)
	}

	// Remove tags if requested
	if len(removeTags) > 0 {
		context.RemoveTagsFromContext(ctx, removeTags...)
	}

	// Update the context
	if _, err = manager.UpdateContext(ctx.ID, data, !replace); err != nil {
		return fmt.Errorf("failed to update context: %w", err)
	}

	// Move to new parent if requested
	if moveTo != "" {
		var newParentID string

		// Check if it's a path or ID
		if strings.Contains(moveTo, "/") {
			// It's a path
			parent, err := manager.GetContext(moveTo)
			if err != nil {
				return fmt.Errorf("failed to get parent context: %w", err)
			}
			newParentID = parent.ID
		} else {
			// It's an ID
			parent, err := manager.GetContext(moveTo)
			if err != nil {
				return fmt.Errorf("failed to get parent context: %w", err)
			}
			newParentID = parent.ID
		}

		// Move the context
		if hstore, ok := manager.GetStore().(context.HierarchicalContextStore); ok {
			if err := hstore.Move(ctx.ID, newParentID); err != nil {
				return fmt.Errorf("failed to move context: %w", err)
			}
		} else {
			return fmt.Errorf("context store does not support moving contexts")
		}
	}

	// Save changes
	if err := manager.SaveToStorage(); err != nil {
		return fmt.Errorf("failed to save changes: %w", err)
	}

	fmt.Printf("Context '%s' updated successfully.\n", ctx.Name)
	return nil
}

// runContextSetActiveCommand implements the 'context set-active' command
func runContextSetActiveCommand(cmd *cobra.Command, args []string) error {
	manager, err := getContextManager()
	if err != nil {
		return err
	}

	path := args[0]

	// Handle root path
	if path != "/" {
		// Ensure path is properly formatted
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		// Verify the path exists
		_, err := manager.GetContext(path)
		if err != nil {
			return fmt.Errorf("invalid context path: %w", err)
		}
	}

	// Set active path
	if err := manager.SetActivePath(path); err != nil {
		return fmt.Errorf("failed to set active path: %w", err)
	}

	// Save the setting as well
	saveActivePath(path)

	fmt.Printf("Active context path set to: %s\n", path)
	return nil
}

// runContextShowActiveCommand implements the 'context active show' command
func runContextShowActiveCommand(cmd *cobra.Command, args []string) error {
	manager, err := getContextManager()
	if err != nil {
		return err
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	dataOnly, _ := cmd.Flags().GetBool("data-only")

	path := manager.GetActivePath()

	if dataOnly {
		// Get the merged context data
		data, err := manager.GetActiveContextData()
		if err != nil {
			return fmt.Errorf("failed to get active context data: %w", err)
		}

		if jsonOutput {
			jsonData, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal data to JSON: %w", err)
			}
			fmt.Println(string(jsonData))
		} else {
			// Pretty print
			fmt.Println("Active Context Data:")
			for k, v := range data {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}

		return nil
	}

	// Get full context information
	var currentContext *context.ContextData
	var ancestors []*context.ContextData

	if path != "/" {
		currentContext, err = manager.GetContext(path)
		if err != nil {
			return fmt.Errorf("failed to get active context: %w", err)
		}

		// Get ancestors if applicable
		if hstore, ok := manager.GetStore().(context.HierarchicalContextStore); ok && currentContext.ParentID != "" {
			ancestors, _ = hstore.GetAncestors(currentContext.ID)
		}
	}

	if jsonOutput {
		response := map[string]interface{}{
			"active_path": path,
		}

		if currentContext != nil {
			response["context"] = currentContext
		}

		if len(ancestors) > 0 {
			response["ancestors"] = ancestors
		}

		// Get and include merged data
		data, err := manager.GetActiveContextData()
		if err == nil {
			response["merged_data"] = data
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Pretty print
	fmt.Printf("Active Context Path: %s\n", path)

	if currentContext != nil {
		fmt.Println("\nCurrent Context:")
		fmt.Printf("  ID:     %s\n", currentContext.ID)
		fmt.Printf("  Type:   %s\n", currentContext.Type)
		fmt.Printf("  Name:   %s\n", currentContext.Name)
		fmt.Printf("  Tags:   %s\n", strings.Join(currentContext.Tags, ", "))
	}

	if len(ancestors) > 0 {
		fmt.Println("\nAncestor Contexts:")
		for i, ancestor := range ancestors {
			fmt.Printf("  %d. %s (%s)\n", i+1, ancestor.Name, ancestor.Type)
		}
	}

	// Show merged data
	data, err := manager.GetActiveContextData()
	if err == nil && len(data) > 0 {
		fmt.Println("\nMerged Context Data:")
		for k, v := range data {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	return nil
}

// Helper functions

// getContextManager creates or returns the context manager
func getContextManager() (*context.ContextManager, error) {
	// Get storage path from config
	storagePath := getContextStoragePath()

	// Create options
	options := context.PersistentStoreOptions{
		StoragePath:      storagePath,
		AutosaveInterval: 5 * time.Minute,
	}

	// Create store
	store, err := context.NewPersistentHierarchicalStore(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create context store: %w", err)
	}

	// Create manager
	manager := context.NewContextManager(store)

	// Try to set active path from saved setting
	if path := getActivePath(); path != "" {
		_ = manager.SetActivePath(path) // Ignore error, will default to root
	}

	return manager, nil
}

// getContextStoragePath returns the path for storing contexts
func getContextStoragePath() string {
	// Check config
	path := viper.GetString("context.storage_path")
	if path != "" {
		return path
	}

	// Default to home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".intu/contexts.json"
	}

	return filepath.Join(homeDir, ".intu", "contexts.json")
}

// getActivePath returns the saved active context path
func getActivePath() string {
	return viper.GetString("context.active_path")
}

// saveActivePath saves the active context path to config
func saveActivePath(path string) {
	viper.Set("context.active_path", path)
	viper.WriteConfig()
}
