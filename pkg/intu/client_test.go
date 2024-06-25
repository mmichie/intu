package intu

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mmichie/intu/internal/fileops"
	"github.com/mmichie/intu/internal/filters"
	"github.com/mmichie/intu/pkg/intu/mocks"
	"github.com/stretchr/testify/assert"
)

//go:generate mockgen -destination=./mocks/mock_fileops.go -package=mocks github.com/mmichie/intu/internal/fileops FileOperator
//go:generate mockgen -destination=./mocks/mock_ai_provider.go -package=mocks github.com/mmichie/intu/internal/ai Provider

type mockFilter struct{}

func (m *mockFilter) Process(content string) string {
	return "Filtered: " + content
}

func (m *mockFilter) Name() string {
	return "mockFilter"
}

func setupTest(t *testing.T) (*gomock.Controller, *Client, *mocks.MockFileOperator) {
	ctrl := gomock.NewController(t)
	mockFileOps := mocks.NewMockFileOperator(ctrl)
	client := &Client{
		FileOps: mockFileOps,
		Filters: []filters.Filter{&mockFilter{}},
	}
	return ctrl, client, mockFileOps
}

func TestCatFiles(t *testing.T) {
	ctrl, client, mockFileOps := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	options := fileops.Options{Extended: true}

	t.Run("Successfully process multiple files", func(t *testing.T) {
		mockFileOps.EXPECT().FindFiles("*.txt", options).Return([]string{"file1.txt", "file2.txt"}, nil)
		mockFileOps.EXPECT().ReadFile("file1.txt").Return("content1", nil)
		mockFileOps.EXPECT().ReadFile("file2.txt").Return("content2", nil)
		mockFileOps.EXPECT().GetExtendedFileInfo("file1.txt", "Filtered: content1").Return(fileops.FileInfo{Filename: "file1.txt", Content: "Filtered: content1"}, nil)
		mockFileOps.EXPECT().GetExtendedFileInfo("file2.txt", "Filtered: content2").Return(fileops.FileInfo{Filename: "file2.txt", Content: "Filtered: content2"}, nil)

		results, err := client.CatFiles(ctx, "*.txt", options)

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "file1.txt", results[0].Filename)
		assert.Equal(t, "Filtered: content1", results[0].Content)
		assert.Equal(t, "file2.txt", results[1].Filename)
		assert.Equal(t, "Filtered: content2", results[1].Content)
	})

	t.Run("Handle FindFiles error", func(t *testing.T) {
		mockFileOps.EXPECT().FindFiles("*.txt", options).Return(nil, assert.AnError)

		results, err := client.CatFiles(ctx, "*.txt", options)

		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "error finding files")
	})

	t.Run("Handle ReadFile error", func(t *testing.T) {
		mockFileOps.EXPECT().FindFiles("*.txt", options).Return([]string{"file1.txt"}, nil)
		mockFileOps.EXPECT().ReadFile("file1.txt").Return("", assert.AnError)

		results, err := client.CatFiles(ctx, "*.txt", options)

		assert.Error(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, err.Error(), "error processing file1.txt")
	})

	t.Run("Handle GetFileInfo error", func(t *testing.T) {
		mockFileOps.EXPECT().FindFiles("*.txt", options).Return([]string{"file1.txt"}, nil)
		mockFileOps.EXPECT().ReadFile("file1.txt").Return("content", nil)
		mockFileOps.EXPECT().GetExtendedFileInfo("file1.txt", "Filtered: content").Return(fileops.FileInfo{}, assert.AnError)

		results, err := client.CatFiles(ctx, "*.txt", options)

		assert.Error(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, err.Error(), "error processing file1.txt")
	})
}

func TestProcessFile(t *testing.T) {
	ctrl, client, mockFileOps := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("Successfully process file with extended info", func(t *testing.T) {
		mockFileOps.EXPECT().ReadFile("file.txt").Return("content", nil)
		mockFileOps.EXPECT().GetExtendedFileInfo("file.txt", "Filtered: content").Return(fileops.FileInfo{Filename: "file.txt", Content: "Filtered: content"}, nil)

		info, err := client.processFile(ctx, "file.txt", true)

		assert.NoError(t, err)
		assert.Equal(t, "file.txt", info.Filename)
		assert.Equal(t, "Filtered: content", info.Content)
	})

	t.Run("Successfully process file with basic info", func(t *testing.T) {
		mockFileOps.EXPECT().ReadFile("file.txt").Return("content", nil)
		mockFileOps.EXPECT().GetBasicFileInfo("file.txt", "Filtered: content").Return(fileops.FileInfo{Filename: "file.txt", Content: "Filtered: content"}, nil)

		info, err := client.processFile(ctx, "file.txt", false)

		assert.NoError(t, err)
		assert.Equal(t, "file.txt", info.Filename)
		assert.Equal(t, "Filtered: content", info.Content)
	})

	t.Run("Handle ReadFile error", func(t *testing.T) {
		mockFileOps.EXPECT().ReadFile("file.txt").Return("", assert.AnError)

		_, err := client.processFile(ctx, "file.txt", true)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("Handle GetFileInfo error", func(t *testing.T) {
		mockFileOps.EXPECT().ReadFile("file.txt").Return("content", nil)
		mockFileOps.EXPECT().GetExtendedFileInfo("file.txt", "Filtered: content").Return(fileops.FileInfo{}, assert.AnError)

		_, err := client.processFile(ctx, "file.txt", true)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get file info")
	})

	t.Run("Handle context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		mockFileOps.EXPECT().ReadFile("file.txt").Return("content", nil)

		_, err := client.processFile(ctx, "file.txt", true)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestGenerateCommitMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockProvider(ctrl)
	client := &Client{
		Provider: mockProvider,
	}

	ctx := context.Background()
	diffOutput := "diff --git a/file.txt b/file.txt\n...\n"
	expectedResponseContent := "feat: Add new feature X\n\nThis commit includes the following changes:\n- Implement feature X\n- Update related tests\n- Refactor existing code for better integration"

	mockProvider.EXPECT().
		GenerateResponse(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, prompt string) (string, error) {
			// Check if the prompt contains the essential parts
			assert.Contains(t, prompt, "Generate a concise git commit message")
			assert.Contains(t, prompt, diffOutput)
			assert.Contains(t, prompt, "Provide a short summary in the first line")
			assert.Contains(t, prompt, "Optimize for a FAANG engineer")
			return expectedResponseContent, nil
		})

	message, err := client.GenerateCommitMessage(ctx, diffOutput)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponseContent, message)
}
