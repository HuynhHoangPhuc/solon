// Package git provides cached git repository information with a 30-second TTL.
// Cache files are stored in /tmp/sl-git-cache-{md5(cwd)[:8]}.json.
// All writes are atomic (temp file + rename). Errors fail silently.
package git

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	osexec "os/exec"
)

const cacheTTL = 30 * time.Second

// GitInfo holds git repository status for a working directory.
type GitInfo struct {
	Branch   string `json:"branch"`
	Unstaged int    `json:"unstaged"`
	Staged   int    `json:"staged"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
}

// cacheFile is the on-disk cache format.
type cacheFile struct {
	Timestamp int64    `json:"timestamp"`
	Data      *GitInfo `json:"data"`
}

// getCachePath returns the cache file path for a given working directory.
func getCachePath(cwd string) string {
	hash := md5.Sum([]byte(cwd))
	short := fmt.Sprintf("%x", hash)[:8]
	return filepath.Join(os.TempDir(), fmt.Sprintf("sl-git-cache-%s.json", short))
}

// readCache reads the cache file and returns cached data if still fresh.
// Returns (nil, false) on cache miss or expiry.
func readCache(cachePath string) (*GitInfo, bool) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}
	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, false
	}
	age := time.Since(time.Unix(cf.Timestamp, 0))
	if age >= cacheTTL {
		return nil, false
	}
	return cf.Data, true
}

// writeCache atomically writes git info to the cache file.
func writeCache(cachePath string, data *GitInfo) {
	cf := cacheFile{
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	raw, err := json.Marshal(cf)
	if err != nil {
		return
	}
	tmp := cachePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return
	}
	if err := os.Rename(tmp, cachePath); err != nil {
		_ = os.Remove(tmp)
	}
}

// execIn runs a git command in cwd, returns trimmed stdout or "".
func execIn(args []string, cwd string) string {
	cmd := osexec.Command("git", args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// countLines counts non-empty lines in a string.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			count++
		}
	}
	return count
}

// fetchGitInfo queries git for live repository status.
// Returns nil if cwd is not a git repository.
func fetchGitInfo(cwd string) *GitInfo {
	if execIn([]string{"rev-parse", "--git-dir"}, cwd) == "" {
		return nil
	}

	branch := execIn([]string{"branch", "--show-current"}, cwd)
	if branch == "" {
		branch = execIn([]string{"rev-parse", "--short", "HEAD"}, cwd)
	}

	unstaged := countLines(execIn([]string{"diff", "--name-only"}, cwd))
	staged := countLines(execIn([]string{"diff", "--cached", "--name-only"}, cwd))

	var ahead, behind int
	aheadBehind := execIn([]string{"rev-list", "--left-right", "--count", "@{u}...HEAD"}, cwd)
	if aheadBehind != "" {
		parts := strings.Fields(aheadBehind)
		if len(parts) >= 2 {
			behind, _ = strconv.Atoi(parts[0])
			ahead, _ = strconv.Atoi(parts[1])
		}
	}

	return &GitInfo{
		Branch:   branch,
		Unstaged: unstaged,
		Staged:   staged,
		Ahead:    ahead,
		Behind:   behind,
	}
}

// GetGitInfo returns git info for cwd with a 30-second TTL cache.
// Returns nil if cwd is not a git repository.
func GetGitInfo(cwd string) *GitInfo {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil
		}
	}
	cachePath := getCachePath(cwd)
	if cached, ok := readCache(cachePath); ok {
		return cached
	}
	data := fetchGitInfo(cwd)
	writeCache(cachePath, data)
	return data
}

// InvalidateCache removes the cache file for cwd (call after file changes).
func InvalidateCache(cwd string) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return
		}
	}
	_ = os.Remove(getCachePath(cwd))
}
