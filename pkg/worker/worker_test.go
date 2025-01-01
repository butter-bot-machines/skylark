package worker

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	pool := NewPool(2, 10, 3)
	defer pool.Stop()

	// Test successful task
	t.Run("successful task", func(t *testing.T) {
		task := &Task{
			ID: "task1",
			Execute: func() (any, error) {
				return "success", nil
			},
		}

		if err := pool.Submit(task); err != nil {
			t.Errorf("Submit() error = %v", err)
			return
		}

		result := <-task.Result
		if result.Error != nil {
			t.Errorf("Task execution error = %v", result.Error)
		}
		if result.Output != "success" {
			t.Errorf("Task output = %v, want %v", result.Output, "success")
		}
	})

	// Test task with error and retry
	t.Run("task with retry", func(t *testing.T) {
		attempts := 0
		task := &Task{
			ID:         "task2",
			RetryCount: 2,
			Execute: func() (any, error) {
				attempts++
				if attempts < 2 {
					return nil, errors.New("temporary error")
				}
				return "retry success", nil
			},
		}

		if err := pool.Submit(task); err != nil {
			t.Errorf("Submit() error = %v", err)
			return
		}

		result := <-task.Result
		if result.Error != nil {
			t.Errorf("Task execution error = %v", result.Error)
		}
		if attempts != 2 {
			t.Errorf("Task attempts = %v, want %v", attempts, 2)
		}
	})

	// Test task timeout
	t.Run("task timeout", func(t *testing.T) {
		task := &Task{
			ID:      "task3",
			Timeout: 100 * time.Millisecond,
			Execute: func() (any, error) {
				time.Sleep(200 * time.Millisecond)
				return nil, nil
			},
		}

		if err := pool.Submit(task); err != nil {
			t.Errorf("Submit() error = %v", err)
			return
		}

		result := <-task.Result
		if result.Error == nil {
			t.Error("Expected timeout error")
		}
	})

	// Test task dependencies
	t.Run("task dependencies", func(t *testing.T) {
		// First task
		task1 := &Task{
			ID: "dep1",
			Execute: func() (any, error) {
				return "dep1 done", nil
			},
		}

		if err := pool.Submit(task1); err != nil {
			t.Errorf("Submit() error = %v", err)
			return
		}

		<-task1.Result

		// Dependent task
		task2 := &Task{
			ID:        "dep2",
			DependsOn: []string{"dep1"},
			Execute: func() (any, error) {
				return "dep2 done", nil
			},
		}

		if err := pool.Submit(task2); err != nil {
			t.Errorf("Submit() error = %v", err)
			return
		}

		result := <-task2.Result
		if result.Error != nil {
			t.Errorf("Task execution error = %v", result.Error)
		}

		// Task with missing dependency
		task3 := &Task{
			ID:        "dep3",
			DependsOn: []string{"missing"},
			Execute: func() (any, error) {
				return nil, nil
			},
		}

		if err := pool.Submit(task3); err == nil {
			t.Error("Expected error for missing dependency")
		}
	})

	// Test concurrent task execution
	t.Run("concurrent tasks", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]TaskResult, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				task := &Task{
					ID: fmt.Sprintf("concurrent%d", i),
					Execute: func() (any, error) {
						time.Sleep(100 * time.Millisecond)
						return i, nil
					},
				}

				if err := pool.Submit(task); err != nil {
					t.Errorf("Submit() error = %v", err)
					return
				}

				results[i] = <-task.Result
			}(i)
		}

		wg.Wait()

		for i, result := range results {
			if result.Error != nil {
				t.Errorf("Task %d error = %v", i, result.Error)
			}
			if result.Output != i {
				t.Errorf("Task %d output = %v, want %v", i, result.Output, i)
			}
		}
	})

	// Test worker metrics
	t.Run("worker metrics", func(t *testing.T) {
		// Reset metrics with some tasks
		for i := 0; i < 3; i++ {
			task := &Task{
				ID: fmt.Sprintf("metric%d", i),
				Execute: func() (any, error) {
					return nil, nil
				},
			}

			if err := pool.Submit(task); err != nil {
				t.Errorf("Submit() error = %v", err)
				return
			}

			<-task.Result
		}

		// Check metrics
		for _, w := range pool.workers {
			w.metrics.mu.Lock()
			if w.metrics.TasksProcessed == 0 {
				t.Error("Expected worker to process tasks")
			}
			if w.metrics.TotalTime == 0 {
				t.Error("Expected worker to track time")
			}
			w.metrics.mu.Unlock()
		}
	})

	// Test pool shutdown
	t.Run("pool shutdown", func(t *testing.T) {
		pool := NewPool(1, 5, 1)
		
		// Submit a task
		task := &Task{
			ID: "shutdown",
			Execute: func() (any, error) {
				return "done", nil
			},
		}

		if err := pool.Submit(task); err != nil {
			t.Errorf("Submit() error = %v", err)
			return
		}

		// Wait for task to complete
		<-task.Result

		// Stop the pool
		pool.Stop()

		// Try to submit after shutdown
		if err := pool.Submit(task); err == nil {
			t.Error("Expected error submitting to stopped pool")
		}
	})
}

func TestTaskValidation(t *testing.T) {
	pool := NewPool(1, 5, 1)
	defer pool.Stop()

	tests := []struct {
		name    string
		task    *Task
		wantErr bool
	}{
		{
			name:    "missing ID",
			task:    &Task{Execute: func() (any, error) { return nil, nil }},
			wantErr: true,
		},
		{
			name:    "missing execute function",
			task:    &Task{ID: "test"},
			wantErr: true,
		},
		{
			name: "valid task",
			task: &Task{
				ID:      "test",
				Execute: func() (any, error) { return nil, nil },
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := pool.Submit(tt.task); (err != nil) != tt.wantErr {
				t.Errorf("Submit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskQueue(t *testing.T) {
	// Create pool with small queue
	pool := NewPool(1, 2, 1)
	defer pool.Stop()

	// Fill queue
	for i := 0; i < 2; i++ {
		task := &Task{
			ID: fmt.Sprintf("queue%d", i),
			Execute: func() (any, error) {
				time.Sleep(100 * time.Millisecond)
				return nil, nil
			},
		}

		if err := pool.Submit(task); err != nil {
			t.Errorf("Submit() error = %v", err)
			return
		}
	}

	// Try to submit to full queue
	task := &Task{
		ID: "overflow",
		Execute: func() (any, error) {
			return nil, nil
		},
	}

	if err := pool.Submit(task); err == nil {
		t.Error("Expected error submitting to full queue")
	}
}
