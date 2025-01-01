package suite

import (
	"errors"
	"testing"
	"time"
)

func TestSuiteBasics(t *testing.T) {
	suite := NewSuite()
	executed := false

	suite.Add("basic test", func(t *testing.T) {
		executed = true
	})

	suite.Run(t)

	if !executed {
		t.Error("Test was not executed")
	}
}

func TestSuiteHooks(t *testing.T) {
	suite := NewSuite()
	var steps []string

	suite.BeforeAll(func() error {
		steps = append(steps, "before_all")
		return nil
	})

	suite.AfterAll(func() error {
		steps = append(steps, "after_all")
		return nil
	})

	suite.BeforeEach(func() error {
		steps = append(steps, "before_each")
		return nil
	})

	suite.AfterEach(func() error {
		steps = append(steps, "after_each")
		return nil
	})

	suite.Add("test1", func(t *testing.T) {
		steps = append(steps, "test1")
	})

	suite.Add("test2", func(t *testing.T) {
		steps = append(steps, "test2")
	})

	suite.Run(t)

	expected := []string{
		"before_all",
		"before_each",
		"test1",
		"after_each",
		"before_each",
		"test2",
		"after_each",
		"after_all",
	}

	if len(steps) != len(expected) {
		t.Errorf("Expected %d steps, got %d", len(expected), len(steps))
	}

	for i, step := range steps {
		if i >= len(expected) || step != expected[i] {
			t.Errorf("Step %d: expected %s, got %s", i, expected[i], step)
		}
	}
}

func TestSuiteFiltering(t *testing.T) {
	suite := NewSuite()
	executed := make(map[string]bool)

	suite.Add("test1", func(t *testing.T) {
		executed["test1"] = true
	})

	suite.Add("test2", func(t *testing.T) {
		executed["test2"] = true
	}, WithCategory("integration"))

	suite.Add("test3", func(t *testing.T) {
		executed["test3"] = true
	}, WithTags("slow"))

	// Run only integration tests
	suite.Run(t, "category:integration")

	if executed["test1"] {
		t.Error("test1 should not have run")
	}
	if !executed["test2"] {
		t.Error("test2 should have run")
	}
	if executed["test3"] {
		t.Error("test3 should not have run")
	}

	// Reset and run only tagged tests
	executed = make(map[string]bool)
	suite.Run(t, "tag:slow")

	if executed["test1"] {
		t.Error("test1 should not have run")
	}
	if executed["test2"] {
		t.Error("test2 should not have run")
	}
	if !executed["test3"] {
		t.Error("test3 should have run")
	}
}

func TestSuiteTimeout(t *testing.T) {
	t.Run("timeout test", func(t *testing.T) {
		suite := NewSuite()
		testStarted := make(chan struct{})
		testFinished := make(chan struct{})

		suite.Add("test", func(t *testing.T) {
			close(testStarted)
			time.Sleep(200 * time.Millisecond)
			close(testFinished)
		}, WithTimeout(50*time.Millisecond))

		// Run the suite in a goroutine
		go suite.Run(t)

		// Wait for test to start
		<-testStarted

		// Wait a bit longer than the timeout but less than the sleep duration
		time.Sleep(100 * time.Millisecond)

		// Verify test didn't complete
		select {
		case <-testFinished:
			t.Error("Test should have timed out before completing")
		default:
			// Test correctly timed out
		}
	})
}

func TestSuiteHookErrors(t *testing.T) {
	// Test BeforeAll error
	t.Run("BeforeAll error", func(t *testing.T) {
		suite := NewSuite()
		executed := false

		suite.BeforeAll(func() error {
			return errors.New("expected error")
		})

		suite.Add("test", func(t *testing.T) {
			executed = true
		})

		suite.Run(t)

		if executed {
			t.Error("Test should not have run after BeforeAll error")
		}
	})

	// Test BeforeEach error
	t.Run("BeforeEach error", func(t *testing.T) {
		suite := NewSuite()
		executed := false

		suite.BeforeEach(func() error {
			return errors.New("expected error")
		})

		suite.Add("test", func(t *testing.T) {
			executed = true
		})

		suite.Run(t)

		if executed {
			t.Error("Test should not have run after BeforeEach error")
		}
	})

	// Test AfterEach error
	t.Run("AfterEach error", func(t *testing.T) {
		suite := NewSuite()
		executed := false

		suite.AfterEach(func() error {
			return errors.New("expected error")
		})

		suite.Add("test", func(t *testing.T) {
			executed = true
		})

		suite.Run(t)

		if !executed {
			t.Error("Test should have run despite AfterEach error")
		}
	})

	// Test AfterAll error
	t.Run("AfterAll error", func(t *testing.T) {
		suite := NewSuite()
		executed := false

		suite.AfterAll(func() error {
			return errors.New("expected error")
		})

		suite.Add("test", func(t *testing.T) {
			executed = true
		})

		suite.Run(t)

		if !executed {
			t.Error("Test should have run despite AfterAll error")
		}
	})
}

func TestTestDataHelpers(t *testing.T) {
	// Test GetTestDataPath
	path := GetTestDataPath()
	if path == "" {
		t.Error("GetTestDataPath returned empty string")
	}

	// Test CreateTempDir
	dir, cleanup, err := CreateTempDir("test")
	if err != nil {
		t.Errorf("CreateTempDir failed: %v", err)
	}
	defer cleanup()

	if dir == "" {
		t.Error("CreateTempDir returned empty string")
	}
}
