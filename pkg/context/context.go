package context

import (
	"regexp"
	"strings"
)

// Reference represents a section reference in a document
type Reference struct {
	Header    string
	Level     int
	StartLine int
	EndLine   int
}

// Context represents assembled context from references
type Context struct {
	References map[string]string // Map of header to content
	TotalSize  int              // Total size in characters
	TokenCount int              // Estimated token count
}

// HeaderPattern matches Markdown headers with level capture
var HeaderPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)(?:\s+#*)?$`)

// ParseReferences finds all section references in a document
func ParseReferences(content string) []Reference {
	var refs []Reference
	var currentRef *Reference
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		if matches := HeaderPattern.FindStringSubmatch(line); matches != nil {
			// If we have a current reference, set its end line
			if currentRef != nil {
				currentRef.EndLine = i - 1
				if currentRef.EndLine >= currentRef.StartLine {
					refs = append(refs, *currentRef)
				}
			}

			// Start new reference
			currentRef = &Reference{
				Header:    strings.TrimSpace(matches[2]),
				Level:     len(matches[1]), // Number of # characters
				StartLine: i + 1,          // Content starts after header
			}
		}
	}

	// Handle last reference
	if currentRef != nil {
		currentRef.EndLine = len(lines) - 1
		if currentRef.EndLine >= currentRef.StartLine {
			refs = append(refs, *currentRef)
		}
	}

	return refs
}

// AssembleContext creates a context from references with size limits
func AssembleContext(content string, headers []string, maxSize, maxTokens int) (*Context, error) {
	// Parse all references
	refs := ParseReferences(content)

	// Create header map for quick lookup
	headerMap := make(map[string]bool)
	for _, h := range headers {
		headerMap[strings.ToLower(h)] = true
	}

	// Initialize context
	ctx := &Context{
		References: make(map[string]string),
	}

	// Find matching references
	var matches []Reference
	for _, ref := range refs {
		if headerMap[strings.ToLower(ref.Header)] {
			matches = append(matches, ref)
		}
	}

	// Sort references by priority
	sortReferencesByPriority(matches)

	// Extract content within limits
	lines := strings.Split(content, "\n")
	for _, ref := range matches {
		// Get section content
		if ref.EndLine >= len(lines) {
			ref.EndLine = len(lines) - 1
		}
		sectionContent := strings.Join(lines[ref.StartLine:ref.EndLine+1], "\n")

		// Check size limits
		sectionSize := len(sectionContent)
		sectionTokens := estimateTokenCount(sectionContent)

		if ctx.TotalSize+sectionSize > maxSize {
			// Try truncating
			available := maxSize - ctx.TotalSize
			if available > 100 { // Only include if we can get meaningful content
				sectionContent = truncateContent(sectionContent, available)
				sectionSize = len(sectionContent)
				sectionTokens = estimateTokenCount(sectionContent)
			} else {
				continue // Skip this section
			}
		}

		if ctx.TokenCount+sectionTokens > maxTokens {
			continue // Skip this section
		}

		// Add to context
		ctx.References[ref.Header] = sectionContent
		ctx.TotalSize += sectionSize
		ctx.TokenCount += sectionTokens
	}

	return ctx, nil
}

// sortReferencesByPriority sorts references by their priority
func sortReferencesByPriority(refs []Reference) {
	// Sort by:
	// 1. Header level (lower levels first)
	// 2. Position in document (earlier first)
	for i := 0; i < len(refs)-1; i++ {
		for j := i + 1; j < len(refs); j++ {
			if refs[i].Level > refs[j].Level ||
				(refs[i].Level == refs[j].Level && refs[i].StartLine > refs[j].StartLine) {
				refs[i], refs[j] = refs[j], refs[i]
			}
		}
	}
}

// estimateTokenCount provides a rough estimate of token count
func estimateTokenCount(text string) int {
	// Simple estimation: ~4 characters per token
	return len(text) / 4
}

// truncateContent truncates content to fit within maxSize while preserving meaning
func truncateContent(content string, maxSize int) string {
	if len(content) <= maxSize {
		return content
	}

	// Try to truncate at sentence boundary
	if idx := findLastSentenceBoundary(content[:maxSize]); idx > 0 {
		return content[:idx]
	}

	// Fall back to word boundary
	if idx := findLastWordBoundary(content[:maxSize]); idx > 0 {
		return content[:idx]
	}

	// Last resort: hard truncate
	return content[:maxSize]
}

// findLastSentenceBoundary finds the last sentence boundary before position
func findLastSentenceBoundary(text string) int {
	// Look for period followed by space or newline
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == '.' {
			// Make sure we're at the end of a sentence
			if i == len(text)-1 || text[i+1] == ' ' || text[i+1] == '\n' {
				// Include the period and any trailing space
				if i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\n') {
					return i + 2
				}
				return i + 1
			}
		}
	}
	return -1
}

// findLastWordBoundary finds the last word boundary before position
func findLastWordBoundary(text string) int {
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == ' ' || text[i] == '\n' {
			// Include the space to maintain readability
			return i + 1
		}
	}
	return -1
}

// GetParentHeader returns the parent header for a given header
func GetParentHeader(refs []Reference, header string) string {
	var targetRef *Reference
	for i := range refs {
		if refs[i].Header == header {
			targetRef = &refs[i]
			break
		}
	}

	if targetRef == nil {
		return ""
	}

	// Look for the nearest header with lower level
	for i := len(refs) - 1; i >= 0; i-- {
		if refs[i].StartLine < targetRef.StartLine && refs[i].Level < targetRef.Level {
			return refs[i].Header
		}
	}

	return ""
}

// GetSiblingHeaders returns the previous and next sibling headers
func GetSiblingHeaders(refs []Reference, header string) (prev, next string) {
	var targetRef *Reference
	var targetIndex int
	for i := range refs {
		if refs[i].Header == header {
			targetRef = &refs[i]
			targetIndex = i
			break
		}
	}

	if targetRef == nil {
		return "", ""
	}

	// Look for siblings (same level headers)
	// Previous sibling
	for i := targetIndex - 1; i >= 0; i-- {
		if refs[i].Level == targetRef.Level {
			prev = refs[i].Header
			break
		}
	}

	// Next sibling
	for i := targetIndex + 1; i < len(refs); i++ {
		if refs[i].Level == targetRef.Level {
			next = refs[i].Header
			break
		}
	}

	return prev, next
}
