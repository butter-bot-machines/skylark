package processor

import (
	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/process"
)

// CommandProcessor handles individual command processing
type CommandProcessor interface {
	// Process processes a single command and returns its response
	Process(cmd *parser.Command) (string, error)
}

// FileProcessor handles file-level processing
type FileProcessor interface {
	// ProcessFile processes a single file
	ProcessFile(path string) error

	// ProcessDirectory processes all markdown files in a directory
	ProcessDirectory(dir string) error
}

// ResponseHandler manages command responses
type ResponseHandler interface {
	// HandleResponse processes a command response
	HandleResponse(cmd *parser.Command, response string) error

	// UpdateFile updates a file with command responses
	UpdateFile(path string, responses []Response) error
}

// Response represents a command and its response
type Response struct {
	Command  *parser.Command
	Response string
}

// ProcessManager handles the core command processing pipeline
type ProcessManager interface {
	FileProcessor
	CommandProcessor
	ResponseHandler

	// GetProcessManager returns the process manager for worker pool integration
	GetProcessManager() process.Manager
}

// Factory creates new processors
type Factory interface {
	// NewProcessor creates a new processor with the given configuration
	NewProcessor(cfg Config) (ProcessManager, error)
}

// Config defines processor configuration
type Config struct {
	AssistantsDir string // Directory containing assistant configurations
	ToolsDir      string // Directory containing tool configurations
	NetworkPolicy NetworkPolicy
}

// NetworkPolicy defines network access rules
type NetworkPolicy struct {
	AllowOutbound bool     // Allow outbound connections
	AllowInbound  bool     // Allow inbound connections
	AllowedHosts  []string // Allowed hostnames
	AllowedPorts  []int    // Allowed ports
}
