package parser

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      *Command
		wantError bool
	}{
		{
			name:  "basic command",
			input: "!command text",
			want: &Command{
				Assistant: "command",
				Text:      "text",
				Original:  "!command text",
				Context:   make(map[string]Block),
			},
		},
		{
			name:  "with whitespace",
			input: "  !command   text  ",
			want: &Command{
				Assistant: "command",
				Text:      "text",
				Original:  "!command   text",
				Context:   make(map[string]Block),
			},
		},
		{
			name:  "uppercase assistant",
			input: "!ASSISTANT help me",
			want: &Command{
				Assistant: "assistant",
				Text:      "help me",
				Original:  "!ASSISTANT help me",
				Context:   make(map[string]Block),
			},
		},
		{
			name:  "with references",
			input: "!assistant analyze # Section 1 # and # Section 2 #",
			want: &Command{
				Assistant:  "assistant",
				Text:       "analyze # Section 1 # and # Section 2 #",
				Original:   "!assistant analyze # Section 1 # and # Section 2 #",
				References: []string{"Section 1", "Section 2"},
				Context:    make(map[string]Block),
			},
		},
		{
			name:      "missing prefix",
			input:     "command text",
			wantError: true,
		},
		{
			name:      "! after text",
			input:     "hello !command",
			wantError: true,
		},
		{
			name:      "exceeds size",
			input:     "!" + strings.Repeat("x", maxCommandSize+1),
			wantError: true,
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.ParseCommand(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseCommand() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBlocks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Block
	}{
		{
			name: "headers",
			content: `# Header 1
Content 1
## Header 2
Content 2`,
			want: []Block{
				{Type: Header, Level: 1, Content: "Header 1"},
				{Type: Paragraph, Content: "Content 1"},
				{Type: Header, Level: 2, Content: "Header 2"},
				{Type: Paragraph, Content: "Content 2"},
			},
		},
		{
			name: "lists",
			content: `- Item 1
- Item 2
* Item 3`,
			want: []Block{
				{Type: List, Content: "- Item 1\n- Item 2\n* Item 3"},
			},
		},
		{
			name: "quotes",
			content: `> Quote 1
> Quote 2`,
			want: []Block{
				{Type: Quote, Content: "Quote 1\nQuote 2"},
			},
		},
		{
			name:    "code blocks",
			content: "```\ncode line 1\ncode line 2\n```",
			want: []Block{
				{Type: Code, Content: "code line 1\ncode line 2"},
			},
		},
		{
			name: "tables",
			content: `| A | B |
|-|-|
|1|2|`,
			want: []Block{
				{Type: Table, Content: "| A | B |\n|-|-|\n|1|2|"},
			},
		},
		{
			name: "mixed content",
			content: `# Section
Paragraph 1

- List item 1
- List item 2

> Quote text

` + "```" + `
code
` + "```",
			want: []Block{
				{Type: Header, Level: 1, Content: "Section"},
				{Type: Paragraph, Content: "Paragraph 1"},
				{Type: List, Content: "- List item 1\n- List item 2"},
				{Type: Quote, Content: "Quote text"},
				{Type: Code, Content: "code"},
			},
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.ParseBlocks(tt.content)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssembleContext(t *testing.T) {
	tests := []struct {
		name         string
		blocks       []Block
		currentIndex int
		want         []Block
	}{
		{
			name: "header with parents",
			blocks: []Block{
				{Type: Header, Level: 1, Content: "Parent"},
				{Type: Header, Level: 2, Content: "Child"},
				{Type: Paragraph, Content: "Content"},
			},
			currentIndex: 1,
			want: []Block{
				{Type: Header, Level: 1, Content: "Parent"},
				{Type: Header, Level: 2, Content: "Child"},
			},
		},
		{
			name: "header with siblings",
			blocks: []Block{
				{Type: Header, Level: 1, Content: "Section 1"},
				{Type: Header, Level: 1, Content: "Section 2"},
				{Type: Header, Level: 1, Content: "Section 3"},
			},
			currentIndex: 1,
			want: []Block{
				{Type: Header, Level: 1, Content: "Section 2"},
				{Type: Header, Level: 1, Content: "Section 3"},
			},
		},
		{
			name: "nested headers",
			blocks: []Block{
				{Type: Header, Level: 1, Content: "Top"},
				{Type: Header, Level: 2, Content: "Middle"},
				{Type: Header, Level: 3, Content: "Current"},
				{Type: Header, Level: 3, Content: "Sibling"},
				{Type: Header, Level: 2, Content: "Other"},
			},
			currentIndex: 2,
			want: []Block{
				{Type: Header, Level: 1, Content: "Top"},
				{Type: Header, Level: 2, Content: "Middle"},
				{Type: Header, Level: 3, Content: "Current"},
				{Type: Header, Level: 3, Content: "Sibling"},
			},
		},
		{
			name: "non-header block",
			blocks: []Block{
				{Type: Header, Level: 1, Content: "Section"},
				{Type: Paragraph, Content: "Current"},
				{Type: Paragraph, Content: "More"},
			},
			currentIndex: 1,
			want: []Block{
				{Type: Header, Level: 1, Content: "Section"},
				{Type: Paragraph, Content: "Current"},
			},
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.AssembleContext(tt.blocks, tt.currentIndex)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AssembleContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchBlocks(t *testing.T) {
	tests := []struct {
		name      string
		blocks    []Block
		ref       string
		want      []Block
		wantWarns []string
	}{
		{
			name: "exact match",
			blocks: []Block{
				{Type: Header, Content: "Section One"},
				{Type: Header, Content: "Section Two"},
			},
			ref: "Section One",
			want: []Block{
				{Type: Header, Content: "Section One"},
			},
			wantWarns: []string{},
		},
		{
			name: "partial match",
			blocks: []Block{
				{Type: Header, Content: "Section One"},
				{Type: Header, Content: "Section Two"},
			},
			ref: "One",
			want: []Block{
				{Type: Header, Content: "Section One"},
			},
			wantWarns: []string{},
		},
		{
			name: "case insensitive",
			blocks: []Block{
				{Type: Header, Content: "Section One"},
			},
			ref: "SECTION",
			want: []Block{
				{Type: Header, Content: "Section One"},
			},
			wantWarns: []string{},
		},
		{
			name: "multiple matches",
			blocks: []Block{
				{Type: Header, Content: "Section One"},
				{Type: Paragraph, Content: "More about section one"},
			},
			ref: "section",
			want: []Block{
				{Type: Header, Content: "Section One"},
				{Type: Paragraph, Content: "More about section one"},
			},
			wantWarns: []string{},
		},
		{
			name: "no matches",
			blocks: []Block{
				{Type: Header, Content: "Section One"},
			},
			ref:       "Missing",
			want:      nil,
			wantWarns: []string{"No blocks matched query 'Missing'"},
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.ClearWarnings()
			got := p.MatchBlocks(tt.blocks, tt.ref)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MatchBlocks() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(p.GetWarnings(), tt.wantWarns) {
				t.Errorf("Warnings = %v, want %v", p.GetWarnings(), tt.wantWarns)
			}
		})
	}
}
