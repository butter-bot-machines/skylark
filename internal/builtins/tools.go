package builtins

import (
	"embed"
	"fmt"
)

//go:embed tools/currentdatetime/main.go
var Tools embed.FS

// GetToolSource returns the source code for a builtin tool
func GetToolSource(name string) ([]byte, error) {
	return Tools.ReadFile(fmt.Sprintf("tools/%s/main.go", name))
}
