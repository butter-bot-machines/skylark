package watcher

import (
	"testing"
	"time"
)

func TestNewFileWatcher(t *testing.T) {
	debounceTime := 500 * time.Millisecond
	w := NewFileWatcher(debounceTime)

	if w == nil {
		t.Fatal("Expected non-nil FileWatcher")
	}

	if w.debounceTime != debounceTime {
		t.Errorf("Expected debounceTime %v, got %v", debounceTime, w.debounceTime)
	}

	if len(w.watchedPaths) != 0 {
		t.Errorf("Expected empty watchedPaths, got %d entries", len(w.watchedPaths))
	}

	if cap(w.eventQueue) != 1000 {
		t.Errorf("Expected eventQueue capacity 1000, got %d", cap(w.eventQueue))
	}
}

func TestFileWatcherWatch(t *testing.T) {
	w := NewFileWatcher(500 * time.Millisecond)
	err := w.Watch("testdata")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// TestFileEvent ensures FileEvent struct works as expected
func TestFileEvent(t *testing.T) {
	event := FileEvent{
		Path:    "test.md",
		Type:    Modified,
		Content: []byte("test content"),
	}

	if event.Path != "test.md" {
		t.Errorf("Expected path 'test.md', got %s", event.Path)
	}

	if event.Type != Modified {
		t.Errorf("Expected type Modified, got %v", event.Type)
	}

	if string(event.Content) != "test content" {
		t.Errorf("Expected content 'test content', got %s", string(event.Content))
	}
}
