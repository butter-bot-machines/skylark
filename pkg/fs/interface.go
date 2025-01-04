package fs

import (
	"io"
	"io/fs"
)

// ReadFS extends io/fs.FS with additional read operations
type ReadFS interface {
	fs.FS
	fs.StatFS
	fs.ReadDirFS
	fs.GlobFS
}

// WriteFS defines write operations not covered by io/fs
type WriteFS interface {
	// Write writes data to a file, creating it if necessary
	Write(name string, data []byte) error

	// WriteFile creates or truncates a file and writes data
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// MkdirAll creates a directory and all parent directories
	MkdirAll(path string, perm fs.FileMode) error

	// Remove removes a file or empty directory
	Remove(name string) error

	// RemoveAll removes a file or directory and any children
	RemoveAll(path string) error

	// Rename renames (moves) a file or directory
	Rename(oldpath, newpath string) error
}

// FS combines read and write operations
type FS interface {
	ReadFS
	WriteFS
}

// File represents an open file
type File interface {
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
	fs.File
}

// Error types for filesystem operations
var (
	ErrNotExist    = fs.ErrNotExist
	ErrPermission  = fs.ErrPermission
	ErrInvalidPath = fs.ErrInvalid
)
