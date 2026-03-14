// notify.go: notify subcommand — dispatches hook events to notification providers.
// Always exits 0 to avoid blocking Claude Code.
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"solon-hooks/internal/notify"
)

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Handle Stop hook notifications",
	RunE:  runNotify,
}

func runNotify(cmd *cobra.Command, args []string) error {
	input := readNotifyStdin()

	cwd := ""
	if v, ok := input["cwd"]; ok {
		if s, ok := v.(string); ok {
			cwd = s
		}
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	env := notify.LoadEnv(cwd)
	results := dispatchNotifications(input, env)

	if len(results) > 0 {
		successful := 0
		for _, r := range results {
			if r.success {
				successful++
			}
		}
		fmt.Fprintf(os.Stderr, "[notify] Summary: %d/%d succeeded\n", successful, len(results))
	}

	// Always exit 0 — never block Claude
	os.Exit(0)
	return nil
}

// notifyResult tracks per-provider send outcome for logging.
type notifyResult struct {
	provider  string
	success   bool
	errMsg    string
	throttled bool
}

// readNotifyStdin reads stdin with a 5-second timeout, parses JSON.
// Returns empty map on timeout, empty input, or parse error.
func readNotifyStdin() map[string]interface{} {
	ch := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(os.Stdin)
		ch <- string(data)
	}()

	var raw string
	select {
	case raw = <-ch:
	case <-time.After(5 * time.Second):
		fmt.Fprintln(os.Stderr, "[notify] Stdin timeout")
		return make(map[string]interface{})
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return make(map[string]interface{})
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		fmt.Fprintf(os.Stderr, "[notify] Invalid JSON input: %s\n", err.Error())
		return make(map[string]interface{})
	}
	return result
}

// dispatchNotifications iterates all providers and sends if enabled.
func dispatchNotifications(input map[string]interface{}, env map[string]string) []notifyResult {
	var results []notifyResult

	for _, provider := range notify.Providers {
		prefix := strings.ToUpper(provider.Name) + "_"
		if !hasProviderEnv(prefix, env) {
			continue
		}
		if !provider.IsEnabled(env) {
			continue
		}

		res, err := provider.Send(input, env)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[notify] %s error: %s\n", provider.Name, err.Error())
			results = append(results, notifyResult{provider: provider.Name, errMsg: err.Error()})
			continue
		}

		if res.Success {
			fmt.Fprintf(os.Stderr, "[notify] %s: sent\n", provider.Name)
		} else if res.Throttled {
			fmt.Fprintf(os.Stderr, "[notify] %s: throttled\n", provider.Name)
		} else {
			fmt.Fprintf(os.Stderr, "[notify] %s: failed - %s\n", provider.Name, res.Error)
		}

		results = append(results, notifyResult{
			provider:  provider.Name,
			success:   res.Success,
			errMsg:    res.Error,
			throttled: res.Throttled,
		})
	}

	return results
}

// hasProviderEnv checks if any env key starts with the given prefix.
func hasProviderEnv(prefix string, env map[string]string) bool {
	for k := range env {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}
