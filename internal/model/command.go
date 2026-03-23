package model

// Command represents an IPC request from CLI to Daemon.
type Command struct {
	Cmd  string         `json:"cmd"`
	Args map[string]any `json:"args,omitempty"`
}
