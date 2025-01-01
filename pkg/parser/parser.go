package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Pattern for commands with explicit assistant
	// Assistant must contain a hyphen to distinguish it from regular commands
	assistantCommandPattern = regexp.MustCompile(`^!([a-zA-Z]+-[a-zA-Z-]+)\s+(.+)$`)
	// Pattern for commands without assistant
	defaultCommandPattern = regexp.MustCompile(`^!(.+)$`)

	// referencePattern matches header references
	// Example: # Header Text #
	// Example: # Header Text
	referencePattern = regexp.MustCompile(`#\s*([^#\n]+?)(?:\s*#|$)`)
)

// Command represents a parsed command with its context
type Command struct {
	Assistant  string            // Name of the assistant to handle the command
	Text       string            // The command text to process
	References []string          // Referenced section headers
	Context    map[string]string // Map of section headers to their content
}

// ParseCommand parses a line of text to extract command information
func ParseCommand(line string) (*Command, error) {
	// First try to match command with assistant
	if matches := assistantCommandPattern.FindStringSubmatch(line); matches != nil {
		return &Command{
			Assistant:  strings.ToLower(matches[1]),
			Text:       strings.TrimSpace(matches[2]),
			References: make([]string, 0),
			Context:    make(map[string]string),
		}, nil
	}

	// Then try to match command without assistant
	if matches := defaultCommandPattern.FindStringSubmatch(line); matches != nil {
		return &Command{
			Assistant:  "default",
			Text:       strings.TrimSpace(matches[1]),
			References: make([]string, 0),
			Context:    make(map[string]string),
		}, nil
	}

	return nil, fmt.Errorf("invalid command format: %s", line)
}

// ParseReferences extracts header references from the command text
func ParseReferences(text string) []string {
	matches := referencePattern.FindAllStringSubmatch(text, -1)
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, strings.TrimSpace(match[1]))
	}
	return refs
}

// ExtractContext extracts content for referenced sections from the document
func ExtractContext(cmd *Command, content string, maxSectionSize int, maxTotalSize int) error {
	if maxSectionSize <= 0 {
		maxSectionSize = 4000 // Default max section size as per plan
	}
	if maxTotalSize <= 0 {
		maxTotalSize = 8000 // Default total context size as per plan
	}

	sections := splitIntoSections(content)
	totalSize := 0

	for header, sectionContent := range sections {
		for _, ref := range cmd.References {
			if strings.EqualFold(header, ref) {
				// Truncate section if it exceeds max size
				if len(sectionContent) > maxSectionSize {
					sectionContent = sectionContent[:maxSectionSize]
				}

				// Check if adding this section would exceed total size
				if totalSize+len(sectionContent) > maxTotalSize {
					return fmt.Errorf("total context size would exceed limit of %d bytes", maxTotalSize)
				}

				cmd.Context[header] = sectionContent
				totalSize += len(sectionContent)
			}
		}
	}

	return nil
}

// splitIntoSections splits document content into header-content pairs
func splitIntoSections(content string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(content, "\n")
	var currentHeader string
	var currentContent []string

	for _, line := range lines {
		if matches := referencePattern.FindStringSubmatch(line); matches != nil {
			// Save previous section if it exists
			if currentHeader != "" {
				sections[currentHeader] = strings.Join(currentContent, "\n")
			}
			// Start new section
			currentHeader = strings.TrimSpace(matches[1])
			currentContent = make([]string, 0)
		} else if currentHeader != "" {
			currentContent = append(currentContent, line)
		}
	}

	// Save last section
	if currentHeader != "" {
		sections[currentHeader] = strings.Join(currentContent, "\n")
	}

	return sections
}
