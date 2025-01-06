package context

import (
	"strings"
	"testing"
)

func TestParseReferences(t *testing.T) {
	content := `# Header 1
Content 1

## Header 2
Content 2

### Header 3
Content 3

## Header 4
Content 4
More content`

	refs := ParseReferences(content)

	tests := []struct {
		name       string
		header     string
		level      int
		hasContent bool
	}{
		{
			name:       "top level header",
			header:     "Header 1",
			level:      1,
			hasContent: true,
		},
		{
			name:       "second level header",
			header:     "Header 2",
			level:      2,
			hasContent: true,
		},
		{
			name:       "third level header",
			header:     "Header 3",
			level:      3,
			hasContent: true,
		},
		{
			name:       "another second level header",
			header:     "Header 4",
			level:      2,
			hasContent: true,
		},
	}

	if len(refs) != len(tests) {
		t.Errorf("Expected %d references, got %d", len(tests), len(refs))
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if refs[i].Header != tt.header {
				t.Errorf("Header = %v, want %v", refs[i].Header, tt.header)
			}
			if refs[i].Level != tt.level {
				t.Errorf("Level = %v, want %v", refs[i].Level, tt.level)
			}
			if tt.hasContent && refs[i].EndLine <= refs[i].StartLine {
				t.Error("Reference has no content lines")
			}
		})
	}
}

func TestAssembleContext(t *testing.T) {
	content := `# Section 1
This is content for section 1.
It has multiple lines.

## Subsection 1.1
This is a subsection.

# Section 2
This is content for section 2.

## Subsection 2.1
This is another subsection.`

	tests := []struct {
		name      string
		headers   []string
		maxSize   int
		maxTokens int
		wantCount int
		wantError bool
	}{
		{
			name:      "single section",
			headers:   []string{"Section 1"},
			maxSize:   1000,
			maxTokens: 1000,
			wantCount: 1,
			wantError: false,
		},
		{
			name:      "multiple sections",
			headers:   []string{"Section 1", "Section 2"},
			maxSize:   1000,
			maxTokens: 1000,
			wantCount: 2,
			wantError: false,
		},
		{
			name:      "size limit",
			headers:   []string{"Section 1", "Section 2"},
			maxSize:   10,
			maxTokens: 1000,
			wantCount: 0,
			wantError: false,
		},
		{
			name:      "token limit",
			headers:   []string{"Section 1", "Section 2"},
			maxSize:   1000,
			maxTokens: 1,
			wantCount: 0,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := AssembleContext(content, tt.headers, tt.maxSize, tt.maxTokens)
			if (err != nil) != tt.wantError {
				t.Errorf("AssembleContext() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if len(ctx.References) != tt.wantCount {
					t.Errorf("Got %d references, want %d", len(ctx.References), tt.wantCount)
				}
			}
		})
	}
}

func TestTruncateContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxSize  int
		wantType string // "sentence", "word", or "hard"
	}{
		{
			name:     "sentence boundary",
			content:  "First sentence. Second sentence. Third sentence.",
			maxSize:  20,
			wantType: "sentence",
		},
		{
			name:     "word boundary",
			content:  "These are some words without periods",
			maxSize:  15,
			wantType: "word",
		},
		{
			name:     "hard truncate",
			content:  "NoSpacesOrPeriodsHere",
			maxSize:  10,
			wantType: "hard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateContent(tt.content, tt.maxSize)
			if len(result) > tt.maxSize {
				t.Errorf("Truncated content length %d exceeds max size %d", len(result), tt.maxSize)
			}

			switch tt.wantType {
			case "sentence":
				if !strings.HasSuffix(result, ". ") && !strings.HasSuffix(result, ".\n") {
					t.Error("Expected truncation at sentence boundary")
				}
			case "word":
				if !strings.HasSuffix(result, " ") && !strings.HasSuffix(result, "\n") {
					t.Error("Expected truncation at word boundary")
				}
			}
		})
	}
}

func TestHeaderHierarchy(t *testing.T) {
	content := `# Top Level
Content

## Section 1
Content 1

### Subsection 1.1
Content 1.1

## Section 2
Content 2

### Subsection 2.1
Content 2.1

## Section 3
Content 3`

	refs := ParseReferences(content)

	// Test parent header resolution
	tests := []struct {
		name       string
		header     string
		wantParent string
	}{
		{
			name:       "top level has no parent",
			header:     "Top Level",
			wantParent: "",
		},
		{
			name:       "section has top level parent",
			header:     "Section 1",
			wantParent: "Top Level",
		},
		{
			name:       "subsection has section parent",
			header:     "Subsection 1.1",
			wantParent: "Section 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetParentHeader(refs, tt.header)
			if got != tt.wantParent {
				t.Errorf("GetParentHeader() = %v, want %v", got, tt.wantParent)
			}
		})
	}

	// Test sibling header resolution
	siblingTests := []struct {
		name     string
		header   string
		wantPrev string
		wantNext string
	}{
		{
			name:     "first section",
			header:   "Section 1",
			wantPrev: "",
			wantNext: "Section 2",
		},
		{
			name:     "middle section",
			header:   "Section 2",
			wantPrev: "Section 1",
			wantNext: "Section 3",
		},
		{
			name:     "last section",
			header:   "Section 3",
			wantPrev: "Section 2",
			wantNext: "",
		},
	}

	for _, tt := range siblingTests {
		t.Run(tt.name, func(t *testing.T) {
			prev, next := GetSiblingHeaders(refs, tt.header)
			if prev != tt.wantPrev {
				t.Errorf("Previous sibling = %v, want %v", prev, tt.wantPrev)
			}
			if next != tt.wantNext {
				t.Errorf("Next sibling = %v, want %v", next, tt.wantNext)
			}
		})
	}
}
