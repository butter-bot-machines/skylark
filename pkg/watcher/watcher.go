package watcher

import (
	"time"
)

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
	watchedPaths map[string]bool
	eventQueue   chan FileEvent
	debounceTime time.Duration
}

// NewFileWatcher creates a new file watcher instance
func NewFileWatcher(debounceTime time.Duration) *FileWatcher {
	return &FileWatcher{
		watchedPaths: make(map[string]bool),
		eventQueue:   make(chan FileEvent, 1000), // Maximum queue size as per plan
		debounceTime: debounceTime,
	}
}

// Watch starts watching a directory for changes
func (w *FileWatcher) Watch(path string) error {
	// TODO: Implement fsnotify watching
	return nil
}

// Stop stops watching for changes
func (w *FileWatcher) Stop() error {
	// TODO: Implement cleanup
	return nil
}

// Events returns the event channel
func (w *FileWatcher) Events() <-chan FileEvent {
	return w.eventQueue
}
