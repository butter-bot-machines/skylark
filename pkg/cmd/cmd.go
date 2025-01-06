package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/job"
	"github.com/butter-bot-machines/skylark/pkg/logging"
	slogging "github.com/butter-bot-machines/skylark/pkg/logging/slog"
	"github.com/butter-bot-machines/skylark/pkg/processor/concrete"
	wconcrete "github.com/butter-bot-machines/skylark/pkg/watcher/concrete"
	"github.com/butter-bot-machines/skylark/pkg/worker"
	wkconcrete "github.com/butter-bot-machines/skylark/pkg/worker/concrete"
)

const Version = "0.1.0"

// CLI represents the command-line interface
type CLI struct {
	config *config.Manager
	logger logging.Logger
}

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	return &CLI{
		logger: slogging.NewLogger(logging.LevelDebug, os.Stdout),
	}
}

// Run executes the CLI with the given arguments
func (c *CLI) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("expected 'init', 'watch', 'run' or 'version' subcommands")
	}

	switch args[0] {
	case "init":
		return c.Init(args[1:])
	case "watch":
		return c.Watch(args[1:])
	case "run":
		return c.RunOnce(args[1:])
	case "version":
		return c.Version(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

// Init initializes a new Skylark project
func (c *CLI) Init(args []string) error {
	var projectDir string
	if len(args) > 0 {
		// Create named project directory
		projectDir = args[0]
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}
	} else {
		// Use current directory
		var err error
		projectDir = "."
		if projectDir, err = filepath.Abs(projectDir); err != nil {
			return fmt.Errorf("failed to resolve current directory: %w", err)
		}
	}

	// Create .skai directory structure
	skaiDir := filepath.Join(projectDir, ".skai")
	dirs := []string{
		filepath.Join(skaiDir, "assistants", "default"),
		filepath.Join(skaiDir, "assistants", "default", "knowledge"),
		filepath.Join(skaiDir, "tools"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create default config.yaml
	configContent := `version: "1.0"

environment:
  log_level: "info"
  log_file: "skylark.log"

models:
  openai:
    gpt-4:
      api_key: "${OPENAI_API_KEY}"
      temperature: 0.7
      max_tokens: 2000
      top_p: 0.9
    gpt-3.5-turbo:
      api_key: "${OPENAI_API_KEY}"
      temperature: 0.5
      max_tokens: 1000
      top_p: 0.9

tools:
  currentdatetime: {}  # Builtin tool, no config needed
  web_search:
    env:
      TIMEOUT: "30s"

workers:
  count: 4
  queue_size: 100

file_watch:
  debounce_delay: "500ms"
  max_delay: "2s"
  extensions:
    - ".md"

watch_paths:
  - "."
`
	if err := os.WriteFile(filepath.Join(skaiDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create config.yaml: %w", err)
	}

	// Create default assistant prompt.md
	promptContent := `---
name: default
description: Default assistant for general tasks
model: gpt-4
---
You are a helpful assistant that provides accurate and concise information.

When processing commands, you should:
1. Understand the user's request thoroughly
2. Consider any provided context
3. Use available tools when appropriate
4. Provide clear, well-structured responses
`
	if err := os.WriteFile(filepath.Join(skaiDir, "assistants", "default", "prompt.md"), []byte(promptContent), 0644); err != nil {
		return fmt.Errorf("failed to create prompt.md: %w", err)
	}

	fmt.Printf("Initialized Skylark project in %s\n", projectDir)
	return nil
}

// Watch starts watching for file changes
func (c *CLI) Watch(args []string) error {
	// Parse timeout flag
	var timeout time.Duration
	if len(args) > 0 && args[0] == "--timeout" {
		if len(args) < 2 {
			return fmt.Errorf("--timeout requires a duration (e.g., 5s)")
		}
		var err error
		timeout, err = time.ParseDuration(args[1])
		if err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}
		args = args[2:]
	}

	// Load configuration
	if err := c.loadConfig(); err != nil {
		return err
	}

	c.logger.Info("starting watch command",
		"timeout", timeout)

	// Create processor
	proc, err := concrete.NewProcessor(c.config.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create processor: %w", err)
	}

	// Create worker pool
	cfg := c.config.GetConfig()
	c.logger.Debug("creating worker pool",
		"worker_count", cfg.Workers.Count,
		"queue_size", cfg.Workers.QueueSize)

	pool, err := wkconcrete.NewPool(worker.Options{
		Config:    c.config,
		Logger:    c.logger,
		ProcMgr:   proc.GetProcessManager(),
		QueueSize: cfg.Workers.QueueSize,
		Workers:   cfg.Workers.Count,
	})
	if err != nil {
		return fmt.Errorf("failed to create worker pool: %w", err)
	}
	defer pool.Stop()

	// Create channels
	jobQueue := make(chan job.Job, cfg.Workers.QueueSize)
	done := make(chan struct{})
	progressDone := make(chan struct{})
	sigChan := make(chan os.Signal, 1)

	// Start components
	c.logger.Debug("creating file watcher")
	watcher, err := wconcrete.NewWatcher(cfg, jobQueue, proc)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Start worker pool consumer
	go func() {
		defer close(done)
		for j := range jobQueue {
			pool.Queue() <- j
		}
	}()

	// Start progress monitoring
	go c.monitorProgress(pool, progressDone)

	// Show initial message
	fmt.Println("Watching for changes...")

	// Wait for interrupt or timeout
	signal.Notify(sigChan, os.Interrupt)

	if timeout > 0 {
		// Use timeout if specified
		select {
		case <-sigChan:
			c.logger.Info("received interrupt")
		case <-time.After(timeout):
			c.logger.Info("timeout reached", "duration", timeout)
		}
	} else {
		// Wait indefinitely
		<-sigChan
		c.logger.Info("received interrupt")
	}

	// Cleanup in reverse order of creation
	c.logger.Info("shutting down")

	// 1. Stop accepting new events
	watcher.Stop()
	c.logger.Debug("stopped file watcher")

	// 2. Stop accepting new jobs
	close(jobQueue)
	c.logger.Debug("closed job queue")

	// 3. Wait for worker to finish
	<-done
	c.logger.Debug("worker pool drained")

	// 4. Stop progress monitoring
	close(progressDone)
	c.logger.Debug("stopped progress monitoring")

	// Final stats
	stats := pool.Stats()
	c.logger.Info("final status",
		"processed", stats.ProcessedJobs(),
		"failed", stats.FailedJobs(),
		"queued", stats.QueuedJobs())

	return nil
}

// RunOnce processes files once without watching
func (c *CLI) RunOnce(args []string) error {
	// Load configuration
	if err := c.loadConfig(); err != nil {
		return err
	}

	c.logger.Info("starting run command")

	// Create processor
	proc, err := concrete.NewProcessor(c.config.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create processor: %w", err)
	}

	// Create worker pool
	cfg := c.config.GetConfig()
	c.logger.Debug("creating worker pool",
		"worker_count", cfg.Workers.Count,
		"queue_size", cfg.Workers.QueueSize)

	pool, err := wkconcrete.NewPool(worker.Options{
		Config:    c.config,
		Logger:    c.logger,
		ProcMgr:   proc.GetProcessManager(),
		QueueSize: cfg.Workers.QueueSize,
		Workers:   cfg.Workers.Count,
	})
	if err != nil {
		return fmt.Errorf("failed to create worker pool: %w", err)
	}
	defer pool.Stop()

	// Track progress
	done := make(chan struct{})
	go c.monitorProgress(pool, done)

	// Queue files for processing
	fileCount := 0
	c.logger.Debug("scanning for markdown files")

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip .skai directory and non-markdown files
		if info.IsDir() {
			if filepath.Base(path) == ".skai" {
				return filepath.SkipDir // Skip the entire .skai directory
			}
			return nil
		}
		if filepath.Ext(path) == ".md" {
			c.logger.Debug("queueing file", "path", path)
			pool.Queue() <- job.NewFileChangeJob(path, proc)
			fileCount++
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Show initial count
	c.logger.Info("starting processing",
		"file_count", fileCount)
	fmt.Printf("Processing %d files...\n", fileCount)

	// Wait for all jobs to complete
	for {
		stats := pool.Stats()
		if stats.ProcessedJobs()+stats.FailedJobs() == uint64(fileCount) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Signal progress monitor to stop
	close(done)

	// Get final status
	stats := pool.Stats()
	c.logger.Info("processing complete",
		"processed", stats.ProcessedJobs(),
		"failed", stats.FailedJobs(),
		"total", fileCount)

	if stats.FailedJobs() > 0 {
		return fmt.Errorf("%d/%d files failed processing", stats.FailedJobs(), fileCount)
	}

	fmt.Printf("\nSuccessfully processed %d files\n", stats.ProcessedJobs())
	return nil
}

// monitorProgress displays progress information
func (c *CLI) monitorProgress(pool worker.Pool, done chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastStats worker.Stats
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			stats := pool.Stats()
			if stats != lastStats {
				c.logger.Debug("progress update",
					"processed", stats.ProcessedJobs(),
					"failed", stats.FailedJobs(),
					"queued", stats.QueuedJobs())
				lastStats = stats
			}
			fmt.Printf("\rProcessed: %d, Failed: %d, Queued: %d",
				stats.ProcessedJobs(),
				stats.FailedJobs(),
				stats.QueuedJobs())
		}
	}
}

// Version displays version information
func (c *CLI) Version(args []string) error {
	fmt.Printf("Skylark version %s\n", Version)
	return nil
}

// loadConfig loads and validates the configuration
func (c *CLI) loadConfig() error {
	// Find .skai directory
	dir, err := findSkaiDir()
	if err != nil {
		return err
	}

	// Load configuration
	c.config = config.NewManager(dir)
	if err := c.config.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	return nil
}

// findSkaiDir finds the nearest .skai directory
func findSkaiDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".skai")); err == nil {
			return filepath.Join(dir, ".skai"), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(".skai directory not found")
		}
		dir = parent
	}
}
