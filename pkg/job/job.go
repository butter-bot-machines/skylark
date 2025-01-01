package job

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
	Path    string
	Content []byte
}

func (j *FileChangeJob) Process() error {
	// Process file change (to be implemented by consumers)
	return nil
}

func (j *FileChangeJob) OnFailure(err error) {
	// Handle failure (to be implemented by consumers)
}

func (j *FileChangeJob) MaxRetries() int {
	return 3
}
