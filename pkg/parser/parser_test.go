package parser

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantAssistant string
		wantText  string
		wantError bool
	}{
		{
			name:  "command with assistant",
			input: "!test-assistant generate summary",
			wantAssistant: "test-assistant",
			wantText: "generate summary",
			wantError: false,
		},
		{
			name:  "command without assistant",
			input: "!generate summary",
			wantAssistant: "default",
			wantText: "generate summary",
			wantError: false,
		},
		{
			name:      "invalid command format",
			input:     "generate summary",
			wantAssistant: "",
			wantText: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseCommand() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if got.Assistant != tt.wantAssistant {
					t.Errorf("ParseCommand() Assistant = %v, want %v", got.Assistant, tt.wantAssistant)
				}
				if got.Text != tt.wantText {
					t.Errorf("ParseCommand() Text = %v, want %v", got.Text, tt.wantText)
				}
			}
		})
	}
}

func TestParseReferences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single reference",
			input: "# Section 1",
			want:  []string{"Section 1"},
		},
		{
			name:  "multiple references",
			input: "# Section 1 #\n# Section 2 #",
			want:  []string{"Section 1", "Section 2"},
		},
		{
			name:  "reference with trailing hash",
			input: "# Section 1 #",
			want:  []string{"Section 1"},
		},
		{
			name:  "no references",
			input: "Plain text",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseReferences(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseReferences() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractContext(t *testing.T) {
	sampleContent := `# Section 1
Content for section 1
More content

# Section 2
Content for section 2

# Section 3
Content for section 3`

	tests := []struct {
		name           string
		cmd           *Command
		maxSectionSize int
		maxTotalSize   int
		wantError      bool
		wantContextLen int
	}{
		{
			name: "single section",
			cmd: &Command{
				References: []string{"Section 1"},
				Context:    make(map[string]string),
			},
			maxSectionSize: 1000,
			maxTotalSize:   2000,
			wantError:      false,
			wantContextLen: 1,
		},
		{
			name: "multiple sections",
			cmd: &Command{
				References: []string{"Section 1", "Section 2"},
				Context:    make(map[string]string),
			},
			maxSectionSize: 1000,
			maxTotalSize:   2000,
			wantError:      false,
			wantContextLen: 2,
		},
		{
			name: "section size limit",
			cmd: &Command{
				References: []string{"Section 1"},
				Context:    make(map[string]string),
			},
			maxSectionSize: 10,
			maxTotalSize:   2000,
			wantError:      false,
			wantContextLen: 1,
		},
		{
			name: "total size limit",
			cmd: &Command{
				References: []string{"Section 1", "Section 2", "Section 3"},
				Context:    make(map[string]string),
			},
			maxSectionSize: 1000,
			maxTotalSize:   10,
			wantError:      true,
			wantContextLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExtractContext(tt.cmd, sampleContent, tt.maxSectionSize, tt.maxTotalSize)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractContext() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if len(tt.cmd.Context) != tt.wantContextLen {
				t.Errorf("ExtractContext() got %d sections, want %d", len(tt.cmd.Context), tt.wantContextLen)
			}
		})
	}
}

func TestSplitIntoSections(t *testing.T) {
	input := `# Section 1
Content for section 1
More content

# Section 2
Content for section 2

# Section 3 #
Content for section 3`

	want := map[string]string{
		"Section 1": "Content for section 1\nMore content\n",
		"Section 2": "Content for section 2\n",
		"Section 3": "Content for section 3",
	}

	got := splitIntoSections(input)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitIntoSections() = %v, want %v", got, want)
	}
}
