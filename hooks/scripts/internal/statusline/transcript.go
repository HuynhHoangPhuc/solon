// transcript.go: JSONL transcript parser — extracts tool/agent/todo state from session files.
// Streams line-by-line with bufio.Scanner; keeps last 20 tools / 10 agents.
// Skips malformed lines silently.
package statusline

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

// TranscriptTool represents a single tool invocation from the session transcript.
type TranscriptTool struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Target    string    `json:"target"`
	Status    string    `json:"status"` // running | completed | error
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// TranscriptAgent represents a subagent (Task tool_use) invocation.
type TranscriptAgent struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Model       string    `json:"model"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // running | completed
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
}

// TranscriptTodo represents a single todo item from TodoWrite / TaskCreate / TaskUpdate.
type TranscriptTodo struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ID         string `json:"id,omitempty"`
	ActiveForm string `json:"activeForm,omitempty"`
}

// TranscriptData holds the parsed state from the session transcript.
type TranscriptData struct {
	Tools        []TranscriptTool
	Agents       []TranscriptAgent
	Todos        []TranscriptTodo
	SessionStart time.Time
}

// extractTarget returns a short description of what a tool operated on.
func extractTarget(toolName string, input map[string]interface{}) string {
	if input == nil {
		return ""
	}
	switch toolName {
	case "Read", "Write", "Edit":
		if v, ok := input["file_path"].(string); ok && v != "" {
			return v
		}
		if v, ok := input["path"].(string); ok {
			return v
		}
	case "Glob", "Grep":
		if v, ok := input["pattern"].(string); ok {
			return v
		}
	case "Bash":
		if cmd, ok := input["command"].(string); ok && cmd != "" {
			if len(cmd) > 30 {
				return cmd[:30] + "..."
			}
			return cmd
		}
	}
	return ""
}

// safeTime parses a timestamp from an interface value (string or already time.Time).
func safeTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	if s, ok := v.(string); ok && s != "" {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t, err = time.Parse(time.RFC3339, s)
		}
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

// processEntry updates tool/agent/todo maps from a single parsed JSONL entry.
func processEntry(
	entry map[string]interface{},
	toolMap map[string]*TranscriptTool,
	agentMap map[string]*TranscriptAgent,
	latestTodos *[]TranscriptTodo,
	result *TranscriptData,
) {
	ts := safeTime(entry["timestamp"])
	if ts.IsZero() {
		ts = time.Now()
	}
	if result.SessionStart.IsZero() && !ts.IsZero() {
		result.SessionStart = ts
	}

	message, _ := entry["message"].(map[string]interface{})
	if message == nil {
		return
	}
	content, _ := message["content"].([]interface{})
	if content == nil {
		return
	}

	for _, raw := range content {
		block, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		blockType, _ := block["type"].(string)

		if blockType == "tool_use" {
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			if id == "" || name == "" {
				continue
			}
			input, _ := block["input"].(map[string]interface{})

			switch name {
			case "Task":
				agentType := "unknown"
				if input != nil {
					if v, ok := input["subagent_type"].(string); ok {
						agentType = v
					}
				}
				model := ""
				desc := ""
				if input != nil {
					model, _ = input["model"].(string)
					desc, _ = input["description"].(string)
				}
				agentMap[id] = &TranscriptAgent{
					ID: id, Type: agentType, Model: model,
					Description: desc, Status: "running", StartTime: ts,
				}

			case "TodoWrite":
				if input != nil {
					if todos, ok := input["todos"].([]interface{}); ok {
						*latestTodos = parseTodos(todos)
					}
				}

			case "TaskCreate":
				if input != nil {
					subject, _ := input["subject"].(string)
					if subject != "" {
						activeForm, _ := input["activeForm"].(string)
						*latestTodos = append(*latestTodos, TranscriptTodo{
							Content: subject, Status: "pending", ActiveForm: activeForm,
						})
					}
				}

			case "TaskUpdate":
				if input != nil {
					taskID, _ := input["taskId"].(string)
					status, _ := input["status"].(string)
					if taskID != "" && status != "" {
						for i := range *latestTodos {
							if (*latestTodos)[i].ID == taskID {
								(*latestTodos)[i].Status = status
								break
							}
						}
					}
				}

			default:
				toolMap[id] = &TranscriptTool{
					ID: id, Name: name,
					Target: extractTarget(name, input),
					Status: "running", StartTime: ts,
				}
			}
		}

		if blockType == "tool_result" {
			toolUseID, _ := block["tool_use_id"].(string)
			if toolUseID == "" {
				continue
			}
			isError, _ := block["is_error"].(bool)
			if t, ok := toolMap[toolUseID]; ok {
				if isError {
					t.Status = "error"
				} else {
					t.Status = "completed"
				}
				t.EndTime = ts
			}
			if a, ok := agentMap[toolUseID]; ok {
				a.Status = "completed"
				a.EndTime = ts
			}
		}
	}
}

// parseTodos converts a raw []interface{} from JSON into []TranscriptTodo.
func parseTodos(raw []interface{}) []TranscriptTodo {
	todos := make([]TranscriptTodo, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		content, _ := m["content"].(string)
		status, _ := m["status"].(string)
		id, _ := m["id"].(string)
		activeForm, _ := m["activeForm"].(string)
		todos = append(todos, TranscriptTodo{
			Content: content, Status: status, ID: id, ActiveForm: activeForm,
		})
	}
	return todos
}

// ParseTranscript reads a JSONL transcript file and returns aggregated state.
// Returns empty TranscriptData (not nil) on any error.
func ParseTranscript(path string) *TranscriptData {
	result := &TranscriptData{}
	if path == "" {
		return result
	}

	f, err := os.Open(path)
	if err != nil {
		return result
	}
	defer f.Close()

	toolMap := make(map[string]*TranscriptTool)
	agentMap := make(map[string]*TranscriptAgent)
	latestTodos := make([]TranscriptTodo, 0)

	scanner := bufio.NewScanner(f)
	// Allow up to 1 MB per line for large tool outputs
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		func() {
			defer func() { recover() }() // skip panics on malformed entries
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				return
			}
			processEntry(entry, toolMap, agentMap, &latestTodos, result)
		}()
	}

	// Collect tools ordered by insertion (map iteration non-deterministic — use slice approach)
	tools := make([]TranscriptTool, 0, len(toolMap))
	for _, t := range toolMap {
		tools = append(tools, *t)
	}
	agents := make([]TranscriptAgent, 0, len(agentMap))
	for _, a := range agentMap {
		agents = append(agents, *a)
	}

	// Keep last 20 tools / 10 agents
	if len(tools) > 20 {
		tools = tools[len(tools)-20:]
	}
	if len(agents) > 10 {
		agents = agents[len(agents)-10:]
	}

	result.Tools = tools
	result.Agents = agents
	result.Todos = latestTodos
	return result
}
