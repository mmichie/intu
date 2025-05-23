package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigStore(t *testing.T) {
	// Create temp directory for tests
	tempDir, err := os.MkdirTemp("", "pipeline-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "pipelines.json")
	store := NewConfigStore(configPath)

	t.Run("Load empty store", func(t *testing.T) {
		err := store.Load()
		assert.NoError(t, err)
		assert.Empty(t, store.List())
	})

	t.Run("Add configuration", func(t *testing.T) {
		config := &PipelineConfig{
			Name:     "test-simple",
			Type:     PipelineTypeSimple,
			Provider: "openai",
		}

		err := store.Add(config)
		assert.NoError(t, err)

		// Verify it was saved
		assert.True(t, store.Exists("test-simple"))
		assert.Contains(t, store.List(), "test-simple")
	})

	t.Run("Get configuration", func(t *testing.T) {
		config, err := store.Get("test-simple")
		require.NoError(t, err)
		assert.Equal(t, "test-simple", config.Name)
		assert.Equal(t, PipelineTypeSimple, config.Type)
		assert.Equal(t, "openai", config.Provider)
	})

	t.Run("Get non-existent configuration", func(t *testing.T) {
		_, err := store.Get("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Update configuration", func(t *testing.T) {
		config := &PipelineConfig{
			Name:        "test-simple",
			Type:        PipelineTypeSimple,
			Provider:    "claude",
			Description: "Updated description",
		}

		err := store.Add(config)
		assert.NoError(t, err)

		// Verify update
		updated, err := store.Get("test-simple")
		require.NoError(t, err)
		assert.Equal(t, "claude", updated.Provider)
		assert.Equal(t, "Updated description", updated.Description)
	})

	t.Run("Delete configuration", func(t *testing.T) {
		err := store.Delete("test-simple")
		assert.NoError(t, err)
		assert.False(t, store.Exists("test-simple"))
		assert.NotContains(t, store.List(), "test-simple")
	})

	t.Run("Delete non-existent configuration", func(t *testing.T) {
		err := store.Delete("non-existent")
		assert.Error(t, err)
	})

	t.Run("Multiple configurations", func(t *testing.T) {
		configs := []*PipelineConfig{
			{
				Name:     "pipeline1",
				Type:     PipelineTypeSimple,
				Provider: "openai",
			},
			{
				Name:      "pipeline2",
				Type:      PipelineTypeSerial,
				Providers: []string{"openai", "claude"},
			},
			{
				Name:      "pipeline3",
				Type:      PipelineTypeParallel,
				Providers: []string{"openai", "claude"},
				Combiner:  CombinerTypeConcat,
			},
		}

		for _, cfg := range configs {
			err := store.Add(cfg)
			require.NoError(t, err)
		}

		// Verify all were added
		list := store.List()
		assert.Len(t, list, 3)
		assert.Contains(t, list, "pipeline1")
		assert.Contains(t, list, "pipeline2")
		assert.Contains(t, list, "pipeline3")

		// Verify ListConfigs
		allConfigs := store.ListConfigs()
		assert.Len(t, allConfigs, 3)
	})

	t.Run("Filter by type", func(t *testing.T) {
		simpleConfigs := store.FilterByType(PipelineTypeSimple)
		assert.Len(t, simpleConfigs, 1)
		assert.Equal(t, "pipeline1", simpleConfigs[0].Name)

		serialConfigs := store.FilterByType(PipelineTypeSerial)
		assert.Len(t, serialConfigs, 1)
		assert.Equal(t, "pipeline2", serialConfigs[0].Name)
	})

	t.Run("Filter by provider", func(t *testing.T) {
		openaiConfigs := store.FilterByProvider("openai")
		assert.Len(t, openaiConfigs, 3) // All use openai

		claudeConfigs := store.FilterByProvider("claude")
		assert.Len(t, claudeConfigs, 2) // pipeline2 and pipeline3
	})

	t.Run("Export and Import", func(t *testing.T) {
		// Export a config
		data, err := store.Export("pipeline1")
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Delete it
		err = store.Delete("pipeline1")
		require.NoError(t, err)

		// Import it back
		err = store.Import(data)
		assert.NoError(t, err)
		assert.True(t, store.Exists("pipeline1"))
	})

	t.Run("Clear all", func(t *testing.T) {
		err := store.Clear()
		assert.NoError(t, err)
		assert.Empty(t, store.List())
	})

	t.Run("Persistence", func(t *testing.T) {
		// Add a config
		config := &PipelineConfig{
			Name:     "persistent",
			Type:     PipelineTypeSimple,
			Provider: "openai",
		}
		err := store.Add(config)
		require.NoError(t, err)

		// Create new store instance
		store2 := NewConfigStore(configPath)
		err = store2.Load()
		require.NoError(t, err)

		// Verify config was persisted
		assert.True(t, store2.Exists("persistent"))
		loaded, err := store2.Get("persistent")
		require.NoError(t, err)
		assert.Equal(t, "persistent", loaded.Name)
	})

	t.Run("Backup and Restore", func(t *testing.T) {
		// Add some configs
		configs := []*PipelineConfig{
			{
				Name:     "backup1",
				Type:     PipelineTypeSimple,
				Provider: "openai",
			},
			{
				Name:     "backup2",
				Type:     PipelineTypeSimple,
				Provider: "claude",
			},
		}

		for _, cfg := range configs {
			err := store.Add(cfg)
			require.NoError(t, err)
		}

		// Create backup
		err = store.Backup()
		assert.NoError(t, err)

		// Find backup file
		backupFiles, _ := filepath.Glob(configPath + ".backup.*")
		require.Len(t, backupFiles, 1)
		backupPath := backupFiles[0]
		defer os.Remove(backupPath)

		// Clear store
		err = store.Clear()
		require.NoError(t, err)

		// Restore from backup
		err = store.Restore(backupPath)
		assert.NoError(t, err)

		// Verify restored
		assert.True(t, store.Exists("backup1"))
		assert.True(t, store.Exists("backup2"))
	})

	t.Run("Invalid configuration validation", func(t *testing.T) {
		invalidConfig := &PipelineConfig{
			Name: "invalid",
			Type: PipelineTypeSimple,
			// Missing provider
		}

		err := store.Add(invalidConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider")
	})

	t.Run("Concurrent access", func(t *testing.T) {
		// Test concurrent operations
		done := make(chan bool)

		// Writer
		go func() {
			for i := 0; i < 10; i++ {
				config := &PipelineConfig{
					Name:     fmt.Sprintf("concurrent-%d", i),
					Type:     PipelineTypeSimple,
					Provider: "openai",
				}
				store.Add(config)
			}
			done <- true
		}()

		// Reader
		go func() {
			for i := 0; i < 10; i++ {
				store.List()
				store.ListConfigs()
			}
			done <- true
		}()

		// Wait for both
		<-done
		<-done

		// Verify some were added
		list := store.List()
		assert.NotEmpty(t, list)
	})
}

func TestDefaultConfigStore(t *testing.T) {
	// Test default store location
	store := DefaultConfigStore()
	assert.NotNil(t, store)
	assert.Contains(t, store.filePath, ".intu")
	assert.Contains(t, store.filePath, "pipelines.json")
}
