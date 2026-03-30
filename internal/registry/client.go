package registry

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Client is an HTTP client with rate limiting and disk caching.
type Client struct {
	http      *http.Client
	cacheDir  string
	cacheTTL  time.Duration
	semaphore chan struct{}
	mu        sync.Mutex
}

// NewClient creates a new registry client.
func NewClient(concurrency int, cacheTTL time.Duration) *Client {
	cacheDir := defaultCacheDir()
	os.MkdirAll(cacheDir, 0o755)

	return &Client{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		cacheDir:  cacheDir,
		cacheTTL:  cacheTTL,
		semaphore: make(chan struct{}, concurrency),
	}
}

// GetJSON fetches a URL and unmarshals the JSON response into dest.
// Results are cached to disk.
func (c *Client) GetJSON(url string, dest any) error {
	// Check cache first
	if data, ok := c.fromCache(url); ok {
		return json.Unmarshal(data, dest)
	}

	// Acquire semaphore for rate limiting
	c.semaphore <- struct{}{}
	defer func() { <-c.semaphore }()

	resp, err := c.httpGetWithRetry(url, 3)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response from %s: %w", url, err)
	}

	// Cache the response
	c.toCache(url, data)

	return json.Unmarshal(data, dest)
}

// HeadContentLength returns the Content-Length header value for a URL.
func (c *Client) HeadContentLength(url string) (int64, error) {
	c.semaphore <- struct{}{}
	defer func() { <-c.semaphore }()

	resp, err := c.http.Head(url)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.ContentLength, nil
}

func (c *Client) httpGetWithRetry(url string, maxRetries int) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := c.http.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			continue
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) cacheKey(url string) string {
	h := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", h[:16])
}

func (c *Client) cachePath(url string) string {
	return filepath.Join(c.cacheDir, c.cacheKey(url)+".json")
}

func (c *Client) fromCache(url string) ([]byte, bool) {
	path := c.cachePath(url)
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > c.cacheTTL {
		os.Remove(path)
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func (c *Client) toCache(url string, data []byte) {
	path := c.cachePath(url)
	os.WriteFile(path, data, 0o644)
}

func defaultCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "packman")
}
