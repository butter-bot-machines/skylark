package parser

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/butter-bot-machines/skylark/pkg/logging"
)

var logger *slog.Logger

func init() {
	logger = logging.NewLogger(&logging.Options{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
}

const (
	maxCommandSize = 4000 // Maximum size for a single command
	maxTotalSize   = 8000 // Maximum total size for all context
)

// BlockType represents different markdown block types
type BlockType int

const (
	Header BlockType = iota
	List
	Paragraph
	Quote
	Table
	Code
)

// Block represents a markdown content block
type Block struct {
	Type    BlockType
	Level   int    // For headers
	Content string // Block content
}

// Command represents a parsed command
type Command struct {
	Assistant  string           // Assistant name (default if not specified)
	Text       string           // Command text
	Original   string           // Original command line
	References []string         // Referenced sections
	Context    map[string]Block // Section content by reference
}

// Parser handles command parsing
type Parser struct {
	commandPattern *regexp.Regexp
	refPattern     *regexp.Regexp
	warnings       []string // Accumulated warnings
}

// New creates a new parser
func New() *Parser {
	return &Parser{
		commandPattern: regexp.MustCompile(`^!(?:\s*(\S+)\s+)?(.+)$`), // Allow whitespace after !
		refPattern:     regexp.MustCompile(`#\s*([^#\n]+?)(?:\s*#|$)`),
		warnings:       make([]string, 0),
	}
}

// ClearWarnings resets the warning list
func (p *Parser) ClearWarnings() {
	p.warnings = p.warnings[:0]
}

// GetWarnings returns accumulated warnings
func (p *Parser) GetWarnings() []string {
	return p.warnings
}

// addWarning adds a warning message
func (p *Parser) addWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	p.warnings = append(p.warnings, msg)
	logger.Warn(msg)
}

// ParseCommands parses all commands from content
func (p *Parser) ParseCommands(content string) ([]*Command, error) {
	var commands []*Command
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "!") {
			cmd, err := p.ParseCommand(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse command: %w", err)
			}
			commands = append(commands, cmd)
		}
	}

	return commands, nil
}

// ParseCommand parses a single command line
func (p *Parser) ParseCommand(line string) (*Command, error) {
	trimmed := strings.TrimSpace(line)

	// Check command size
	if len(trimmed) > maxCommandSize {
		return nil, fmt.Errorf("command exceeds maximum size of %d characters", maxCommandSize)
	}

	matches := p.commandPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return nil, fmt.Errorf("invalid command format: %s", line)
	}

	// Extract assistant name and text
	var assistant, text string
	if matches[1] == "" {
		// No assistant specified
		assistant = "default"
		text = matches[2]
		logger.Debug("parsed command without assistant prefix",
			"text", text)
	} else {
		// First word is assistant name
		assistant = strings.ToLower(matches[1]) // Simple lowercase normalization
		text = matches[2]
		logger.Debug("parsed command with assistant",
			"assistant", assistant,
			"text", text)
	}

	original := strings.TrimSpace(line)
	references := p.ParseReferences(text)

	cmd := &Command{
		Assistant:  assistant,
		Text:       text,
		Original:   original,
		References: references,
		Context:    make(map[string]Block),
	}

	logger.Debug("created command",
		"assistant", cmd.Assistant,
		"text", cmd.Text,
		"original", cmd.Original,
		"references", cmd.References)

	return cmd, nil
}

// ParseReferences extracts section references from text
func (p *Parser) ParseReferences(text string) []string {
	var refs []string
	matches := p.refPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		refs = append(refs, strings.TrimSpace(match[1]))
	}
	return refs
}

// ParseBlocks parses markdown content into blocks
func (p *Parser) ParseBlocks(content string) []Block {
	var blocks []Block
	lines := strings.Split(content, "\n")
	
	var currentBlock *Block
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Handle code blocks
		if strings.HasPrefix(trimmed, "```") {
			if currentBlock != nil && currentBlock.Type == Code {
				// End code block if delimiter matches
				if strings.TrimSpace(trimmed) == "```" {
					blocks = append(blocks, *currentBlock)
					currentBlock = nil
				}
			} else {
				// Start code block
				currentBlock = &Block{Type: Code}
			}
			continue
		}

		// Inside code block
		if currentBlock != nil && currentBlock.Type == Code {
			if currentBlock.Content == "" {
				currentBlock.Content = line
			} else {
				currentBlock.Content += "\n" + line
			}
			continue
		}

		// Handle other block types
		switch {
		case strings.HasPrefix(trimmed, "#"):
			if currentBlock != nil {
				blocks = append(blocks, *currentBlock)
			}
			level := strings.Count(trimmed, "#")
			currentBlock = &Block{
				Type:    Header,
				Level:   level,
				Content: strings.TrimSpace(trimmed[level:]),
			}
		
		case strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*"):
			if currentBlock != nil && currentBlock.Type != List {
				blocks = append(blocks, *currentBlock)
				currentBlock = nil
			}
			if currentBlock == nil {
				currentBlock = &Block{Type: List}
			}
			if currentBlock.Content == "" {
				currentBlock.Content = trimmed
			} else {
				currentBlock.Content += "\n" + trimmed
			}
		
		case strings.HasPrefix(trimmed, ">"):
			if currentBlock != nil && currentBlock.Type != Quote {
				blocks = append(blocks, *currentBlock)
				currentBlock = nil
			}
			if currentBlock == nil {
				currentBlock = &Block{Type: Quote}
			}
			content := strings.TrimPrefix(trimmed, ">")
			if currentBlock.Content == "" {
				currentBlock.Content = strings.TrimSpace(content)
			} else {
				currentBlock.Content += "\n" + strings.TrimSpace(content)
			}
		
		case strings.HasPrefix(trimmed, "|"):
			if currentBlock != nil && currentBlock.Type != Table {
				blocks = append(blocks, *currentBlock)
				currentBlock = nil
			}
			if currentBlock == nil {
				currentBlock = &Block{Type: Table}
			}
			if currentBlock.Content == "" {
				currentBlock.Content = trimmed
			} else {
				currentBlock.Content += "\n" + trimmed
			}
		
		case trimmed == "":
			if currentBlock != nil {
				blocks = append(blocks, *currentBlock)
				currentBlock = nil
			}
		
		default:
			if currentBlock != nil && currentBlock.Type != Paragraph {
				blocks = append(blocks, *currentBlock)
				currentBlock = nil
			}
			if currentBlock == nil {
				currentBlock = &Block{Type: Paragraph}
			}
			if currentBlock.Content == "" {
				currentBlock.Content = line
			} else {
				currentBlock.Content += "\n" + line
			}
		}
	}

	if currentBlock != nil {
		blocks = append(blocks, *currentBlock)
	}

	return blocks
}

// MatchBlocks finds blocks matching a reference
func (p *Parser) MatchBlocks(blocks []Block, ref string) []Block {
	var matches []Block
	refNorm := normalizeText(ref)

	for _, block := range blocks {
		contentNorm := normalizeText(block.Content)
		if strings.Contains(contentNorm, refNorm) {
			matches = append(matches, block)
		}
	}

	if len(matches) == 0 {
		p.addWarning("No blocks matched query '%s'", ref)
	}

	return matches
}

// AssembleContext builds context for a command
func (p *Parser) AssembleContext(blocks []Block, currentIndex int) []Block {
	var context []Block
	var parents []Block
	
	// Find current section's level
	currentLevel := 0
	if blocks[currentIndex].Type == Header {
		currentLevel = blocks[currentIndex].Level
	}

	// Collect parent headers
	for i := currentIndex - 1; i >= 0; i-- {
		block := blocks[i]
		if block.Type == Header {
			if block.Level < currentLevel {
				parents = append([]Block{block}, parents...)
				currentLevel = block.Level
			}
		}
	}

	// Add parent headers in order
	context = append(context, parents...)

	// Add current section
	if blocks[currentIndex].Type == Header {
		// For headers, add the header itself
		context = append(context, blocks[currentIndex])
	} else {
		// For non-headers, find the most recent parent header
		for i := currentIndex - 1; i >= 0; i-- {
			if blocks[i].Type == Header {
				context = append(context, blocks[i])
				break
			}
		}
		context = append(context, blocks[currentIndex])
	}

	// Add siblings (headers at same level)
	if blocks[currentIndex].Type == Header {
		level := blocks[currentIndex].Level
		for i := currentIndex + 1; i < len(blocks); i++ {
			block := blocks[i]
			if block.Type == Header {
				if block.Level < level {
					break
				}
				if block.Level == level {
					context = append(context, block)
				}
			}
		}
	}

	return context
}

// normalizeText prepares text for matching
func normalizeText(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)
	// Replace punctuation with spaces
	text = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(text, " ")
	// Collapse whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	// Trim
	return strings.TrimSpace(text)
}
