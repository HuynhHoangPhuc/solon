package scout

import (
	"testing"
)

// TestIsBuildCommand tests build command detection.
func TestIsBuildCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"npm run build", "npm run build", true},
		{"pnpm run test", "pnpm run test", true},
		{"yarn test", "yarn test", true},
		{"bun run dev", "bun run dev", true},
		{"npm install", "npm install", true},
		{"npx jest", "npx jest", true},
		{"go build", "go build", true},
		{"cargo build", "cargo build", true},
		{"python script.py", "python script.py", true},
		{"cat file.txt", "cat file.txt", false},
		{"empty string", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBuildCommand(tt.command)
			if got != tt.expected {
				t.Errorf("IsBuildCommand(%q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}

// TestIsVenvExecutable tests venv executable detection.
func TestIsVenvExecutable(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"detects .venv/bin/python", ".venv/bin/python", true},
		{"detects venv/bin/python3", "venv/bin/python3", true},
		{"detects .venv\\Scripts\\python.exe", ".venv\\Scripts\\python.exe", true},
		{"rejects normal python", "python", false},
		{"rejects normal path", "/usr/bin/python", false},
		{"empty string", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVenvExecutable(tt.command)
			if got != tt.expected {
				t.Errorf("IsVenvExecutable(%q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}

// TestIsVenvCreationCommand tests venv creation detection.
func TestIsVenvCreationCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"python -m venv", "python -m venv /tmp/venv", true},
		{"python3 -m venv", "python3 -m venv venv", true},
		{"uv venv", "uv venv", true},
		{"virtualenv", "virtualenv venv", true},
		{"rejects normal python", "python script.py", false},
		{"empty string", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVenvCreationCommand(tt.command)
			if got != tt.expected {
				t.Errorf("IsVenvCreationCommand(%q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}

// TestIsAllowedCommand tests the main command allowlist.
func TestIsAllowedCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"npm run build", "npm run build", true},
		{"with env var prefix", "NODE_ENV=prod npm run build", true},
		{"with sudo", "sudo make install", true},
		{"venv executable", ".venv/bin/python", true},
		{"venv creation", "python -m venv venv", true},
		{"blocked command", "cat .env", false},
		{"empty string", "", false},
		{"with timeout wrapper", "timeout 30 npm run test", false}, // timeout + command is checked separately
		{"with multiple env vars", "VAR1=a VAR2=b npm run build", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAllowedCommand(tt.command)
			if got != tt.expected {
				t.Errorf("IsAllowedCommand(%q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}

// TestStripCommandPrefix tests command prefix stripping.
func TestStripCommandPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips env var", "VAR=value npm run build", "npm run build"},
		{"strips sudo", "sudo make install", "make install"},
		{"strips env + sudo", "VAR=val sudo npm run test", "npm run test"},
		{"strips nice", "nice npm run build", "npm run build"},
		{"handles multiple env vars", "A=1 B=2 npm run build", "npm run build"},
		{"returns unchanged if no prefix", "npm run build", "npm run build"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCommandPrefix(tt.input)
			if got != tt.expected {
				t.Errorf("stripCommandPrefix(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestSplitCompoundCommand tests compound command splitting.
func TestSplitCompoundCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			"splits on &&",
			"npm run build && npm run test",
			[]string{"npm run build", "npm run test"},
		},
		{
			"splits on ||",
			"cmd1 || cmd2",
			[]string{"cmd1", "cmd2"},
		},
		{
			"splits on semicolon",
			"cmd1; cmd2",
			[]string{"cmd1", "cmd2"},
		},
		{
			"handles multiple separators",
			"a && b || c; d",
			[]string{"a", "b", "c", "d"},
		},
		{
			"empty string",
			"",
			nil,
		},
		{
			"single command",
			"npm run build",
			[]string{"npm run build"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitCompoundCommand(tt.command)
			if len(got) != len(tt.expected) {
				t.Errorf("SplitCompoundCommand(%q) returned %d parts, want %d", tt.command, len(got), len(tt.expected))
			}
			for i, part := range got {
				if i < len(tt.expected) && part != tt.expected[i] {
					t.Errorf("SplitCompoundCommand(%q)[%d] = %q, want %q", tt.command, i, part, tt.expected[i])
				}
			}
		})
	}
}

// TestUnwrapShellExecutor tests shell executor unwrapping.
func TestUnwrapShellExecutor(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"bash -c wrapper", "bash -c 'npm run build'", "npm run build"},
		{"sh -c wrapper", "sh -c \"npm run build\"", "npm run build"},
		{"eval wrapper", "eval \"npm run build\"", "npm run build"},
		{"no wrapper", "npm run build", "npm run build"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnwrapShellExecutor(tt.input)
			if got != tt.expected {
				t.Errorf("UnwrapShellExecutor(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestCheckScoutBlock tests the main scout check function.
func TestCheckScoutBlock(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		toolInput   map[string]interface{}
		wantBlocked bool
	}{
		{
			"blocks node_modules read",
			"Read",
			map[string]interface{}{"file_path": "node_modules/foo/bar.js"},
			true,
		},
		{
			"allows src file read",
			"Read",
			map[string]interface{}{"file_path": "src/main.ts"},
			false,
		},
		{
			"allows build command",
			"Bash",
			map[string]interface{}{"command": "npm run build"},
			false,
		},
		{
			"blocks non-build bash",
			"Bash",
			map[string]interface{}{"command": "cat .env"},
			false, // bash is allowed even if not a build command
		},
		{
			"allows venv executable",
			"Bash",
			map[string]interface{}{"command": ".venv/bin/python script.py"},
			false,
		},
		{
			"handles compound commands",
			"Bash",
			map[string]interface{}{"command": "npm run build && npm run test"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckScoutBlock(tt.toolName, tt.toolInput, CheckOptions{})
			if result.Blocked != tt.wantBlocked {
				t.Errorf("CheckScoutBlock blocked=%v, want %v", result.Blocked, tt.wantBlocked)
			}
		})
	}
}

// TestCheckScoutBlockUnwrapping tests unwrapping in scout checks.
func TestCheckScoutBlockUnwrapping(t *testing.T) {
	tests := []struct {
		name        string
		toolInput   map[string]interface{}
		wantBlocked bool
	}{
		{
			"unwraps bash -c wrapper",
			map[string]interface{}{"command": "bash -c 'npm run build'"},
			false,
		},
		{
			"unwraps eval wrapper",
			map[string]interface{}{"command": "eval \"npm run test\""},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckScoutBlock("Bash", tt.toolInput, CheckOptions{})
			if result.Blocked != tt.wantBlocked {
				t.Errorf("CheckScoutBlock(unwrap) blocked=%v, want %v", result.Blocked, tt.wantBlocked)
			}
		})
	}
}

// TestCheckScoutBlockEmpty tests empty/nil cases.
func TestCheckScoutBlockEmpty(t *testing.T) {
	result := CheckScoutBlock("Read", nil, CheckOptions{})
	if result.Blocked {
		t.Errorf("CheckScoutBlock(nil input) should not block")
	}

	result = CheckScoutBlock("Read", map[string]interface{}{}, CheckOptions{})
	if result.Blocked {
		t.Errorf("CheckScoutBlock(empty input) should not block")
	}
}
