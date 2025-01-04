package types

import "time"

// EventType represents the type of security event
type EventType string

const (
	// Key management events
	EventKeyAccess    EventType = "key_access"
	EventKeyCreated   EventType = "key_created"
	EventKeyRotated   EventType = "key_rotated"
	EventKeyRemoved   EventType = "key_removed"
	EventKeyExpired   EventType = "key_expired"

	// File operation events
	EventFileAccess   EventType = "file_access"
	EventFileModified EventType = "file_modified"
	EventFileCreated  EventType = "file_created"
	EventFileRemoved  EventType = "file_removed"

	// Resource events
	EventMemoryLimit  EventType = "memory_limit"
	EventCPULimit     EventType = "cpu_limit"
	EventDiskLimit    EventType = "disk_limit"

	// Security events
	EventAuthFailure    EventType = "auth_failure"
	EventAccessDenied   EventType = "access_denied"
	EventThreatDetected EventType = "threat_detected"
)

// Severity represents the severity level of a security event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Event represents a security event
type Event struct {
	ID        string                 `json:"id"`
	Timestamp time.Time             `json:"timestamp"`
	Type      EventType             `json:"type"`
	Severity  Severity              `json:"severity"`
	Source    string                `json:"source"`
	Details   string                `json:"details"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceUsage represents resource consumption
type ResourceUsage struct {
	MemoryBytes   int64   `json:"memory_bytes"`
	CPUPercent    float64 `json:"cpu_percent"`
	OpenFiles     int     `json:"open_files"`
	NetworkBytes  int64   `json:"network_bytes"`
	OpCount       int     `json:"op_count"`
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	MaxMemoryBytes  int64   `json:"max_memory_bytes"`
	MaxCPUPercent   float64 `json:"max_cpu_percent"`
	MaxOpenFiles    int     `json:"max_open_files"`
	MaxNetworkBytes int64   `json:"max_network_bytes"`
	MaxOpCount      int     `json:"max_op_count"`
}
