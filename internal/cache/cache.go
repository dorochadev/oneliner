package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Cache struct {
	path string
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type cacheEntry struct {
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
}

func New(path string) (*Cache, error) {
	c := &Cache{
		path: path,
		data: make(map[string]cacheEntry),
	}
	if err := c.load(); err != nil {
		return nil, fmt.Errorf("loading cache: %w", err)
	}
	return c, nil
}

func (c *Cache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.Open(c.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("opening cache file: %w", err)
	}
	defer file.Close()

	// Try to decode as new format first
	var newData map[string]cacheEntry
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&newData); err != nil {
		// If that fails, try legacy format
		file.Seek(0, 0) // Reset file pointer
		var legacyData map[string]string
		if err := json.NewDecoder(file).Decode(&legacyData); err != nil {
			return fmt.Errorf("decoding cache (tried both new and legacy format): %w", err)
		}

		// Migrate legacy format to new format
		c.data = make(map[string]cacheEntry, len(legacyData))
		for k, v := range legacyData {
			c.data[k] = cacheEntry{
				Command:   v,
				Timestamp: time.Now(), // Use current time for legacy entries
			}
		}

		// Save in new format
		return c.saveNoLock()
	}

	c.data = newData
	return nil
}

func (c *Cache) save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.saveNoLock()
}

func (c *Cache) saveNoLock() error {

	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding cache: %w", err)
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	tempPath := c.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tempPath, c.path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.data[key]
	return entry.Command, ok
}

func (c *Cache) Set(key, value string) error {
	c.mu.Lock()
	c.data[key] = cacheEntry{
		Command:   value,
		Timestamp: time.Now(),
	}
	dataCopy := make(map[string]cacheEntry, len(c.data))
	for k, v := range c.data {
		dataCopy[k] = v
	}
	c.mu.Unlock()

	data, err := json.MarshalIndent(dataCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding cache: %w", err)
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	tempPath := c.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tempPath, c.path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}

func HashQuery(query, osys, cwd, username, shell string, explain bool) string {
	h := sha256.New()
	h.Write([]byte(query))
	h.Write([]byte(osys))
	h.Write([]byte(cwd))
	h.Write([]byte(username))
	h.Write([]byte(shell))
	if explain {
		h.Write([]byte("explain"))
	}
	return hex.EncodeToString(h.Sum(nil))
}
