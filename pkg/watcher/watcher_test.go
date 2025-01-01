package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewFileWatcher(t *testing.T) {
	debounceTime := 500 * time.Millisecond
	w, err := NewFileWatcher(debounceTime)
	if err != nil {
		t.Fatalf("Failed to create FileWatcher: %v", err)
	}

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

	if w.cache == nil {
		t.Error("Expected non-nil cache")
	}
}

func TestFileWatcherWatch(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a test markdown file
	testFile := filepath.Join(tempDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Test Content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	w, err := NewFileWatcher(500 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create FileWatcher: %v", err)
	}
	defer w.Stop()

	// Watch the temporary directory
	err = w.Watch(tempDir)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test watching same path twice
	err = w.Watch(tempDir)
	if err != nil {
		t.Errorf("Expected no error when watching same path twice, got %v", err)
	}

	// Verify that the watcher detects file changes
	done := make(chan bool)
	go func() {
		select {
		case event := <-w.Events():
			if event.Path != testFile {
				t.Errorf("Expected event for %s, got %s", testFile, event.Path)
			}
			done <- true
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for file event")
			done <- false
		}
	}()

	// Modify the test file
	time.Sleep(100 * time.Millisecond) // Wait a bit before modifying
	err = os.WriteFile(testFile, []byte("# Modified Content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	<-done
}

func TestFileStateCache(t *testing.T) {
	cache := NewFileStateCache(2) // Small size for testing
	
	state1 := FileState{
		Hash:         "hash1",
		LastModified: time.Now(),
		LastChecked:  time.Now(),
	}
	
	state2 := FileState{
		Hash:         "hash2",
		LastModified: time.Now(),
		LastChecked:  time.Now(),
	}
	
	// Test setting and getting states
	cache.Set("file1.md", state1)
	if got, exists := cache.Get("file1.md"); !exists || got.Hash != state1.Hash {
		t.Error("Failed to get cached state for file1.md")
	}
	
	// Test cache eviction
	cache.Set("file2.md", state2)
	cache.Set("file3.md", FileState{Hash: "hash3"}) // Should evict oldest entry
	
	if _, exists := cache.Get("file1.md"); exists {
		t.Error("Expected file1.md to be evicted from cache")
	}
	
	// Test removal
	cache.Remove("file2.md")
	if _, exists := cache.Get("file2.md"); exists {
		t.Error("Expected file2.md to be removed from cache")
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
