package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "os"
    "time"
)

// Input represents the tool's input format
type Input struct {
    Format string `json:"format,omitempty"` // Optional format string
}

// Output represents the tool's output format
type Output struct {
    DateTime string `json:"datetime"` // RFC3339 formatted time
}

func main() {
    usage := flag.Bool("usage", false, "Display usage schema")
    health := flag.Bool("health", false, "Check tool health")
    flag.Parse()

    if *usage {
        schema := map[string]interface{}{
            "schema": map[string]interface{}{
                "name": "currentdatetime",
                "description": "Returns current date and time in RFC3339 format",
                "parameters": map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "format": map[string]interface{}{
                            "type": "string",
                            "description": "Optional time format string (defaults to RFC3339)",
                        },
                    },
                    "additionalProperties": false,
                },
            },
            "env": map[string]interface{}{},
        }
        json.NewEncoder(os.Stdout).Encode(schema)
        return
    }

    if *health {
        health := map[string]interface{}{
            "status": true, // this is the only required field, but others can be included
        }
        json.NewEncoder(os.Stdout).Encode(health)
        return
    }

    // Read input
    input, err := io.ReadAll(os.Stdin)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
        os.Exit(1)
    }

    // Parse input
    var params Input
    if len(input) > 0 {
        if err := json.Unmarshal(input, &params); err != nil {
            fmt.Fprintf(os.Stderr, "Invalid input format: %v\n", err)
            os.Exit(1)
        }
    }

    // Get current time
    now := time.Now()
    format := time.RFC3339
    if params.Format != "" {
        format = params.Format
    }

    // Format output
    output := Output{
        DateTime: now.Format(format),
    }

    // Write JSON response
    if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to encode output: %v\n", err)
        os.Exit(1)
    }
}
