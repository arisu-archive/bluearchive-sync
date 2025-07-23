package global

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileCacheManager struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileCacheManager(baseDir string) *FileCacheManager {
	return &FileCacheManager{
		baseDir: baseDir,
	}
}

func (c *FileCacheManager) Get(path string) (io.ReadCloser, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fullPath := filepath.Join(c.baseDir, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cached file %s: %w", fullPath, err)
	}
	return file, nil
}

func (c *FileCacheManager) Put(path string, data io.Reader) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fullPath := filepath.Join(c.baseDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	return nil
}

func (c *FileCacheManager) Exists(path string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fullPath := filepath.Join(c.baseDir, path)
	_, err := os.Stat(fullPath)
	return err == nil
}
