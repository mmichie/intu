package context

import (
	"crypto/rand"
	"encoding/hex"
	"path"
	"strings"
	"time"
)

// GenerateID generates a random ID for contexts
func GenerateID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// PathToID converts a path to an ID by combining path components
func PathToID(p string) string {
	// Normalize the path
	normalized := path.Clean(p)
	normalized = strings.Trim(normalized, "/")

	// Replace slashes with underscores
	return strings.ReplaceAll(normalized, "/", "_")
}

// BuildPath constructs a path from context components
func BuildPath(components ...string) string {
	var result []string
	for _, component := range components {
		if component != "" {
			result = append(result, component)
		}
	}
	return "/" + strings.Join(result, "/")
}

// AddTagsToContext adds tags to a context
func AddTagsToContext(ctx *ContextData, tags ...string) {
	// Create a map for deduplication
	tagMap := make(map[string]bool)
	for _, tag := range ctx.Tags {
		tagMap[tag] = true
	}

	// Add new tags
	for _, tag := range tags {
		tagMap[tag] = true
	}

	// Convert back to slice
	ctx.Tags = make([]string, 0, len(tagMap))
	for tag := range tagMap {
		ctx.Tags = append(ctx.Tags, tag)
	}
}

// RemoveTagsFromContext removes tags from a context
func RemoveTagsFromContext(ctx *ContextData, tags ...string) {
	// Create a map for lookup
	tagsToRemove := make(map[string]bool)
	for _, tag := range tags {
		tagsToRemove[tag] = true
	}

	// Filter out tags to remove
	filteredTags := make([]string, 0, len(ctx.Tags))
	for _, tag := range ctx.Tags {
		if !tagsToRemove[tag] {
			filteredTags = append(filteredTags, tag)
		}
	}

	ctx.Tags = filteredTags
}

// MergeContextData merges data from source into target
func MergeContextData(target, source *ContextData) {
	if source.Data == nil {
		return
	}

	if target.Data == nil {
		target.Data = make(map[string]interface{})
	}

	// Merge data
	for k, v := range source.Data {
		target.Data[k] = v
	}

	// Merge tags
	AddTagsToContext(target, source.Tags...)

	// Update timestamp
	target.Updated = getNow()
}

// Override time.Now for testing
var getNow = time.Now
