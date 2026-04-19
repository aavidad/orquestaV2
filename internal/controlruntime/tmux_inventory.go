package controlruntime

import (
	"fmt"
	"os/exec"
	"strings"
)

type TMUXSessionInfo struct {
	SessionName string
	CurrentPath string
}

func TMUXCommandPath() (string, error) {
	return exec.LookPath("tmux")
}

func ListTMUXSessions(tmuxCommand string) ([]TMUXSessionInfo, error) {
	tmuxCommand = strings.TrimSpace(tmuxCommand)
	if tmuxCommand == "" {
		return nil, fmt.Errorf("tmux command vacio")
	}
	cmd := exec.Command(tmuxCommand, "list-panes", "-a", "-F", "#{session_name}|#{pane_current_path}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	sessions := make([]TMUXSessionInfo, 0, len(lines))
	seen := make(map[string]int)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		sessionName := strings.TrimSpace(parts[0])
		currentPath := ""
		if len(parts) > 1 {
			currentPath = strings.TrimSpace(parts[1])
		}
		if sessionName == "" {
			continue
		}
		if idx, ok := seen[sessionName]; ok {
			if strings.TrimSpace(sessions[idx].CurrentPath) == "" && currentPath != "" {
				sessions[idx].CurrentPath = currentPath
			}
			continue
		}
		seen[sessionName] = len(sessions)
		sessions = append(sessions, TMUXSessionInfo{
			SessionName: sessionName,
			CurrentPath: currentPath,
		})
	}
	return sessions, nil
}

func KillTMUXSession(tmuxCommand, sessionName string) error {
	return tmuxKillSession(strings.TrimSpace(tmuxCommand), strings.TrimSpace(sessionName))
}

func NormalizeTMUXCurrentPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.HasSuffix(raw, " (deleted)") {
		return strings.TrimSpace(strings.TrimSuffix(raw, " (deleted)")), true
	}
	return raw, false
}
