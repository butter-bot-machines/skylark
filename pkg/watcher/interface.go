package watcher

import (
	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/job"
	"github.com/butter-bot-machines/skylark/pkg/processor"
)

// EventHandler handles file system events
type EventHandler interface {
	// HandleEvent processes a file system event
	HandleEvent(path string) error
}

// Debouncer coalesces rapid events
type Debouncer interface {
	// Debounce delays execution of fn until events settle
	Debounce(key string, fn func())
	// Stop stops the debouncer
	Stop()
}

// PathManager manages watched paths
type PathManager interface {
	// AddPath adds a path to watch
	AddPath(path string) error
	// RemovePath removes a path from watching
	RemovePath(path string) error
	// IsWatched checks if a path is being watched
	IsWatched(path string) bool
}

// FileWatcher monitors files for changes
type FileWatcher interface {
	// Stop stops the watcher
	Stop() error
}

// Factory creates new watchers
type Factory interface {
	// NewWatcher creates a new watcher with the given options
	NewWatcher(cfg *config.Config, jobQueue chan<- job.Job, proc processor.ProcessManager) (FileWatcher, error)
}
