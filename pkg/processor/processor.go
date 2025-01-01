package processor

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
	"github.com/butter-bot-machines/skylark/pkg/provider"
	"github.com/butter-bot-machines/skylark/pkg/provider/openai"
	"github.com/butter-bot-machines/skylark/pkg/sandbox"
	"github.com/butter-bot-machines/skylark/pkg/tool"
)

var logger *slog.Logger
func init() {
	logger = logging.NewLogger(&logging.Options{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
}

// commandResponse pairs a command with its response
type commandResponse struct {
	cmd      *parser.Command
	response string
}

// Processor handles the core command processing pipeline
type Processor struct {
	config     *config.Config
	assistants *assistant.Manager
	parser     *parser.Parser
}

// New creates a new processor
func New(cfg *config.Config) (*Processor, error) {
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

	// Create OpenAI provider
	p, err = openai.New("gpt-4", modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI provider: %w", err)
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

	return &Processor{
		config:     cfg,
		assistants: assistantMgr,
		parser:     parser.New(),
	}, nil
}

// ProcessFile processes a single file
func (p *Processor) ProcessFile(path string) error {
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
	var responses []commandResponse

	for _, cmd := range commands {
		logger.Debug("processing command",
			"assistant", cmd.Assistant,
			"text", cmd.Text,
			"original", cmd.Original)
		
		// Get assistant
		assistant, err := p.assistants.Get(cmd.Assistant)
		if err != nil {
			return fmt.Errorf("failed to get assistant: %w", err)
		}

		// Process command
		response, err := assistant.Process(cmd)
		if err != nil {
			return fmt.Errorf("failed to process command: %w", err)
		}

		responses = append(responses, commandResponse{cmd, response})
	}

	// Update file with all responses
	if err := p.updateFile(path, responses); err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}

	return nil
}

// ProcessDirectory processes all markdown files in a directory
func (p *Processor) ProcessDirectory(dir string) error {
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

// updateFile updates the file with all command responses
func (p *Processor) updateFile(path string, responses []commandResponse) error {
	// Read current content
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Split content into lines and remove old responses
	lines := strings.Split(string(content), "\n")
	var contentLines []string
	skipNextBlank := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip response lines and their following blank line
		if strings.HasPrefix(trimmed, "> ") {
			skipNextBlank = true
			continue
		}
		if skipNextBlank && trimmed == "" {
			skipNextBlank = false
			continue
		}
		
		contentLines = append(contentLines, line)
	}

	// Build new content with responses
	var newLines []string
	commandsFound := make(map[string]bool)

	for i, line := range contentLines {
		trimmed := strings.TrimSpace(line)
		newLines = append(newLines, line)

		// Check if this line is a command
		for _, cr := range responses {
			if trimmed == cr.cmd.Original {
				commandsFound[cr.cmd.Original] = true
				
				// Add blank line before response
				if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
					newLines = append(newLines, "")
				}
				
				// Add response
				newLines = append(newLines, "> "+cr.response)
				
				// Add blank line after response if next line is not blank and not a command
				if i+1 < len(contentLines) {
					nextLine := strings.TrimSpace(contentLines[i+1])
					if nextLine != "" && !strings.HasPrefix(nextLine, "!") {
						newLines = append(newLines, "")
					}
				}
				break
			}
		}
	}

	// Verify all commands were found
	for _, cr := range responses {
		if !commandsFound[cr.cmd.Original] {
			return fmt.Errorf("command not found in file: %s", cr.cmd.Original)
		}
	}

	// Ensure single blank line at end
	for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
		newLines = newLines[:len(newLines)-1]
	}
	newLines = append(newLines, "")

	// Write back
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(path, []byte(newContent), 0644)
}
