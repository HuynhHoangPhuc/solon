// notify.go: Provider registry and shared types for the notify package.
package notify

// Provider defines a notification channel (Telegram, Discord, Slack, etc.).
type Provider struct {
	Name      string
	IsEnabled func(env map[string]string) bool
	Send      func(input map[string]interface{}, env map[string]string) (*SendResult, error)
}

// Providers is the ordered list of all registered notification providers.
var Providers = []Provider{Telegram, Discord, Slack}

// strField safely retrieves a string value from an untyped map.
// Returns fallback if the key is absent or the value is not a string.
func strField(m map[string]interface{}, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}
