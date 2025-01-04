package concrete

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/job"
	"github.com/butter-bot-machines/skylark/pkg/processor"
	"github.com/butter-bot-machines/skylark/pkg/watcher"
	"github.com/fsnotify/fsnotify"
)

// watcherImpl implements watcher.FileWatcher
type watcherImpl struct {
	fsWatcher *fsnotify.Watcher
	jobQueue  chan<- job.Job
	debouncer watcher.Debouncer
	processor processor.ProcessManager
	done      chan struct{}
	wg        sync.WaitGroup
	stopped   bool
	mu        sync.Mutex
}

// NewWatcher creates a new file watcher
func NewWatcher(cfg *config.Config, jobQueue chan<- job.Job, proc processor.ProcessManager) (watcher.FileWatcher, error) {
	// Validate inputs
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if jobQueue == nil {
		return nil, fmt.Errorf("job queue is required")
	}
	if proc == nil {
		return nil, fmt.Errorf("processor is required")
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	w := &watcherImpl{
		fsWatcher: fsWatcher,
		jobQueue:  jobQueue,
		processor: proc,
		debouncer: newDebouncer(cfg.FileWatch.DebounceDelay, cfg.FileWatch.MaxDelay, nil), // Use default real clock
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
func (w *watcherImpl) Stop() error {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return nil
	}
	w.stopped = true
	close(w.done)
	w.mu.Unlock()

	w.wg.Wait()
	w.debouncer.Stop()
	return w.fsWatcher.Close()
}

func (w *watcherImpl) watch() {
	defer w.wg.Done()

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			// Skip files in .skai directory and non-markdown files
			if filepath.Ext(event.Name) != ".md" || filepath.Base(filepath.Dir(event.Name)) == ".skai" {
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

func (w *watcherImpl) handleEvent(event fsnotify.Event) {
	// Create job from event using NewFileChangeJob
	j := job.NewFileChangeJob(event.Name, w.processor)

	// Send to job queue
	w.jobQueue <- j
}
