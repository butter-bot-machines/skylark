package watcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileState represents the cached state of a file
type FileState struct {
	Hash         string
	LastModified time.Time
	LastChecked  time.Time
}

// FileStateCache manages the state of watched files
type FileStateCache struct {
	mu     sync.RWMutex
	states map[string]FileState
	maxSize int
}

func NewFileStateCache(maxSize int) *FileStateCache {
	return &FileStateCache{
		states: make(map[string]FileState),
		maxSize: maxSize,
	}
}

func (c *FileStateCache) Get(path string) (FileState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state, exists := c.states[path]
	return state, exists
}

func (c *FileStateCache) Set(path string, state FileState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If at capacity, remove oldest entry
	if len(c.states) >= c.maxSize {
		var oldest string
		oldestTime := time.Now()
		for p, s := range c.states {
			if s.LastChecked.Before(oldestTime) {
				oldest = p
				oldestTime = s.LastChecked
			}
		}
		delete(c.states, oldest)
	}

	c.states[path] = state
}

func (c *FileStateCache) Remove(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.states, path)
}

// FileEvent represents a file system event
type FileEvent struct {
	Path    string
	Type    EventType
	Content []byte
}

// EventType represents the type of file system event
type EventType int

const (
	Created EventType = iota
	Modified
	Deleted
)

// FileWatcher watches for file changes
type FileWatcher struct {
	watcher      *fsnotify.Watcher
	watchedPaths map[string]bool
	eventQueue   chan FileEvent
	debounceTime time.Duration
	cache        *FileStateCache
	workerPool   *WorkerPool
	done         chan struct{}
	mu           sync.RWMutex
}

// NewFileWatcher creates a new file watcher instance
func NewFileWatcher(debounceTime time.Duration) (*FileWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	return &FileWatcher{
		watcher:      fsWatcher,
		watchedPaths: make(map[string]bool),
		eventQueue:   make(chan FileEvent, 1000), // Maximum queue size as per plan
		debounceTime: debounceTime,
		cache:        NewFileStateCache(1000), // Maximum cache size as per plan
		workerPool:   NewWorkerPool(0, 32),   // Use defaults as per plan
		done:         make(chan struct{}),
	}, nil
}

// calculateFileHash generates a SHA-256 hash of the file content
func calculateFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Watch starts watching a directory for changes
func (w *FileWatcher) Watch(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.watchedPaths[absPath] {
		return nil // Already watching this path
	}

	if err := w.watcher.Add(absPath); err != nil {
		return fmt.Errorf("failed to add watcher: %w", err)
	}

	w.watchedPaths[absPath] = true

	// Start processing events
	go w.processEvents()

	return nil
}

// processEvents handles the fsnotify events with debouncing
func (w *FileWatcher) processEvents() {
	pending := make(map[string]time.Time)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if !w.isMarkdownFile(event.Name) {
				continue
			}

			pending[event.Name] = time.Now()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			fmt.Printf("error: %v\n", err)

		case <-ticker.C:
			now := time.Now()
			for path, lastEvent := range pending {
				if now.Sub(lastEvent) >= w.debounceTime {
					w.handleEvent(path)
					delete(pending, path)
				}
			}

		case <-w.done:
			return
		}
	}
}

// isMarkdownFile checks if the file has a .md extension
func (w *FileWatcher) isMarkdownFile(path string) bool {
	return filepath.Ext(path) == ".md"
}

// handleEvent processes a debounced file event
func (w *FileWatcher) handleEvent(path string) {
	// Check if file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		w.cache.Remove(path)
		w.eventQueue <- FileEvent{
			Path: path,
			Type: Deleted,
		}
		return
	}

	// Calculate new hash
	hash, err := calculateFileHash(path)
	if err != nil {
		fmt.Printf("error calculating hash for %s: %v\n", path, err)
		return
	}

	// Check cache
	if state, exists := w.cache.Get(path); exists {
		if state.Hash == hash {
			return // File hasn't changed
		}
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("error reading file %s: %v\n", path, err)
		return
	}

	// Update cache
	w.cache.Set(path, FileState{
		Hash:         hash,
		LastModified: info.ModTime(),
		LastChecked:  time.Now(),
	})

	// Send event
	w.eventQueue <- FileEvent{
		Path:    path,
		Type:    Modified,
		Content: content,
	}
}

// Stop stops watching for changes
func (w *FileWatcher) Stop() error {
	close(w.done)
	return w.watcher.Close()
}

// Events returns the event channel
func (w *FileWatcher) Events() <-chan FileEvent {
	return w.eventQueue
}
