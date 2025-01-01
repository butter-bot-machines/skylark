package config

// SecurityConfig contains security-related settings
type SecurityConfig struct {
	// EncryptionKey is a base64 encoded 32-byte key used for AES-256 encryption
	EncryptionKey string `yaml:"encryption_key"`

	// KeyStoragePath is the path where encrypted API keys are stored
	KeyStoragePath string `yaml:"key_storage_path"`

	// FilePermissions defines allowed file operations
	FilePermissions FilePermissionsConfig `yaml:"file_permissions"`

	// ResourceLimits defines resource usage limits
	ResourceLimits ResourceLimitsConfig `yaml:"resource_limits"`

	// AuditLog configures security event logging
	AuditLog AuditLogConfig `yaml:"audit_log"`
}

// FilePermissionsConfig defines file access controls
type FilePermissionsConfig struct {
	// AllowedPaths lists directories that can be accessed
	AllowedPaths []string `yaml:"allowed_paths"`

	// BlockedPaths lists directories that cannot be accessed
	BlockedPaths []string `yaml:"blocked_paths"`

	// AllowSymlinks determines if symbolic links can be followed
	AllowSymlinks bool `yaml:"allow_symlinks"`

	// MaxFileSize is the maximum allowed file size in bytes
	MaxFileSize int64 `yaml:"max_file_size"`
}

// ResourceLimitsConfig defines resource usage limits
type ResourceLimitsConfig struct {
	// MaxMemoryMB is the maximum memory usage per process in megabytes
	MaxMemoryMB int `yaml:"max_memory_mb"`

	// MaxCPUSeconds is the maximum CPU time per process in seconds
	MaxCPUSeconds int `yaml:"max_cpu_seconds"`

	// MaxFileDescriptors is the maximum number of open files
	MaxFileDescriptors int `yaml:"max_file_descriptors"`

	// MaxProcesses is the maximum number of child processes
	MaxProcesses int `yaml:"max_processes"`
}

// AuditLogConfig defines security event logging settings
type AuditLogConfig struct {
	// Enabled determines if security audit logging is active
	Enabled bool `yaml:"enabled"`

	// Path is where audit log files are stored
	Path string `yaml:"path"`

	// RetentionDays is how long to keep audit logs
	RetentionDays int `yaml:"retention_days"`

	// Events lists which types of events to log
	Events []string `yaml:"events"`
}
