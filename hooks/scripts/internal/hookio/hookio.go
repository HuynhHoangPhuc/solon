// Hook I/O helpers: stdin reading, stdout writing, error blocking.
// All errors use fail-open (exit 0) except explicit blocks (exit 2).
package hookio

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ReadInput reads JSON from stdin into target (any pointer type).
func ReadInput(target interface{}) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("empty stdin")
	}
	return json.Unmarshal(data, target)
}

// WriteOutput encodes output as JSON and writes to stdout.
func WriteOutput(output interface{}) {
	_ = json.NewEncoder(os.Stdout).Encode(output)
}

// WriteContext writes plain text to stdout (for context injection hooks).
func WriteContext(text string) {
	_, _ = os.Stdout.WriteString(text)
}

// Block writes message to stderr and exits with code 2 (explicit deny).
func Block(message string) {
	_, _ = os.Stderr.WriteString(message)
	os.Exit(2)
}

// Log writes a prefixed message to stderr (non-blocking, visible in hook output).
func Log(hookName, message string) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", hookName, message)
}

// RunHook wraps a hook handler with fail-open error handling.
// Reads JSON from stdin into target, then calls handler. Any error → exit 0.
func RunHook(hookName string, target interface{}, handler func() error) {
	if err := ReadInput(target); err != nil {
		Log(hookName, err.Error())
		os.Exit(0)
	}
	if err := handler(); err != nil {
		Log(hookName, err.Error())
		os.Exit(0)
	}
}
