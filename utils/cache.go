// cache.go - Caching functionality for faster startup
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CacheData struct {
	Timestamp    time.Time   `json:"timestamp"`
	Repositories []Directory `json:"repositories"`
	Stats        ScanStats   `json:"stats"`
}

const cacheValidDuration = 24 * time.Hour // Cache is valid for 24 hours

func getCacheFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".gitseeker", "cache.json")
}

// SaveCache saves the scan result to the cache
func SaveCache(result *ScanResult) error {
	cacheData := CacheData{
		Timestamp:    time.Now(),
		Repositories: result.Repositories,
		Stats:        result.Stats,
	}

	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	cachePath := getCacheFilePath()
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// LoadCache loads the scan result from the cache
func LoadCache() *ScanResult {
	cachePath := getCacheFilePath()

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil
	}

	var cacheData CacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return nil
	}

	// Check if cache is still valid
	if time.Since(cacheData.Timestamp) > cacheValidDuration {
		return nil
	}

	return &ScanResult{
		Repositories: cacheData.Repositories,
		Stats:        cacheData.Stats,
	}
}

// ClearCache clears the cache file
func ClearCache() error {
	cachePath := getCacheFilePath()
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}
