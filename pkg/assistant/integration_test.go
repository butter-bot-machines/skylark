package assistant

import (
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
			command: "use test-mock",
			responses: []provider.Response{
				{Content: "The result is success"},
			},
			toolResult:   `{"result":"success"}`,
			wantExecuted: true,
			wantRequests: 1,
			wantResponse: "The result is success",
		},
		{
			name:    "provider tool call",
			command: "run test",
			responses: []provider.Response{
				{
					Content: "Let me run that",
					ToolCalls: []provider.ToolCall{
						{
							ID: "call_1",
							Function: provider.Function{
								Name:      "test-mock",
								Arguments: `{}`,
							},
						},
					},
				},
				{Content: "The result is success"},
			},
			toolResult:   `{"result":"success"}`,
			wantExecuted: true,
			wantRequests: 2,
			wantResponse: "The result is success",
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
			name:    "tool execution error",
			command: "use test-mock error",
			responses: []provider.Response{
				{Content: "Failed to execute"},
			},
			toolResult:   `{"error":"test error"}`,
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
								Name:      "test-mock",
								Arguments: `{"mode":"first"}`,
							},
						},
						{
							ID: "call_2",
							Function: provider.Function{
								Name:      "test-mock",
								Arguments: `{"mode":"second"}`,
							},
						},
					},
				},
				{Content: "Here's the analysis with both results"},
			},
			toolResult:   `{"result":"test"}`,
			wantExecuted: true,
			wantRequests: 2,
			wantResponse: "Here's the analysis with both results",
		},
		{
			name:    "tool result in context",
			command: "use test-mock",
			responses: []provider.Response{
				{Content: "The result is success"},
			},
			toolResult:   `{"result":"success"}`,
			wantExecuted: true,
			wantRequests: 1,
			wantResponse: "The result is success",
		},
		{
			name:    "context size limit",
			command: "use test-mock " + strings.Repeat("x", 8000), // Exceed context limit
			responses: []provider.Response{
				{Content: "Content too large"},
			},
			wantError: true,
		},
		{
			name:    "tool timeout",
			command: "use test-mock timeout",
			responses: []provider.Response{
				{Content: "Tool timed out"},
			},
			toolResult: `{"mode":"timeout"}`,
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
			toolDir := filepath.Join(toolsDir, "test-mock")
			if err := os.MkdirAll(toolDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create mock tool source
			mainGo := `package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	usage := flag.Bool("usage", false, "Display usage schema")
	health := flag.Bool("health", false, "Check tool health")
	flag.Parse()

	if *usage {
		fmt.Print(` + "`" + `{"schema":{"name":"test-mock","description":"Test tool","parameters":{"type":"object","properties":{"mode":{"type":"string","description":"Test mode"}}}},"env":{}}` + "`" + `)
		return
	}

	if *health {
		fmt.Print(` + "`" + `{"status":true,"details":"healthy"}` + "`" + `)
		return
	}

	// Read input
	input, _ := io.ReadAll(os.Stdin)
	
	// Parse input
	var params struct {
		Mode string ` + "`json:\"mode\"`" + `
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &params); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid input: %v\n", err)
			os.Exit(1)
		}
	}

	// Handle test modes
	switch params.Mode {
	case "error":
		fmt.Fprintf(os.Stderr, "Test error\n")
		os.Exit(1)
	case "timeout":
		time.Sleep(200 * time.Millisecond)
	}

	// Return mock result
	result := os.Getenv("MOCK_RESULT")
	if result == "" {
		result = ` + "`" + `{"result":"success"}` + "`" + `
	}
	fmt.Print(result)
}`

			// Write tool source
			if err := os.WriteFile(filepath.Join(toolDir, "main.go"), []byte(mainGo), 0644); err != nil {
				t.Fatal(err)
			}

			// Create test components
			testProv := &testProvider{responses: tt.responses}
			toolMgr, err := tool.NewManager(toolsDir)
			if err != nil {
				t.Fatal(err)
			}
			defer toolMgr.Close()

			// Compile tool
			if err := toolMgr.Compile("test-mock"); err != nil {
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

			// Set environment variables
			sb.EnvWhitelist = []string{"MOCK_RESULT"}
			if err := os.Setenv("MOCK_RESULT", tt.toolResult); err != nil {
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
				Tools:           []string{"test-mock"},
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
				normalized := strings.Map(func(r rune) rune {
					if r == ' ' || r == '\n' {
						return -1
					}
					return r
				}, request)
				if !strings.Contains(normalized, `Toolresult:{"result":"success"}`) {
					t.Logf("Got request: %s", request)
					t.Error("Tool result not found in provider context")
				}
			}
		})
	}
}
