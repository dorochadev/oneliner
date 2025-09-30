package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Cache struct {
	path string
	mu   sync.Mutex
	data map[string]string // key: hash, value: command (with explanation if present)
}

func New(path string) (*Cache, error) {
	c := &Cache{path: path, data: make(map[string]string)}
	if err := c.load(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Cache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	file, err := os.Open(c.path)
	if os.IsNotExist(err) {
		return nil // no cache yet
	}
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(&c.data)
}

func (c *Cache) save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	file, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(c.data)
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *Cache) Set(key, value string) error {
	c.mu.Lock()
	c.data[key] = value
	c.mu.Unlock()
	return c.save()
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
