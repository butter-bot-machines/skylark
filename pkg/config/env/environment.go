package env

import (
	"os"
	"strconv"
	"time"
)

// Environment implements config.Environment for accessing environment variables
type Environment struct {
	// No state needed, just methods for accessing os.Getenv
}

// New creates a new environment accessor
func New() *Environment {
	return &Environment{}
}

// GetString returns an environment variable as a string
func (e *Environment) GetString(key string) string {
	return os.Getenv(key)
}

// GetInt returns an environment variable as an integer
func (e *Environment) GetInt(key string) int {
	str := os.Getenv(key)
	if str == "" {
		return 0
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return val
}

// GetBool returns an environment variable as a boolean
func (e *Environment) GetBool(key string) bool {
	str := os.Getenv(key)
	if str == "" {
		return false
	}

	val, err := strconv.ParseBool(str)
	if err != nil {
		return false
	}
	return val
}

// GetDuration returns an environment variable as a duration
func (e *Environment) GetDuration(key string) time.Duration {
	str := os.Getenv(key)
	if str == "" {
		return 0
	}

	val, err := time.ParseDuration(str)
	if err != nil {
		return 0
	}
	return val
}

// GetStringWithDefault returns an environment variable as a string with a default value
func (e *Environment) GetStringWithDefault(key string, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// GetIntWithDefault returns an environment variable as an integer with a default value
func (e *Environment) GetIntWithDefault(key string, defaultValue int) int {
	str := os.Getenv(key)
	if str == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetBoolWithDefault returns an environment variable as a boolean with a default value
func (e *Environment) GetBoolWithDefault(key string, defaultValue bool) bool {
	str := os.Getenv(key)
	if str == "" {
		return defaultValue
	}

	val, err := strconv.ParseBool(str)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetDurationWithDefault returns an environment variable as a duration with a default value
func (e *Environment) GetDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	str := os.Getenv(key)
	if str == "" {
		return defaultValue
	}

	val, err := time.ParseDuration(str)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetRequired returns an environment variable or panics if it's not set
func (e *Environment) GetRequired(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic("required environment variable not set: " + key)
	}
	return val
}

// Has returns true if an environment variable is set
func (e *Environment) Has(key string) bool {
	_, exists := os.LookupEnv(key)
	return exists
}
