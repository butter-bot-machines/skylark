package concrete

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/butter-bot-machines/skylark/pkg/assistant"
	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/process"
	procesos "github.com/butter-bot-machines/skylark/pkg/process/os"
	"github.com/butter-bot-machines/skylark/pkg/processor"
	"github.com/butter-bot-machines/skylark/pkg/provider"
	"github.com/butter-bot-machines/skylark/pkg/provider/openai"
	"github.com/butter-bot-machines/skylark/pkg/sandbox"
	"github.com/butter-bot-machines/skylark/pkg/timing"
	"github.com/butter-bot-machines/skylark/pkg/tool"
)

var logger *slog.Logger

func init() {
	logger = logging.NewLogger(&logging.Options{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
}

// processorImpl implements processor.ProcessManager
type processorImpl struct {
	config     *config.Config
	assistants *assistant.Manager
	parser     *parser.Parser
	procMgr    process.Manager
}

// NewProcessor creates a new processor
func NewProcessor(cfg *config.Config) (processor.ProcessManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Create tool manager
	toolMgr := tool.NewManager(filepath.Join(cfg.Environment.ConfigDir, "tools"))

	// Create provider
	var p provider.Provider
	var err error

	// Get OpenAI GPT-4 config
	modelConfig, ok := cfg.GetModelConfig("openai", "gpt-4")
	if !ok {
		return nil, fmt.Errorf("OpenAI GPT-4 configuration not found")
	}

	// In tests, use mock provider
	if modelConfig.APIKey == "test-key" {
		p = newMockProvider()
	} else {
		// Create OpenAI provider with default options
		p, err = openai.New("gpt-4", modelConfig, openai.Options{})
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenAI provider: %w", err)
		}
	}

	// Create network policy
	networkPolicy := &sandbox.NetworkPolicy{
		AllowOutbound: true,  // Allow tools to make outbound connections
		AllowInbound:  false, // No inbound connections needed
		AllowedHosts: []string{
			"api.openai.com", // Allow OpenAI API
		},
		AllowedPorts: []int{
			443, // HTTPS
		},
	}

	// Create assistant manager
	assistantMgr, err := assistant.NewManager(
		filepath.Join(cfg.Environment.ConfigDir, "assistants"),
		toolMgr,
		p,
		networkPolicy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create assistant manager: %w", err)
	}

	// Create process manager with system clock
	procMgr := procesos.NewManager(timing.New())

	return &processorImpl{
		config:     cfg,
		assistants: assistantMgr,
		parser:     parser.New(),
		procMgr:    procMgr,
	}, nil
}

// Process processes a single command and returns its response
func (p *processorImpl) Process(cmd *parser.Command) (string, error) {
	logger.Debug("processing command",
		"assistant", cmd.Assistant,
		"text", cmd.Text,
		"original", cmd.Original)

	// Get assistant
	assistant, err := p.assistants.Get(cmd.Assistant)
	if err != nil {
		return "", fmt.Errorf("failed to get assistant: %w", err)
	}

	// Process command
	response, err := assistant.Process(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to process command: %w", err)
	}

	return response, nil
}

// ProcessFile processes a single file
func (p *processorImpl) ProcessFile(path string) error {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse commands
	commands, err := p.parser.ParseCommands(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse commands: %w", err)
	}

	// Process all commands first
	var responses []processor.Response

	for _, cmd := range commands {
		response, err := p.Process(cmd)
		if err != nil {
			return err
		}
		if response != "" {
			responses = append(responses, processor.Response{
				Command:  cmd,
				Response: response,
			})
		}
	}

	// Update file with all responses
	if err := p.UpdateFile(path, responses); err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}

	return nil
}

// ProcessDirectory processes all markdown files in a directory
func (p *processorImpl) ProcessDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		return p.ProcessFile(path)
	})
}

// HandleResponse processes a command response
func (p *processorImpl) HandleResponse(cmd *parser.Command, response string) error {
	// For now, just validate inputs
	if cmd == nil {
		return fmt.Errorf("command is required")
	}
	if response == "" {
		return fmt.Errorf("response is required")
	}
	return nil
}

// UpdateFile updates a file with command responses
func (p *processorImpl) UpdateFile(path string, responses []processor.Response) error {
	// Read current content
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")
	var newLines []string
	commandsFound := make(map[string]bool)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line is a command that was processed
		var isCommand bool
		var response string
		for _, r := range responses {
			if trimmed == r.Command.Original {
				commandsFound[r.Command.Original] = true
				isCommand = true
				response = r.Response
				// Invalidate the command since it was processed
				line = strings.Replace(line, "!", "-!", 1)
				break
			}
		}

		if isCommand {
			// Add the invalidated command
			newLines = append(newLines, line)

			// Add blank line before response if needed
			if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
				newLines = append(newLines, "")
			}

			// Add response
			newLines = append(newLines, response)

			// Add blank line after response if next line is not blank and not a command
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if nextLine != "" && !strings.HasPrefix(nextLine, "!") {
					newLines = append(newLines, "")
				}
			}
		} else {
			newLines = append(newLines, line)
		}
	}

	// Verify all commands were found
	for _, r := range responses {
		if !commandsFound[r.Command.Original] {
			return fmt.Errorf("command not found in file: %s", r.Command.Original)
		}
	}

	// Ensure single blank line at end
	for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
		newLines = newLines[:len(newLines)-1]
	}
	newLines = append(newLines, "")

	// Only write back if content changed
	newContent := strings.Join(newLines, "\n")
	if string(content) != newContent {
		return os.WriteFile(path, []byte(newContent), 0644)
	}
	return nil
}

// GetProcessManager returns the process manager for worker pool integration
func (p *processorImpl) GetProcessManager() process.Manager {
	return p.procMgr
}
