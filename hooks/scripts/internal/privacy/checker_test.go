package privacy

import (
	"testing"
)

// TestIsSafeFile tests the IsSafeFile function.
func TestIsSafeFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"allows .env.example", ".env.example", true},
		{"allows .env.sample", ".env.sample", true},
		{"allows .env.template", ".env.template", true},
		{"allows path/to/.env.example", "path/to/.env.example", true},
		{"blocks .env (not safe)", ".env", false},
		{"blocks credentials.json", "credentials.json", false},
		{"empty string", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSafeFile(tt.path)
			if got != tt.expected {
				t.Errorf("IsSafeFile(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

// TestIsPrivacySensitive tests the IsPrivacySensitive function.
func TestIsPrivacySensitive(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"detects .env", ".env", true},
		{"detects .env.local", ".env.local", true},
		{"detects .env.production", ".env.production", true},
		{"detects credentials.json", "credentials.json", true},
		{"detects secrets.yaml", "secrets.yaml", true},
		{"detects secrets.yml", "secrets.yml", true},
		{"detects private.pem", "private.pem", true},
		{"detects id_rsa", "id_rsa", true},
		{"detects id_ed25519", "id_ed25519", true},
		{"allows .env.example (safe file)", ".env.example", false},
		{"allows src/main.ts", "src/main.ts", false},
		{"allows package.json", "package.json", false},
		{"empty string", "", false},
		{"detects secret.key", "secret.key", true},
		{"ignores case for credentials", "CREDENTIALS.json", true},
		{"ignores case for secrets", "SECRETS.YAML", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPrivacySensitive(tt.path)
			if got != tt.expected {
				t.Errorf("IsPrivacySensitive(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

// TestHasApprovalPrefix tests prefix detection and stripping.
func TestHasApprovalPrefix(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"detects APPROVED: prefix", "APPROVED:.env", true},
		{"rejects missing prefix", ".env", false},
		{"case sensitive", "approved:.env", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasApprovalPrefix(tt.path)
			if got != tt.expected {
				t.Errorf("HasApprovalPrefix(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

// TestStripApprovalPrefix tests the StripApprovalPrefix function.
func TestStripApprovalPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips prefix", "APPROVED:.env", ".env"},
		{"returns unchanged if no prefix", ".env", ".env"},
		{"strips complex path", "APPROVED:.env.local", ".env.local"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripApprovalPrefix(tt.input)
			if got != tt.expected {
				t.Errorf("StripApprovalPrefix(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestExtractPaths tests path extraction from tool inputs.
func TestExtractPaths(t *testing.T) {
	tests := []struct {
		name           string
		toolInput      map[string]interface{}
		shouldContain  []string
		shouldNotEmpty bool
	}{
		{
			"extracts file_path",
			map[string]interface{}{"file_path": ".env"},
			[]string{".env"},
			true,
		},
		{
			"extracts path field",
			map[string]interface{}{"path": "credentials.json"},
			[]string{"credentials.json"},
			true,
		},
		{
			"extracts .env from bash command",
			map[string]interface{}{"command": "cat .env"},
			[]string{".env"},
			true,
		},
		{
			"extracts APPROVED: path from bash command",
			map[string]interface{}{"command": "cat APPROVED:.env"},
			[]string{"APPROVED:.env"},
			true,
		},
		{
			"extracts env var assignment",
			map[string]interface{}{"command": "FILE=.env.local cat $FILE"},
			[]string{".env.local"},
			true,
		},
		{
			"returns entry for no sensitive paths",
			map[string]interface{}{"file_path": "src/main.ts"},
			[]string{"src/main.ts"},
			true,
		},
		{
			"handles nil input",
			nil,
			[]string{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPaths(tt.toolInput)
			if tt.shouldNotEmpty && len(got) == 0 {
				t.Errorf("ExtractPaths(%v) returned empty, expected non-empty", tt.toolInput)
			}
			if !tt.shouldNotEmpty && len(got) != 0 {
				t.Errorf("ExtractPaths(%v) returned %v, expected empty", tt.toolInput, got)
			}
			for _, expected := range tt.shouldContain {
				found := false
				for _, entry := range got {
					if entry.Value == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractPaths(%v) missing %q in result %v", tt.toolInput, expected, got)
				}
			}
		})
	}
}

// TestCheckPrivacy tests the main privacy check function.
func TestCheckPrivacy(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		toolInput  map[string]interface{}
		opts       PrivacyOpts
		wantBlocked bool
		wantApproved bool
		wantIsBash bool
	}{
		{
			"blocks .env read without approval",
			"Read",
			map[string]interface{}{"file_path": ".env"},
			PrivacyOpts{},
			true,
			false,
			false,
		},
		{
			"allows APPROVED:.env",
			"Read",
			map[string]interface{}{"file_path": "APPROVED:.env"},
			PrivacyOpts{},
			false,
			true,
			false,
		},
		{
			"blocks Bash accessing .env by default",
			"Bash",
			map[string]interface{}{"command": "cat .env"},
			PrivacyOpts{AllowBash: false},
			true,
			false,
			false,
		},
		{
			"allows .env.example",
			"Read",
			map[string]interface{}{"file_path": ".env.example"},
			PrivacyOpts{},
			false,
			false,
			false,
		},
		{
			"allows normal file",
			"Read",
			map[string]interface{}{"file_path": "src/main.ts"},
			PrivacyOpts{},
			false,
			false,
			false,
		},
		{
			"respects disabled option",
			"Read",
			map[string]interface{}{"file_path": ".env"},
			PrivacyOpts{Disabled: true},
			false,
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPrivacy(tt.toolName, tt.toolInput, tt.opts)
			if result.Blocked != tt.wantBlocked {
				t.Errorf("CheckPrivacy.Blocked = %v, want %v", result.Blocked, tt.wantBlocked)
			}
			if result.Approved != tt.wantApproved {
				t.Errorf("CheckPrivacy.Approved = %v, want %v", result.Approved, tt.wantApproved)
			}
			if result.IsBash != tt.wantIsBash {
				t.Errorf("CheckPrivacy.IsBash = %v, want %v", result.IsBash, tt.wantIsBash)
			}
		})
	}
}

// TestIsSuspiciousPath tests suspicious path detection.
func TestIsSuspiciousPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"detects traversal", "../etc/passwd", true},
		{"detects absolute path", "/etc/passwd", true},
		{"allows relative path", "src/main.ts", false},
		{"allows simple path", ".env", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSuspiciousPath(tt.path)
			if got != tt.expected {
				t.Errorf("IsSuspiciousPath(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

// TestBuildPromptData tests prompt data construction.
func TestBuildPromptData(t *testing.T) {
	result := BuildPromptData(".env")
	if result["type"] != "PRIVACY_PROMPT" {
		t.Errorf("BuildPromptData type = %v, want PRIVACY_PROMPT", result["type"])
	}
	if result["file"] != ".env" {
		t.Errorf("BuildPromptData file = %v, want .env", result["file"])
	}
	if result["basename"] != ".env" {
		t.Errorf("BuildPromptData basename = %v, want .env", result["basename"])
	}
	if q, ok := result["question"].(map[string]interface{}); !ok || q["header"] != "File Access" {
		t.Errorf("BuildPromptData question = %v, want File Access header", result["question"])
	}
}
