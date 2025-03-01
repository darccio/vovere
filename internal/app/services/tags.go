package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"vovere/internal/app/models"
)

// TagService handles operations related to tags
type TagService struct {
	repo      *Repository
	cacheLock sync.RWMutex
	tagCache  map[string][]string // map[tagName][]itemIDs
}

// NewTagService creates a new tag service
func NewTagService(repo *Repository) *TagService {
	return &TagService{
		repo:     repo,
		tagCache: make(map[string][]string),
	}
}

// ExtractTags extracts hashtags from content
func (s *TagService) ExtractTags(content string) []string {
	if content == "" {
		return nil
	}

	// Match hashtags with a much broader range of characters
	// Rules:
	// 1. Must start with # preceded by space or beginning of line
	// 2. Cannot contain spaces or blank characters
	// 3. Can contain dots and colons inside, but not at the end

	// Simple approach: find all # followed by non-space characters up to a space or end
	tagFinder := regexp.MustCompile(`(?:^|\s)#([^\s,.;!?]+(?:[.:](?:[^\s,.;!?]+))*)\b`)
	matches := tagFinder.FindAllStringSubmatch(content, -1)

	// Create a map to deduplicate tags
	tagMap := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			tag := match[1]

			// Remove any trailing dots or colons
			tag = strings.TrimRight(tag, ".:")

			// Skip empty tags
			if tag == "" {
				continue
			}

			tagMap[tag] = true
		}
	}

	// Convert map keys to slice
	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}

	return tags
}

// UpdateItemTags updates the tag index for an item
func (s *TagService) UpdateItemTags(item *models.Item, previousTags []string) error {
	// Make a copy of the tags to prevent modifying the slice during operations
	currentTags := make([]string, len(item.Tags))
	copy(currentTags, item.Tags)

	// Create combined ID
	combinedID := fmt.Sprintf("%s:%s", item.ID, item.Type)

	// Clear tag cache outside the lock
	s.cacheLock.Lock()
	s.tagCache = make(map[string][]string)
	s.cacheLock.Unlock()

	// First, remove item from all previous tags that are no longer present
	for _, oldTag := range previousTags {
		if !contains(currentTags, oldTag) {
			if err := s.removeItemFromTag(combinedID, oldTag); err != nil {
				return err
			}
		}
	}

	// Then, add item to all current tags
	for _, tag := range currentTags {
		if err := s.addItemToTag(combinedID, tag); err != nil {
			return err
		}
	}

	return nil
}

// GetItemsByTag returns all items that have a specific tag
func (s *TagService) GetItemsByTag(tag string) ([]*models.Item, error) {
	// Get item IDs for this tag
	itemIDs, err := s.getItemIDsByTag(tag)
	if err != nil {
		return nil, err
	}

	// Load each item
	items := make([]*models.Item, 0, len(itemIDs))
	for _, id := range itemIDs {
		// Extract ID and type from the combined ID
		parts := strings.Split(id, ":")
		if len(parts) != 2 {
			continue // Skip invalid IDs
		}

		itemID := parts[0]
		itemType := models.ItemType(parts[1])

		// Load item
		item, _, err := s.repo.LoadItem(itemID, itemType)
		if err == nil {
			items = append(items, item)
		}
	}

	return items, nil
}

// GetAllTags returns all tags in the repository
func (s *TagService) GetAllTags() ([]string, error) {
	// Path to the tags directory
	tagsDir := filepath.Join(s.repo.BasePath(), ".meta", "tags")

	// Ensure directory exists
	if err := os.MkdirAll(tagsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tags directory: %w", err)
	}

	// List all tag files
	entries, err := os.ReadDir(tagsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tags directory: %w", err)
	}

	tags := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			// Remove the .json extension
			name := entry.Name()
			if filepath.Ext(name) == ".json" {
				tags = append(tags, name[:len(name)-5])
			}
		}
	}

	return tags, nil
}

// GetItemsByMultipleTags returns items that have all the specified tags
func (s *TagService) GetItemsByMultipleTags(tags []string) ([]*models.Item, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	// Get items for the first tag
	items, err := s.GetItemsByTag(tags[0])
	if err != nil {
		return nil, err
	}

	// If there's only one tag, return the items
	if len(tags) == 1 {
		return items, nil
	}

	// Filter items by remaining tags
	for _, tag := range tags[1:] {
		// Get item IDs for this tag
		itemIDs, err := s.getItemIDsByTag(tag)
		if err != nil {
			return nil, err
		}

		// Create a set of item IDs for quick lookup
		itemIDSet := make(map[string]bool)
		for _, id := range itemIDs {
			itemIDSet[id] = true
		}

		// Filter items to only include those with the current tag
		filteredItems := make([]*models.Item, 0, len(items))
		for _, item := range items {
			combinedID := fmt.Sprintf("%s:%s", item.ID, item.Type)
			if itemIDSet[combinedID] {
				filteredItems = append(filteredItems, item)
			}
		}

		items = filteredItems

		// If no items match all tags so far, we can stop early
		if len(items) == 0 {
			return items, nil
		}
	}

	return items, nil
}

// GetTagStatistics returns statistics about tag usage
func (s *TagService) GetTagStatistics() (map[string]int, error) {
	// Get all tags
	tags, err := s.GetAllTags()
	if err != nil {
		return nil, err
	}

	// Create a map of tag to count
	tagStats := make(map[string]int)

	// Count items for each tag
	for _, tag := range tags {
		itemIDs, err := s.getItemIDsByTag(tag)
		if err != nil {
			return nil, err
		}
		tagStats[tag] = len(itemIDs)
	}

	return tagStats, nil
}

// SearchTags returns tags that match the given prefix
func (s *TagService) SearchTags(prefix string) ([]string, error) {
	// Get all tags
	allTags, err := s.GetAllTags()
	if err != nil {
		return nil, err
	}

	// If prefix is empty, return all tags
	if prefix == "" {
		return allTags, nil
	}

	// Filter tags by prefix
	matchingTags := make([]string, 0)
	for _, tag := range allTags {
		if strings.HasPrefix(tag, prefix) {
			matchingTags = append(matchingTags, tag)
		}
	}

	return matchingTags, nil
}

// Private helper methods

// getItemIDsByTag returns all item IDs for a specific tag
func (s *TagService) getItemIDsByTag(tag string) ([]string, error) {
	s.cacheLock.RLock()
	if cachedIDs, found := s.tagCache[tag]; found {
		s.cacheLock.RUnlock()
		return cachedIDs, nil
	}
	s.cacheLock.RUnlock()

	// Path to the tag file
	tagPath := filepath.Join(s.repo.BasePath(), ".meta", "tags", tag+".json")

	// Check if the file exists
	if _, err := os.Stat(tagPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Read the tag file
	data, err := os.ReadFile(tagPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tag file: %w", err)
	}

	// Parse JSON
	var itemIDs []string
	if err := json.Unmarshal(data, &itemIDs); err != nil {
		return nil, fmt.Errorf("failed to parse tag file: %w", err)
	}

	// Update cache
	s.cacheLock.Lock()
	s.tagCache[tag] = itemIDs
	s.cacheLock.Unlock()

	return itemIDs, nil
}

// addItemToTag adds an item ID to a tag's item list
func (s *TagService) addItemToTag(itemID, tag string) error {
	// Get current item IDs for this tag
	itemIDs, err := s.getItemIDsByTag(tag)
	if err != nil {
		return err
	}

	// Check if item is already in the list
	if contains(itemIDs, itemID) {
		return nil // Item already tagged
	}

	// Add item to the list
	itemIDs = append(itemIDs, itemID)

	// Save updated list
	return s.saveTagFile(tag, itemIDs)
}

// removeItemFromTag removes an item ID from a tag's item list
func (s *TagService) removeItemFromTag(itemID, tag string) error {
	// Get current item IDs for this tag
	itemIDs, err := s.getItemIDsByTag(tag)
	if err != nil {
		return err
	}

	// Skip if there are no item IDs
	if len(itemIDs) == 0 {
		return nil
	}

	// Remove item from the list
	newItemIDs := make([]string, 0, len(itemIDs))
	for _, id := range itemIDs {
		if id != itemID {
			newItemIDs = append(newItemIDs, id)
		}
	}

	// If no items left, delete the tag file
	if len(newItemIDs) == 0 {
		tagPath := filepath.Join(s.repo.BasePath(), ".meta", "tags", tag+".json")
		if err := os.Remove(tagPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete empty tag file: %w", err)
		}

		// Clear from cache too
		s.cacheLock.Lock()
		delete(s.tagCache, tag)
		s.cacheLock.Unlock()

		return nil
	}

	// Save updated list
	return s.saveTagFile(tag, newItemIDs)
}

// saveTagFile saves a list of item IDs to a tag file
func (s *TagService) saveTagFile(tag string, itemIDs []string) error {
	// Path to the tag file
	tagsDir := filepath.Join(s.repo.BasePath(), ".meta", "tags")
	tagPath := filepath.Join(tagsDir, tag+".json")

	// Ensure directory exists
	if err := os.MkdirAll(tagsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tags directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(itemIDs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal item IDs: %w", err)
	}

	// Write to file
	if err := os.WriteFile(tagPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tag file: %w", err)
	}

	// Update cache
	s.cacheLock.Lock()
	s.tagCache[tag] = itemIDs
	s.cacheLock.Unlock()

	return nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
