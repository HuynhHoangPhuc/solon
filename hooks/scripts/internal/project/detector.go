// Package project detects project type, package manager, framework, Python, and git info.
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"solon-hooks/internal/exec"
	"solon-hooks/internal/hookio"
)

// ── Python Detection ───────────────────────────────────────────────────────

var unsafePythonChars = regexp.MustCompile(`[;&|` + "`" + `$(){}[\]<>!#*?]`)

func isValidPythonPath(p string) bool {
	if p == "" {
		return false
	}
	if unsafePythonChars.MatchString(p) {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

func getPythonPaths() []string {
	var paths []string
	if v := os.Getenv("PYTHON_PATH"); v != "" {
		paths = append(paths, v)
	}
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		programFilesX86 := os.Getenv("ProgramFiles(x86)")
		if programFilesX86 == "" {
			programFilesX86 = `C:\Program Files (x86)`
		}
		if localAppData != "" {
			paths = append(paths, filepath.Join(localAppData, "Microsoft", "WindowsApps", "python.exe"))
			for _, ver := range []string{"313", "312", "311", "310", "39"} {
				paths = append(paths, filepath.Join(localAppData, "Programs", "Python", "Python"+ver, "python.exe"))
			}
		}
		for _, ver := range []string{"313", "312", "311", "310", "39"} {
			paths = append(paths, filepath.Join(programFiles, "Python"+ver, "python.exe"))
			paths = append(paths, filepath.Join(programFilesX86, "Python"+ver, "python.exe"))
		}
	} else {
		paths = append(paths,
			"/usr/bin/python3", "/usr/local/bin/python3",
			"/opt/homebrew/bin/python3", "/opt/homebrew/bin/python",
			"/usr/bin/python", "/usr/local/bin/python",
		)
	}
	return paths
}

func findPythonBinary() string {
	if runtime.GOOS != "windows" {
		if py3 := exec.ExecSafe("which python3", "", 500); py3 != "" && isValidPythonPath(py3) {
			return py3
		}
		if py := exec.ExecSafe("which python", "", 500); py != "" && isValidPythonPath(py) {
			return py
		}
	} else {
		if where := exec.ExecSafe("where python", "", 500); where != "" {
			first := strings.SplitN(where, "\n", 2)[0]
			first = strings.TrimSpace(first)
			if isValidPythonPath(first) {
				return first
			}
		}
	}
	for _, p := range getPythonPaths() {
		if isValidPythonPath(p) {
			return p
		}
	}
	return ""
}

// GetPythonVersion returns the Python version string or empty string.
func GetPythonVersion() string {
	if bin := findPythonBinary(); bin != "" {
		if v := exec.ExecFileSafe(bin, []string{"--version"}, 0); v != "" {
			return v
		}
	}
	for _, cmd := range []string{"python3", "python"} {
		if v := exec.ExecFileSafe(cmd, []string{"--version"}, 0); v != "" {
			return v
		}
	}
	return ""
}

// ── Git Detection ──────────────────────────────────────────────────────────

// IsGitRepo walks up the directory tree looking for a .git directory.
func IsGitRepo(startDir string) bool {
	dir := startDir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return false
		}
	}
	root := filepath.VolumeName(dir) + string(filepath.Separator)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir || dir == root {
			break
		}
		dir = parent
	}
	_, err := os.Stat(filepath.Join(root, ".git"))
	return err == nil
}

// GetGitRemoteURL returns the origin remote URL, or empty string.
func GetGitRemoteURL() string {
	if !IsGitRepo("") {
		return ""
	}
	return exec.ExecSafe("git config --get remote.origin.url", "", 0)
}

// GetGitBranch returns the current git branch name, or empty string.
func GetGitBranch() string {
	if !IsGitRepo("") {
		return ""
	}
	return exec.ExecSafe("git branch --show-current", "", 0)
}

// GetGitRoot returns the git repository root path, or empty string.
func GetGitRoot() string {
	if !IsGitRepo("") {
		return ""
	}
	return exec.ExecSafe("git rev-parse --show-toplevel", "", 0)
}

// ── Project Type Detection ─────────────────────────────────────────────────

// DetectProjectType returns the project type: "monorepo", "library", or "single-repo".
func DetectProjectType(override string) string {
	if override != "" && override != "auto" {
		return override
	}
	if fileExists("pnpm-workspace.yaml") || fileExists("lerna.json") {
		return "monorepo"
	}
	if fileExists("package.json") {
		if pkg := readJSONFile("package.json"); pkg != nil {
			if _, ok := pkg["workspaces"]; ok {
				return "monorepo"
			}
			if _, ok := pkg["main"]; ok {
				return "library"
			}
			if _, ok := pkg["exports"]; ok {
				return "library"
			}
		}
	}
	return "single-repo"
}

// DetectPackageManager returns the detected package manager or empty string.
func DetectPackageManager(override string) string {
	if override != "" && override != "auto" {
		return override
	}
	switch {
	case fileExists("bun.lockb"):
		return "bun"
	case fileExists("pnpm-lock.yaml"):
		return "pnpm"
	case fileExists("yarn.lock"):
		return "yarn"
	case fileExists("package-lock.json"):
		return "npm"
	}
	return ""
}

// DetectFramework returns the detected JS/TS framework or empty string.
func DetectFramework(override string) string {
	if override != "" && override != "auto" {
		return override
	}
	if !fileExists("package.json") {
		return ""
	}
	pkg := readJSONFile("package.json")
	if pkg == nil {
		return ""
	}
	deps := mergeDeps(pkg)
	for _, check := range []struct{ dep, name string }{
		{"next", "next"},
		{"nuxt", "nuxt"},
		{"astro", "astro"},
		{"@remix-run/node", "remix"},
		{"@remix-run/react", "remix"},
		{"svelte", "svelte"},
		{"@sveltejs/kit", "svelte"},
		{"vue", "vue"},
		{"react", "react"},
		{"express", "express"},
		{"fastify", "fastify"},
		{"hono", "hono"},
	} {
		if _, ok := deps[check.dep]; ok {
			return check.name
		}
	}
	return ""
}

// ── Coding Level ───────────────────────────────────────────────────────────

// GetCodingLevelStyleName maps a coding level integer to a style file name.
func GetCodingLevelStyleName(level int) string {
	names := map[int]string{
		0: "coding-level-0-eli5",
		1: "coding-level-1-junior",
		2: "coding-level-2-mid",
		3: "coding-level-3-senior",
		4: "coding-level-4-lead",
		5: "coding-level-5-god",
	}
	if name, ok := names[level]; ok {
		return name
	}
	return "coding-level-5-god"
}

// GetCodingLevelGuidelines reads the coding level style file and returns its content.
func GetCodingLevelGuidelines(level int, configDir string) string {
	if level == -1 {
		return ""
	}
	styleName := GetCodingLevelStyleName(level)
	if configDir == "" {
		cwd, _ := os.Getwd()
		configDir = filepath.Join(cwd, ".claude")
	}
	stylePath := filepath.Join(configDir, "output-styles", styleName+".md")
	data, err := os.ReadFile(stylePath)
	if err != nil {
		return ""
	}
	// Strip YAML frontmatter (--- ... ---)
	frontmatter := regexp.MustCompile(`(?s)^---.*?---\n*`)
	content := frontmatter.ReplaceAllString(string(data), "")
	return strings.TrimSpace(content)
}

// DetectProject collects full project and environment information.
func DetectProject(typeOverride, pmOverride, frameworkOverride string) hookio.ProjectInfo {
	branch := GetGitBranch()
	root := GetGitRoot()
	url := GetGitRemoteURL()
	pm := DetectPackageManager(pmOverride)
	fw := DetectFramework(frameworkOverride)
	py := GetPythonVersion()

	user := firstNonEmpty(
		os.Getenv("USERNAME"),
		os.Getenv("USER"),
		os.Getenv("LOGNAME"),
	)

	info := hookio.ProjectInfo{
		Type:       DetectProjectType(typeOverride),
		OSPlatform: runtime.GOOS,
		User:       user,
		Locale:     os.Getenv("LANG"),
		Timezone:   localTimezone(),
	}
	if pm != "" {
		info.PackageManager = &pm
	}
	if fw != "" {
		info.Framework = &fw
	}
	if py != "" {
		info.PythonVersion = &py
	}
	if branch != "" {
		info.GitBranch = &branch
	}
	if root != "" {
		info.GitRoot = &root
	}
	if url != "" {
		info.GitURL = &url
	}
	return info
}

// ── Helpers ────────────────────────────────────────────────────────────────

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readJSONFile(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// mergeDeps merges dependencies and devDependencies from package.json.
func mergeDeps(pkg map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for _, key := range []string{"dependencies", "devDependencies"} {
		if deps, ok := pkg[key].(map[string]interface{}); ok {
			for k, v := range deps {
				merged[k] = v
			}
		}
	}
	return merged
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func localTimezone() string {
	// Try reading /etc/localtime symlink on Unix
	if runtime.GOOS != "windows" {
		if link, err := os.Readlink("/etc/localtime"); err == nil {
			const prefix = "/usr/share/zoneinfo/"
			if idx := strings.Index(link, prefix); idx >= 0 {
				return link[idx+len(prefix):]
			}
		}
		if data, err := os.ReadFile("/etc/timezone"); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	if tz := os.Getenv("TZ"); tz != "" {
		return tz
	}
	return "UTC"
}
