package types

// FilePermissionsConfig defines file permission settings
type FilePermissionsConfig struct {
	Default       int      `yaml:"default"`
	Private       int      `yaml:"private"`
	Public        int      `yaml:"public"`
	AllowedPaths  []string `yaml:"allowed_paths"`
	BlockedPaths  []string `yaml:"blocked_paths"`
	AllowSymlinks bool     `yaml:"allow_symlinks"`
	MaxFileSize   int64    `yaml:"max_file_size"`
}

// AuditLogConfig defines audit logging settings
type AuditLogConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Path          string   `yaml:"path"`
	MaxSize       int64    `yaml:"max_size"`
	Compress      bool     `yaml:"compress"`
	RetentionDays int      `yaml:"retention_days"`
	Events        []string `yaml:"events"`
}

// SecurityConfig defines security settings
type SecurityConfig struct {
	AllowedPaths    []string             `yaml:"allowed_paths"`
	MaxFileSize     int64                `yaml:"max_file_size"`
	FilePermissions FilePermissionsConfig `yaml:"file_permissions"`
	EncryptionKey   string               `yaml:"encryption_key"`
	KeyStoragePath  string               `yaml:"key_storage_path"`
	AuditLog        AuditLogConfig       `yaml:"audit_log"`
}
