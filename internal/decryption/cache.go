package decryption

import (
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
)

// decryptResult holds the result of reading and processing a file.
type decryptResult struct {
	content []byte
	secrets []string
}

// Cache stores the result of ReadFileWithSecrets calls, keyed by file path.
// It is safe for concurrent use by multiple goroutines.
type Cache struct {
	mu    sync.RWMutex
	cache map[string]decryptResult
	sf    singleflight.Group
}

// NewCache returns a new, empty Cache.
func NewCache() *Cache {
	return &Cache{
		cache: make(map[string]decryptResult),
	}
}

// The returned content and secrets slices are aliases of the cached data
// and share the same underlying arrays. Callers must not append to, resize,
// or write into the returned slices; do so would corrupt the cache and
// race with other concurrent callers.
//
// Returns an error wrapping fs.ErrNotExist if the file doesn't exist.
func (c *Cache) ReadFileWithSecrets(path string) ([]byte, []string, error) {
	c.mu.RLock()

	res, cached := c.cache[path]

	c.mu.RUnlock()

	if cached {
		return res.content, res.secrets, nil
	}

	raw, err, _ := c.sf.Do(path, func() (any, error) {
		content, secrets, err := ReadFileWithSecrets(path)
		if err != nil {
			return nil, err
		}

		decrypted := decryptResult{content: content, secrets: secrets}

		c.mu.Lock()
		c.cache[path] = decrypted
		c.mu.Unlock()

		return decrypted, nil
	})
	if err != nil {
		return nil, nil, err
	}

	out, ok := raw.(decryptResult)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected type %T from singleflight for path %q", raw, path)
	}

	return out.content, out.secrets, nil
}
