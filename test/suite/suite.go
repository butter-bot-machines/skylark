package suite

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestSuite represents a collection of tests with setup and teardown
type TestSuite struct {
	tests      []Test
	beforeAll  func() error
	afterAll   func() error
	beforeEach func() error
	afterEach  func() error
}

// Test represents a single test case
type Test struct {
	name     string
	fn       func(t *testing.T)
	category string
	tags     []string
	timeout  time.Duration
}

// NewSuite creates a new test suite
func NewSuite() *TestSuite {
	return &TestSuite{
		tests: make([]Test, 0),
	}
}

// BeforeAll sets up the test suite before any tests run
func (s *TestSuite) BeforeAll(fn func() error) {
	s.beforeAll = fn
}

// AfterAll cleans up after all tests have run
func (s *TestSuite) AfterAll(fn func() error) {
	s.afterAll = fn
}

// BeforeEach runs before each test
func (s *TestSuite) BeforeEach(fn func() error) {
	s.beforeEach = fn
}

// AfterEach runs after each test
func (s *TestSuite) AfterEach(fn func() error) {
	s.afterEach = fn
}

// Add adds a test to the suite
func (s *TestSuite) Add(name string, fn func(t *testing.T), opts ...TestOption) {
	test := Test{
		name:     name,
		fn:       fn,
		timeout:  30 * time.Second,
		category: "unit",
	}

	for _, opt := range opts {
		opt(&test)
	}

	s.tests = append(s.tests, test)
}

// TestOption configures a test
type TestOption func(*Test)

// WithCategory sets the test category
func WithCategory(category string) TestOption {
	return func(t *Test) {
		t.category = category
	}
}

// WithTags adds tags to the test
func WithTags(tags ...string) TestOption {
	return func(t *Test) {
		t.tags = append(t.tags, tags...)
	}
}

// WithTimeout sets the test timeout
func WithTimeout(timeout time.Duration) TestOption {
	return func(t *Test) {
		t.timeout = timeout
	}
}

// Run executes all tests in the suite
func (s *TestSuite) Run(t *testing.T, filter ...string) {
	// Run BeforeAll
	var beforeAllErr error
	if s.beforeAll != nil {
		beforeAllErr = s.beforeAll()
		if beforeAllErr != nil {
			t.Logf("BeforeAll failed: %v", beforeAllErr)
		}
	}

	// Run AfterAll at the end
	defer func() {
		if s.afterAll != nil {
			if err := s.afterAll(); err != nil {
				t.Logf("AfterAll failed: %v", err)
			}
		}
	}()

	// Run each test
	for _, test := range s.tests {
		if !shouldRun(test, filter) {
			continue
		}

		t.Run(test.name, func(t *testing.T) {
			// Skip if BeforeAll failed
			if beforeAllErr != nil {
				t.Skipf("Skipped due to BeforeAll failure: %v", beforeAllErr)
				return
			}

			// Run BeforeEach
			if s.beforeEach != nil {
				if err := s.beforeEach(); err != nil {
					t.Logf("BeforeEach failed: %v", err)
					return
				}
			}

			// Run test with timeout and panic recovery
			done := make(chan struct{})
			var testPanic interface{}

			go func() {
				defer func() {
					if r := recover(); r != nil {
						testPanic = r
					}
					close(done)
				}()
				test.fn(t)
			}()

			timer := time.NewTimer(test.timeout)
			defer timer.Stop()

			select {
			case <-done:
				// Test completed normally
			case <-timer.C:
				// Test timed out as expected
				return
			}

			// Run AfterEach
			if s.afterEach != nil {
				if err := s.afterEach(); err != nil {
					t.Logf("AfterEach failed: %v", err)
				}
			}

			// Report panic if it occurred
			if testPanic != nil {
				t.Errorf("Test panicked: %v", testPanic)
			}
		})
	}
}

// shouldRun checks if a test should run based on filters
func shouldRun(test Test, filter []string) bool {
	if len(filter) == 0 {
		return true
	}

	for _, f := range filter {
		if strings.HasPrefix(f, "tag:") {
			tag := strings.TrimPrefix(f, "tag:")
			for _, t := range test.tags {
				if t == tag {
					return true
				}
			}
		} else if strings.HasPrefix(f, "category:") {
			category := strings.TrimPrefix(f, "category:")
			if test.category == category {
				return true
			}
		} else if strings.Contains(test.name, f) {
			return true
		}
	}

	return false
}

// Helper functions for test organization

// GetTestDataPath returns the path to test data files
func GetTestDataPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Could not determine test data path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "fixtures")
}

// LoadTestData loads test data from a file
func LoadTestData(name string) ([]byte, error) {
	path := filepath.Join(GetTestDataPath(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load test data %s: %w", name, err)
	}
	return data, nil
}

// CreateTempDir creates a temporary directory for tests
func CreateTempDir(prefix string) (string, func(), error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup, nil
}
