package job

import (
	"fmt"
	"log/slog"

	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/processor"
)

// Job represents a unit of work that can be processed
type Job interface {
	// Process executes the job
	Process() error

	// OnFailure handles job failure
	OnFailure(error)

	// MaxRetries returns the maximum number of retry attempts
	MaxRetries() int
}

// FileChangeJob represents a file change event
type FileChangeJob struct {
	Path      string              // Path to the file to process
	Processor *processor.Processor // Processor instance to use
	logger    *slog.Logger        // Logger for this job
}

// NewFileChangeJob creates a new file change job
func NewFileChangeJob(path string, proc *processor.Processor) *FileChangeJob {
	return &FileChangeJob{
		Path:      path,
		Processor: proc,
		logger:    logging.NewLogger(&logging.Options{Level: slog.LevelDebug}),
	}
}

func (j *FileChangeJob) Process() error {
	j.logger.Debug("processing file",
		"path", j.Path)

	// Process file using processor
	if err := j.Processor.ProcessFile(j.Path); err != nil {
		j.logger.Error("processing failed",
			"path", j.Path,
			"error", err)
		return fmt.Errorf("failed to process file %s: %w", j.Path, err)
	}

	j.logger.Debug("file processed successfully",
		"path", j.Path)
	return nil
}

func (j *FileChangeJob) OnFailure(err error) {
	j.logger.Error("job failed",
		"path", j.Path,
		"error", err,
		"retries_remaining", j.MaxRetries())
}

func (j *FileChangeJob) MaxRetries() int {
	return 3
}
