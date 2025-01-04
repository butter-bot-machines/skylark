package memory

import (
	"io"
	"path/filepath"
	"sync"
	"testing"
)

func TestFS_BasicOperations(t *testing.T) {
	fsys := New()

	// Test Write and Read
	t.Run("Write and Read", func(t *testing.T) {
		data := []byte("hello world")
		if err := fsys.Write("test.txt", data); err != nil {
			t.Errorf("Write failed: %v", err)
		}

		f, err := fsys.Open("test.txt")
		if err != nil {
			t.Errorf("Open failed: %v", err)
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}

		if string(content) != string(data) {
			t.Errorf("Got %q, want %q", content, data)
		}
	})

	// Test Stat
	t.Run("Stat", func(t *testing.T) {
		info, err := fsys.Stat("test.txt")
		if err != nil {
			t.Errorf("Stat failed: %v", err)
		}

		if info.Name() != "test.txt" {
			t.Errorf("Got name %q, want %q", info.Name(), "test.txt")
		}
		if info.Size() != 11 {
			t.Errorf("Got size %d, want 11", info.Size())
		}
		if info.IsDir() {
			t.Error("File reported as directory")
		}
	})

	// Test Remove
	t.Run("Remove", func(t *testing.T) {
		if err := fsys.Remove("test.txt"); err != nil {
			t.Errorf("Remove failed: %v", err)
		}

		if _, err := fsys.Stat("test.txt"); err == nil {
			t.Error("File still exists after removal")
		}
	})
}

func TestFS_DirectoryOperations(t *testing.T) {
	fsys := New()

	// Test MkdirAll
	t.Run("MkdirAll", func(t *testing.T) {
		if err := fsys.MkdirAll("a/b/c", 0755); err != nil {
			t.Errorf("MkdirAll failed: %v", err)
		}

		info, err := fsys.Stat("a/b/c")
		if err != nil {
			t.Errorf("Stat failed: %v", err)
		}
		if !info.IsDir() {
			t.Error("Directory not created")
		}
	})

	// Test ReadDir
	t.Run("ReadDir", func(t *testing.T) {
		// Create some files and directories
		if err := fsys.Write("a/b/c/file1.txt", []byte("content")); err != nil {
			t.Errorf("Write failed: %v", err)
		}
		if err := fsys.Write("a/b/c/file2.txt", []byte("content")); err != nil {
			t.Errorf("Write failed: %v", err)
		}
		if err := fsys.MkdirAll("a/b/c/subdir", 0755); err != nil {
			t.Errorf("MkdirAll failed: %v", err)
		}

		entries, err := fsys.ReadDir("a/b/c")
		if err != nil {
			t.Errorf("ReadDir failed: %v", err)
		}

		if len(entries) != 3 {
			t.Errorf("Got %d entries, want 3", len(entries))
		}

		// Entries should be sorted by name
		names := []string{entries[0].Name(), entries[1].Name(), entries[2].Name()}
		want := []string{"file1.txt", "file2.txt", "subdir"}
		for i := range names {
			if names[i] != want[i] {
				t.Errorf("Entry %d: got %q, want %q", i, names[i], want[i])
			}
		}
	})

	// Test RemoveAll
	t.Run("RemoveAll", func(t *testing.T) {
		if err := fsys.RemoveAll("a"); err != nil {
			t.Errorf("RemoveAll failed: %v", err)
		}

		if _, err := fsys.Stat("a"); err == nil {
			t.Error("Directory still exists after removal")
		}
	})
}

func TestFS_PathOperations(t *testing.T) {
	fsys := New()

	// Test path cleaning
	t.Run("Path Cleaning", func(t *testing.T) {
		paths := []string{
			"./test.txt",
			"a/../test.txt",
			"//test.txt",
		}

		for _, p := range paths {
			if err := fsys.Write(p, []byte("content")); err == nil {
				t.Errorf("Write should fail for path %q", p)
			}
		}
	})

	// Test path traversal
	t.Run("Path Traversal", func(t *testing.T) {
		paths := []string{
			"../test.txt",
			"/test.txt",
			"a/../../test.txt",
		}

		for _, p := range paths {
			if err := fsys.Write(p, []byte("content")); err == nil {
				t.Errorf("Write should fail for path %q", p)
			}
		}
	})

	// Test rename
	t.Run("Rename", func(t *testing.T) {
		// Create a file
		if err := fsys.Write("old.txt", []byte("content")); err != nil {
			t.Errorf("Write failed: %v", err)
		}

		// Rename it
		if err := fsys.Rename("old.txt", "new.txt"); err != nil {
			t.Errorf("Rename failed: %v", err)
		}

		// Check old path is gone
		if _, err := fsys.Stat("old.txt"); err == nil {
			t.Error("Old file still exists")
		}

		// Check new path exists
		if _, err := fsys.Stat("new.txt"); err != nil {
			t.Error("New file doesn't exist")
		}
	})
}

func TestFS_Concurrency(t *testing.T) {
	fsys := New()
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	// Concurrent writes
	t.Run("Concurrent Writes", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					name := filepath.Join("dir", filepath.FromSlash(filepath.Join("subdir", "file.txt")))
					if err := fsys.Write(name, []byte("content")); err != nil {
						t.Errorf("Write failed: %v", err)
					}
				}
			}(i)
		}
		wg.Wait()
	})

	// Concurrent reads
	t.Run("Concurrent Reads", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					name := filepath.Join("dir", filepath.FromSlash(filepath.Join("subdir", "file.txt")))
					if _, err := fsys.Stat(name); err != nil {
						t.Errorf("Stat failed: %v", err)
					}
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestFS_ErrorCases(t *testing.T) {
	fsys := New()

	// Test non-existent file
	t.Run("Non-existent File", func(t *testing.T) {
		if _, err := fsys.Open("nonexistent.txt"); err == nil {
			t.Error("Open should fail for non-existent file")
		}
	})

	// Test remove non-empty directory
	t.Run("Remove Non-empty Directory", func(t *testing.T) {
		if err := fsys.MkdirAll("dir/subdir", 0755); err != nil {
			t.Errorf("MkdirAll failed: %v", err)
		}
		if err := fsys.Write("dir/file.txt", []byte("content")); err != nil {
			t.Errorf("Write failed: %v", err)
		}

		if err := fsys.Remove("dir"); err == nil {
			t.Error("Remove should fail for non-empty directory")
		}
	})

	// Test invalid paths
	t.Run("Invalid Paths", func(t *testing.T) {
		paths := []string{
			"",
			".",
			"..",
			"/",
			"a/",
			"a//b",
		}

		for _, p := range paths {
			if err := fsys.Write(p, []byte("content")); err == nil {
				t.Errorf("Write should fail for path %q", p)
			}
		}
	})

	// Test file as directory
	t.Run("File as Directory", func(t *testing.T) {
		if err := fsys.Write("file.txt", []byte("content")); err != nil {
			t.Errorf("Write failed: %v", err)
		}

		if err := fsys.MkdirAll("file.txt/subdir", 0755); err == nil {
			t.Error("MkdirAll should fail when parent is a file")
		}

		if _, err := fsys.ReadDir("file.txt"); err == nil {
			t.Error("ReadDir should fail for a file")
		}
	})
}
