package assistant

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/provider"
	"github.com/butter-bot-machines/skylark/pkg/provider/registry"
	"github.com/butter-bot-machines/skylark/pkg/sandbox"
	"github.com/butter-bot-machines/skylark/pkg/tool"
)

// TestAssistantProviderIntegration verifies the integration between Assistant,
// Provider, and Tools, focusing on:
// - Direct tool usage through commands
// - Provider-initiated tool calls
// - Error handling across components
// - Context management and tool result inclusion
func TestAssistantProviderIntegration(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		responses    []provider.Response
		toolResult   string
		wantExecuted bool
		wantRequests int
		wantResponse string
		wantError    bool
	}{
		{
			name:    "direct tool usage",
			command: "use summarize test input",
			responses: []provider.Response{
				{Content: "Here's the summary: test result"},
			},
			toolResult:   "test result",
			wantExecuted: true,
			wantRequests: 1,
			wantResponse: "Here's the summary: test result",
		},
		{
			name:    "provider tool call",
			command: "summarize this",
			responses: []provider.Response{
				{
					Content: "Let me help summarize that",
					ToolCalls: []provider.ToolCall{
						{
							ID: "call_1",
							Function: provider.Function{
								Name:      "summarize",
								Arguments: `{"text":"test"}`,
							},
						},
					},
				},
				{Content: "Here's the summary: test result"},
			},
			toolResult:   "test result",
			wantExecuted: true,
			wantRequests: 2,
			wantResponse: "Here's the summary: test result",
		},
		{
			name:    "provider error",
			command: "break things",
			responses: []provider.Response{
				{Error: &provider.Error{Code: provider.ErrServerError}},
			},
			wantRequests: 1,
			wantError:    true,
		},
		{
			name:      "tool not found",
			command:   "use nonexistent test input",
			wantError: true,
		},
		{
			name:       "tool execution error",
			command:    "use summarize test input",
			toolResult: "test result",
			responses: []provider.Response{
				{Content: "Here's the summary"},
			},
			wantExecuted: true,
			wantError:    true,
		},
		{
			name:    "multiple tool calls",
			command: "analyze this",
			responses: []provider.Response{
				{
					Content: "Let me analyze that",
					ToolCalls: []provider.ToolCall{
						{
							ID: "call_1",
							Function: provider.Function{
								Name:      "summarize",
								Arguments: `{"text":"first"}`,
							},
						},
						{
							ID: "call_2",
							Function: provider.Function{
								Name:      "summarize",
								Arguments: `{"text":"second"}`,
							},
						},
					},
				},
				{Content: "Here's the analysis with both summaries"},
			},
			toolResult:   "test result",
			wantExecuted: true,
			wantRequests: 2,
			wantResponse: "Here's the analysis with both summaries",
		},
		{
			name:    "tool result in context",
			command: "use summarize test input",
			responses: []provider.Response{
				{Content: "Here's the summary with context: test result"},
			},
			toolResult:   "test result",
			wantExecuted: true,
			wantRequests: 1,
			wantResponse: "Here's the summary with context: test result",
		},
		{
			name:    "context size limit",
			command: "use summarize " + strings.Repeat("x", 8000), // Exceed context limit
			responses: []provider.Response{
				{Content: "Content too large"},
			},
			wantError: true,
		},
		{
			name:    "tool timeout",
			command: "use summarize test input",
			responses: []provider.Response{
				{Content: "Tool timed out"},
			},
			toolResult: "sleep", // Special value to trigger sleep in mock tool
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			tempDir := t.TempDir()
			toolsDir := filepath.Join(tempDir, "tools")
			if err := os.MkdirAll(toolsDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create mock tool binary
			toolDir := filepath.Join(toolsDir, "summarize")
			if err := os.MkdirAll(toolDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create mock tool source
			mainGo := fmt.Sprintf(`package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--usage" {
		fmt.Print(`+"`"+`{"schema":{"name":"summarize","description":"Test tool","parameters":{"type":"object","properties":{"content":{"type":"string"}}}}}`+"`"+`)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--health" {
		fmt.Print(`+"`"+`{"status":true,"details":"healthy"}`+"`"+`)
		return
	}

	// Read and discard input
	io.ReadAll(os.Stdin)

	if %q == "tool execution error" {
		os.Exit(1)
	}

	if %q == "sleep" {
		// Sleep longer than sandbox timeout (100ms)
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Print(%q)
}`, tt.name, tt.toolResult, tt.toolResult)

			if err := os.WriteFile(filepath.Join(toolDir, "main.go"), []byte(mainGo), 0644); err != nil {
				t.Fatal(err)
			}

			// Create test components
			testProv := &testProvider{responses: tt.responses}
			toolMgr := tool.NewManager(toolsDir)

			// Compile tool
			if err := toolMgr.Compile("summarize"); err != nil {
				t.Fatal(err)
			}

			// Create sandbox with test limits
			testLimits := &sandbox.ResourceLimits{
				MaxCPUTime:    100 * time.Millisecond,
				MaxMemoryMB:   512,
				MaxFileSizeMB: 10,
				MaxFiles:      100,
				MaxProcesses:  10,
			}
			sb, err := sandbox.NewSandbox(toolsDir, testLimits, &sandbox.NetworkPolicy{
				AllowOutbound: false,
				AllowInbound:  false,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Create provider registry
			reg := registry.New()
			reg.Register("test", func(model string) (provider.Provider, error) {
				return testProv, nil
			})

			// Create assistant
			assistant := &Assistant{
				Name:            "test",
				Tools:           []string{"summarize"},
				Model:           "test:model",
				toolMgr:         toolMgr,
				providers:       reg,
				defaultProvider: "test",
				sandbox:         sb,
				logger:          slog.Default(), // Use default logger for tests
			}

			// Parse command
			p := parser.New()
			cmd, err := p.ParseCommand("!test " + tt.command)
			if err != nil {
				if tt.wantError {
					return
				}
				t.Fatal(err)
			}
			resp, err := assistant.Process(cmd)

			// Verify error handling
			if (err != nil) != tt.wantError {
				t.Errorf("Process() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// Verify provider interaction
			if len(testProv.requests) != tt.wantRequests {
				t.Errorf("Got %d provider requests, want %d", len(testProv.requests), tt.wantRequests)
			}

			// Verify response
			if !tt.wantError && resp != tt.wantResponse {
				t.Errorf("Got response %q, want %q", resp, tt.wantResponse)
			}

			// Verify context in provider requests
			if tt.name == "tool result in context" && len(testProv.requests) > 0 {
				request := testProv.requests[0]
				if !strings.Contains(request, "Tool result: test result") {
					t.Error("Tool result not found in provider context")
				}
			}
		})
	}
}
