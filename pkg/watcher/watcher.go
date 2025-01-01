package watcher

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/job"
	"github.com/fsnotify/fsnotify"
)

// Watcher monitors files for changes
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	jobQueue  chan<- job.Job
	debouncer *Debouncer
	done      chan struct{}
	wg        sync.WaitGroup
}

// New creates a new file watcher
func New(cfg *config.Config, jobQueue chan<- job.Job) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	w := &Watcher{
		fsWatcher: fsWatcher,
		jobQueue:  jobQueue,
		debouncer: newDebouncer(cfg.FileWatch.DebounceDelay, cfg.FileWatch.MaxDelay),
		done:      make(chan struct{}),
	}

	// Add watch paths
	for _, path := range cfg.WatchPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", path, err)
		}
		if err := fsWatcher.Add(absPath); err != nil {
			return nil, fmt.Errorf("failed to watch path %s: %w", absPath, err)
		}
		slog.Info("Watching path", "path", absPath)
	}

	w.wg.Add(1)
	go w.watch()

	return w, nil
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	close(w.done)
	w.wg.Wait()
	return w.fsWatcher.Close()
}

func (w *Watcher) watch() {
	defer w.wg.Done()

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			// Only process .md files
			if filepath.Ext(event.Name) != ".md" {
				continue
			}
			// Debounce events
			w.debouncer.Debounce(event.Name, func() {
				w.handleEvent(event)
			})
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher error", "error", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Create job from event
	job := &FileJob{
		Path: event.Name,
		Type: event.Op.String(),
	}

	// Send to job queue
	w.jobQueue <- job
}

// FileJob represents a file processing job
type FileJob struct {
	Path string
	Type string
}

// Process processes the file job
func (j *FileJob) Process() error {
	// TODO: Implement file processing
	return nil
}

// OnFailure handles job failure
func (j *FileJob) OnFailure(err error) {
	slog.Error("File job failed",
		"path", j.Path,
		"type", j.Type,
		"error", err,
	)
}

// MaxRetries returns the maximum number of retries
func (j *FileJob) MaxRetries() int {
	return 3
}
