package env

import (
	"os"
	"testing"
	"time"
)

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Failed to unset environment variable: %v", err)
	}
}

func TestEnvironment_BasicTypes(t *testing.T) {
	env := New()

	// Test string
	t.Run("String", func(t *testing.T) {
		key := "TEST_STRING"
		want := "test value"
		setEnv(t, key, want)
		defer unsetEnv(t, key)

		if got := env.GetString(key); got != want {
			t.Errorf("GetString() = %v, want %v", got, want)
		}
	})

	// Test integer
	t.Run("Integer", func(t *testing.T) {
		key := "TEST_INT"
		want := 42
		setEnv(t, key, "42")
		defer unsetEnv(t, key)

		if got := env.GetInt(key); got != want {
			t.Errorf("GetInt() = %v, want %v", got, want)
		}
	})

	// Test boolean
	t.Run("Boolean", func(t *testing.T) {
		key := "TEST_BOOL"
		setEnv(t, key, "true")
		defer unsetEnv(t, key)

		if got := env.GetBool(key); !got {
			t.Error("GetBool() = false, want true")
		}
	})

	// Test duration
	t.Run("Duration", func(t *testing.T) {
		key := "TEST_DURATION"
		want := 5 * time.Second
		setEnv(t, key, "5s")
		defer unsetEnv(t, key)

		if got := env.GetDuration(key); got != want {
			t.Errorf("GetDuration() = %v, want %v", got, want)
		}
	})
}

func TestEnvironment_DefaultValues(t *testing.T) {
	env := New()

	// Test string default
	t.Run("String Default", func(t *testing.T) {
		key := "TEST_STRING_DEFAULT"
		want := "default"
		unsetEnv(t, key)

		if got := env.GetStringWithDefault(key, want); got != want {
			t.Errorf("GetStringWithDefault() = %v, want %v", got, want)
		}
	})

	// Test integer default
	t.Run("Integer Default", func(t *testing.T) {
		key := "TEST_INT_DEFAULT"
		want := 42
		unsetEnv(t, key)

		if got := env.GetIntWithDefault(key, want); got != want {
			t.Errorf("GetIntWithDefault() = %v, want %v", got, want)
		}
	})

	// Test boolean default
	t.Run("Boolean Default", func(t *testing.T) {
		key := "TEST_BOOL_DEFAULT"
		want := true
		unsetEnv(t, key)

		if got := env.GetBoolWithDefault(key, want); got != want {
			t.Errorf("GetBoolWithDefault() = %v, want %v", got, want)
		}
	})

	// Test duration default
	t.Run("Duration Default", func(t *testing.T) {
		key := "TEST_DURATION_DEFAULT"
		want := 5 * time.Second
		unsetEnv(t, key)

		if got := env.GetDurationWithDefault(key, want); got != want {
			t.Errorf("GetDurationWithDefault() = %v, want %v", got, want)
		}
	})
}

func TestEnvironment_Required(t *testing.T) {
	env := New()

	// Test required variable exists
	t.Run("Required Exists", func(t *testing.T) {
		key := "TEST_REQUIRED"
		want := "value"
		setEnv(t, key, want)
		defer unsetEnv(t, key)

		if got := env.GetRequired(key); got != want {
			t.Errorf("GetRequired() = %v, want %v", got, want)
		}
	})

	// Test required variable missing
	t.Run("Required Missing", func(t *testing.T) {
		key := "TEST_REQUIRED_MISSING"
		unsetEnv(t, key)

		defer func() {
			if r := recover(); r == nil {
				t.Error("GetRequired() should panic on missing required variable")
			}
		}()

		env.GetRequired(key)
	})
}

func TestEnvironment_InvalidValues(t *testing.T) {
	env := New()

	// Test invalid integer
	t.Run("Invalid Integer", func(t *testing.T) {
		key := "TEST_INVALID_INT"
		setEnv(t, key, "not a number")
		defer unsetEnv(t, key)

		if got := env.GetInt(key); got != 0 {
			t.Errorf("GetInt() = %v, want 0", got)
		}
	})

	// Test invalid boolean
	t.Run("Invalid Boolean", func(t *testing.T) {
		key := "TEST_INVALID_BOOL"
		setEnv(t, key, "not a bool")
		defer unsetEnv(t, key)

		if got := env.GetBool(key); got != false {
			t.Errorf("GetBool() = %v, want false", got)
		}
	})

	// Test invalid duration
	t.Run("Invalid Duration", func(t *testing.T) {
		key := "TEST_INVALID_DURATION"
		setEnv(t, key, "not a duration")
		defer unsetEnv(t, key)

		if got := env.GetDuration(key); got != 0 {
			t.Errorf("GetDuration() = %v, want 0", got)
		}
	})
}

func TestEnvironment_MissingValues(t *testing.T) {
	env := New()

	// Test missing string
	t.Run("Missing String", func(t *testing.T) {
		key := "TEST_MISSING_STRING"
		unsetEnv(t, key)

		if got := env.GetString(key); got != "" {
			t.Errorf("GetString() = %v, want empty string", got)
		}
	})

	// Test missing integer
	t.Run("Missing Integer", func(t *testing.T) {
		key := "TEST_MISSING_INT"
		unsetEnv(t, key)

		if got := env.GetInt(key); got != 0 {
			t.Errorf("GetInt() = %v, want 0", got)
		}
	})

	// Test missing boolean
	t.Run("Missing Boolean", func(t *testing.T) {
		key := "TEST_MISSING_BOOL"
		unsetEnv(t, key)

		if got := env.GetBool(key); got != false {
			t.Errorf("GetBool() = %v, want false", got)
		}
	})

	// Test missing duration
	t.Run("Missing Duration", func(t *testing.T) {
		key := "TEST_MISSING_DURATION"
		unsetEnv(t, key)

		if got := env.GetDuration(key); got != 0 {
			t.Errorf("GetDuration() = %v, want 0", got)
		}
	})
}

func TestEnvironment_Has(t *testing.T) {
	env := New()

	// Test variable exists
	t.Run("Variable Exists", func(t *testing.T) {
		key := "TEST_EXISTS"
		setEnv(t, key, "value")
		defer unsetEnv(t, key)

		if !env.Has(key) {
			t.Error("Has() = false, want true")
		}
	})

	// Test variable missing
	t.Run("Variable Missing", func(t *testing.T) {
		key := "TEST_MISSING"
		unsetEnv(t, key)

		if env.Has(key) {
			t.Error("Has() = true, want false")
		}
	})
}
