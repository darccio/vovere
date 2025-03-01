package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"vovere/internal/app/models"
)

// Repository handles file operations for items
type Repository struct {
	basePath string
}

// NewRepository creates a new repository service
func NewRepository(basePath string) *Repository {
	return &Repository{
		basePath: basePath,
	}
}

// BasePath returns the base path of the repository
func (r *Repository) BasePath() string {
	return r.basePath
}

// DeleteItem deletes an item's metadata and content files
func (r *Repository) DeleteItem(item *models.Item) error {
	// Delete metadata file
	metaPath := r.getMetaPath(item)
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete metadata file: %w", err)
	}

	// Delete content file
	contentPath := r.getContentPath(item)
	if err := os.Remove(contentPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete content file: %w", err)
	}

	return nil
}

// ListItems returns all items of a given type
func (r *Repository) ListItems(itemType models.ItemType) ([]*models.Item, error) {
	metaDir := filepath.Join(r.basePath, ".meta", string(itemType)+"s")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	entries, err := os.ReadDir(metaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*models.Item{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var items []*models.Item
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		item, _, err := r.LoadItem(id, itemType)
		if err != nil {
			continue // Skip items that can't be loaded
		}
		items = append(items, item)
	}

	// Sort by modified time, newest first
	sort.Slice(items, func(i, j int) bool {
		return items[i].Modified.After(items[j].Modified)
	})

	return items, nil
}

// SaveItem saves an item's metadata and content
func (r *Repository) SaveItem(item *models.Item, content string) error {
	// Create a tag service
	tagService := NewTagService(r)

	// Store previous tags
	previousTags := make([]string, len(item.Tags))
	copy(previousTags, item.Tags)

	// Only extract tags from content if we don't already have tags set
	// This ensures manually set tags are preserved
	if content != "" && len(item.Tags) == 0 {
		// Extract tags from content only if no tags were manually set
		item.Tags = tagService.ExtractTags(content)
	}

	// Save metadata
	metaPath := r.getMetaPath(item)
	if err := os.MkdirAll(filepath.Dir(metaPath), 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	metaFile, err := os.Create(metaPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer metaFile.Close()

	item.Modified = time.Now().UTC()
	if err := json.NewEncoder(metaFile).Encode(item); err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	// Save content if provided
	if content != "" {
		contentPath := r.getContentPath(item)
		if err := os.MkdirAll(filepath.Dir(contentPath), 0755); err != nil {
			return fmt.Errorf("failed to create content directory: %w", err)
		}

		if err := os.WriteFile(contentPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write content file: %w", err)
		}
	}

	// Update tag relationships (moved outside the content conditional)
	if err := tagService.UpdateItemTags(item, previousTags); err != nil {
		return fmt.Errorf("failed to update tag relationships: %w", err)
	}

	return nil
}

// LoadItem loads an item's metadata and optionally its content
func (r *Repository) LoadItem(id string, itemType models.ItemType) (*models.Item, string, error) {
	item := &models.Item{
		ID:   id,
		Type: itemType,
	}

	// Load metadata
	metaPath := r.getMetaPath(item)
	metaFile, err := os.Open(metaPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer metaFile.Close()

	if err := json.NewDecoder(metaFile).Decode(item); err != nil {
		return nil, "", fmt.Errorf("failed to decode metadata: %w", err)
	}

	// Load content if it exists
	content := ""
	contentPath := r.getContentPath(item)
	if contentBytes, err := os.ReadFile(contentPath); err == nil {
		content = string(contentBytes)
	} else if !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("failed to read content file: %w", err)
	}

	return item, content, nil
}

// UpdateContent updates an item's content
func (r *Repository) UpdateContent(item *models.Item, content string) error {
	// Create content directory if it doesn't exist
	contentPath := r.getContentPath(item)
	if err := os.MkdirAll(filepath.Dir(contentPath), 0755); err != nil {
		return fmt.Errorf("failed to create content directory: %w", err)
	}

	// Store previous tags
	previousTags := make([]string, len(item.Tags))
	copy(previousTags, item.Tags)

	// Extract new tags from content
	tagService := NewTagService(r)
	extractedTags := tagService.ExtractTags(content)

	// Replace the item's tags with the extracted ones
	item.Tags = extractedTags

	// Write content to file
	if err := os.WriteFile(contentPath, []byte(content), 0644); err != nil {
		// Restore original tags if content write fails
		item.Tags = previousTags
		return fmt.Errorf("failed to write content file: %w", err)
	}

	// Update tag relationships
	if err := tagService.UpdateItemTags(item, previousTags); err != nil {
		return fmt.Errorf("failed to update tag relationships: %w", err)
	}

	// Update modification time in metadata
	item.Modified = time.Now().UTC()
	return r.SaveItem(item, "")
}

// getMetaPath returns the metadata file path for an item
func (r *Repository) getMetaPath(item *models.Item) string {
	return filepath.Join(r.basePath, ".meta", string(item.Type)+"s", item.ID+".json")
}

// getContentPath returns the content file path for an item
func (r *Repository) getContentPath(item *models.Item) string {
	return filepath.Join(r.basePath, string(item.Type)+"s", item.ID+".md")
}
