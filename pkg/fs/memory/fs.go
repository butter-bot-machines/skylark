package memory

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FS implements an in-memory filesystem
type FS struct {
	mu    sync.RWMutex
	files map[string]*file
	dirs  map[string]*dir
}

// New creates a new memory filesystem
func New() *FS {
	return &FS{
		files: make(map[string]*file),
		dirs:  make(map[string]*dir),
	}
}

// Open implements fs.FS
func (f *FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check if it's a directory
	if d, ok := f.dirs[name]; ok {
		return d.clone(), nil
	}

	// Check if it's a file
	if file, ok := f.files[name]; ok {
		return file.clone(), nil
	}

	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// Stat implements fs.StatFS
func (f *FS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check if it's a directory
	if d, ok := f.dirs[name]; ok {
		return d, nil
	}

	// Check if it's a file
	if file, ok := f.files[name]; ok {
		return file, nil
	}

	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadDir implements fs.ReadDirFS
func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check if it's a file
	if _, ok := f.files[name]; ok {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: errors.New("not a directory")}
	}

	// Check if directory exists
	if _, ok := f.dirs[name]; !ok && name != "." {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}

	// Collect entries
	var entries []fs.DirEntry
	prefix := name + "/"
	if name == "." {
		prefix = ""
	}

	// Add subdirectories
	for dname, d := range f.dirs {
		if dname == name {
			continue // Skip the directory itself
		}
		if strings.HasPrefix(dname, prefix) {
			rel := strings.TrimPrefix(dname, prefix)
			if !strings.Contains(rel, "/") {
				entries = append(entries, dirEntry{d})
			}
		}
	}

	// Add files
	for fname, file := range f.files {
		if strings.HasPrefix(fname, prefix) {
			rel := strings.TrimPrefix(fname, prefix)
			if !strings.Contains(rel, "/") {
				entries = append(entries, dirEntry{file})
			}
		}
	}

	// Sort entries by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

// Glob implements fs.GlobFS
func (f *FS) Glob(pattern string) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var matches []string

	// Add matching directories
	for name := range f.dirs {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, name)
		}
	}

	// Add matching files
	for name := range f.files {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, name)
		}
	}

	sort.Strings(matches)
	return matches, nil
}

// Write implements WriteFS
func (f *FS) Write(name string, data []byte) error {
	return f.WriteFile(name, data, 0666)
}

// WriteFile implements WriteFS
func (f *FS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	if !fs.ValidPath(name) || name == "." {
		return &fs.PathError{Op: "write", Path: name, Err: fs.ErrInvalid}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if any parent is a file
	dir := filepath.Dir(name)
	if dir != "." {
		parts := strings.Split(dir, "/")
		for i := 1; i <= len(parts); i++ {
			parent := filepath.Join(parts[:i]...)
			if _, ok := f.files[parent]; ok {
				return &fs.PathError{Op: "write", Path: name, Err: errors.New("parent is a file")}
			}
		}
		if err := f.mkdirAll(dir, 0777); err != nil {
			return err
		}
	}

	// Create or update file
	f.files[name] = &file{
		name:    filepath.Base(name),
		data:    append([]byte{}, data...),
		mode:    perm & 0777,
		modTime: time.Now(),
	}

	return nil
}

// MkdirAll implements WriteFS
func (f *FS) MkdirAll(path string, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mkdirAll(path, perm)
}

// mkdirAll is the internal implementation of MkdirAll
func (f *FS) mkdirAll(path string, perm fs.FileMode) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "mkdir", Path: path, Err: fs.ErrInvalid}
	}

	// Check if any parent is a file
	parts := strings.Split(path, "/")
	for i := 1; i < len(parts); i++ {
		parent := filepath.Join(parts[:i]...)
		if _, ok := f.files[parent]; ok {
			return &fs.PathError{Op: "mkdir", Path: path, Err: errors.New("parent is a file")}
		}
	}

	// Already exists?
	if _, ok := f.dirs[path]; ok {
		return nil
	}

	// Create parent directories
	if parent := filepath.Dir(path); parent != "." {
		if err := f.mkdirAll(parent, perm); err != nil {
			return err
		}
	}

	// Create directory
	f.dirs[path] = &dir{
		name:    filepath.Base(path),
		mode:    perm&0777 | fs.ModeDir,
		modTime: time.Now(),
	}

	return nil
}

// Remove implements WriteFS
func (f *FS) Remove(name string) error {
	if !fs.ValidPath(name) {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrInvalid}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if it's a directory
	if _, ok := f.dirs[name]; ok {
		// Check if directory is empty
		prefix := name + "/"
		for p := range f.dirs {
			if strings.HasPrefix(p, prefix) {
				return &fs.PathError{Op: "remove", Path: name, Err: errors.New("directory not empty")}
			}
		}
		for p := range f.files {
			if strings.HasPrefix(p, prefix) {
				return &fs.PathError{Op: "remove", Path: name, Err: errors.New("directory not empty")}
			}
		}
		delete(f.dirs, name)
		return nil
	}

	// Check if it's a file
	if _, ok := f.files[name]; ok {
		delete(f.files, name)
		return nil
	}

	return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
}

// RemoveAll implements WriteFS
func (f *FS) RemoveAll(path string) error {
	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "removeall", Path: path, Err: fs.ErrInvalid}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Remove matching directories
	prefix := path + "/"
	for p := range f.dirs {
		if p == path || strings.HasPrefix(p, prefix) {
			delete(f.dirs, p)
		}
	}

	// Remove matching files
	for p := range f.files {
		if strings.HasPrefix(p, prefix) {
			delete(f.files, p)
		}
	}

	return nil
}

// Rename implements WriteFS
func (f *FS) Rename(oldpath, newpath string) error {
	if !fs.ValidPath(oldpath) || !fs.ValidPath(newpath) {
		return &fs.PathError{Op: "rename", Path: oldpath, Err: fs.ErrInvalid}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if source exists
	isDir := false
	if _, ok := f.dirs[oldpath]; ok {
		isDir = true
	} else if _, ok := f.files[oldpath]; !ok {
		return &fs.PathError{Op: "rename", Path: oldpath, Err: fs.ErrNotExist}
	}

	// Create parent directories of destination
	if dir := filepath.Dir(newpath); dir != "." {
		if err := f.mkdirAll(dir, 0777); err != nil {
			return err
		}
	}

	if isDir {
		// Rename directory and all contents
		oldprefix := oldpath + "/"
		newprefix := newpath + "/"

		// Move the directory itself
		f.dirs[newpath] = f.dirs[oldpath]
		f.dirs[newpath].name = filepath.Base(newpath)
		delete(f.dirs, oldpath)

		// Move subdirectories
		for p, d := range f.dirs {
			if strings.HasPrefix(p, oldprefix) {
				newp := newprefix + strings.TrimPrefix(p, oldprefix)
				f.dirs[newp] = d
				delete(f.dirs, p)
			}
		}

		// Move files
		for p, file := range f.files {
			if strings.HasPrefix(p, oldprefix) {
				newp := newprefix + strings.TrimPrefix(p, oldprefix)
				f.files[newp] = file
				delete(f.files, p)
			}
		}
	} else {
		// Rename single file
		f.files[newpath] = f.files[oldpath]
		f.files[newpath].name = filepath.Base(newpath)
		delete(f.files, oldpath)
	}

	return nil
}

// file implements fs.File and fs.FileInfo
type file struct {
	name    string
	data    []byte
	mode    fs.FileMode
	modTime time.Time
	offset  int64
}

func (f *file) clone() *file {
	return &file{
		name:    f.name,
		data:    append([]byte{}, f.data...),
		mode:    f.mode,
		modTime: f.modTime,
	}
}

func (f *file) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(b, f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *file) Write(b []byte) (int, error) {
	if f.offset > int64(len(f.data)) {
		return 0, io.ErrUnexpectedEOF
	}
	if f.offset == int64(len(f.data)) {
		f.data = append(f.data, b...)
		f.offset += int64(len(b))
		return len(b), nil
	}
	n := copy(f.data[f.offset:], b)
	f.offset += int64(n)
	return n, nil
}

func (f *file) Close() error               { return nil }
func (f *file) Stat() (fs.FileInfo, error) { return f, nil }
func (f *file) Name() string               { return f.name }
func (f *file) Size() int64                { return int64(len(f.data)) }
func (f *file) Mode() fs.FileMode          { return f.mode }
func (f *file) ModTime() time.Time         { return f.modTime }
func (f *file) IsDir() bool                { return false }
func (f *file) Sys() interface{}           { return nil }

func (f *file) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = f.offset + offset
	case io.SeekEnd:
		abs = int64(len(f.data)) + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("negative position")
	}
	f.offset = abs
	return abs, nil
}

// dir implements fs.File and fs.FileInfo
type dir struct {
	name    string
	mode    fs.FileMode
	modTime time.Time
}

func (d *dir) clone() *dir {
	return &dir{
		name:    d.name,
		mode:    d.mode,
		modTime: d.modTime,
	}
}

func (d *dir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: errors.New("is a directory")}
}

func (d *dir) Close() error               { return nil }
func (d *dir) Stat() (fs.FileInfo, error) { return d, nil }
func (d *dir) Name() string               { return d.name }
func (d *dir) Size() int64                { return 0 }
func (d *dir) Mode() fs.FileMode          { return d.mode }
func (d *dir) ModTime() time.Time         { return d.modTime }
func (d *dir) IsDir() bool                { return true }
func (d *dir) Sys() interface{}           { return nil }

// dirEntry implements fs.DirEntry
type dirEntry struct {
	info fs.FileInfo
}

func (de dirEntry) Name() string               { return de.info.Name() }
func (de dirEntry) IsDir() bool                { return de.info.IsDir() }
func (de dirEntry) Type() fs.FileMode          { return de.info.Mode().Type() }
func (de dirEntry) Info() (fs.FileInfo, error) { return de.info, nil }
