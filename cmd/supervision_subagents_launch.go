package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"orquesta/db"
)

type supervisorSubagentLaunchRequest struct {
	Supervisor   string `json:"supervisor,omitempty"`
	Proyecto     string `json:"proyecto,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type,omitempty"`
	Model        string `json:"model,omitempty"`
}

type supervisorSubagentLaunchResult struct {
	Supervisor string                           `json:"supervisor"`
	Proyecto   string                           `json:"proyecto,omitempty"`
	Launcher   string                           `json:"launcher"`
	Store      supervisorSubagentStoreSummary   `json:"store"`
	Refresh    *supervisorSubagentStoreRefreshResult `json:"refresh,omitempty"`
	Stdout     string                           `json:"stdout,omitempty"`
}

func resolveClaudeSubagentLauncher() string {
	if cmd := strings.TrimSpace(os.Getenv("ORQUESTA_CLAUDE_SUBAGENT_LAUNCHER")); cmd != "" {
		return cmd
	}
	return strings.TrimSpace(configOrDefault("openclaw_claude_subagent_launcher", ""))
}

func launchClaudeSubagentExternal(req supervisorSubagentLaunchRequest) (*supervisorSubagentLaunchResult, error) {
	launcher := resolveClaudeSubagentLauncher()
	if strings.TrimSpace(launcher) == "" {
		return nil, fmt.Errorf("launcher de subagentes Claude no configurado")
	}
	if strings.TrimSpace(req.Description) == "" {
		return nil, fmt.Errorf("description obligatoria")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("prompt obligatorio")
	}
	supervisor := resolveSupervisorName(req.Supervisor)
	storePath := resolveClaudeSubagentStorePath()
	cmd := exec.Command("bash", "-lc", launcher)
	cmd.Dir = mustCurrentWorkingDir()
	cmd.Env = append(os.Environ(),
		"ORQUESTA_SUBAGENT_SUPERVISOR="+supervisor,
		"ORQUESTA_SUBAGENT_PROJECT="+strings.TrimSpace(req.Proyecto),
		"ORQUESTA_SUBAGENT_NAME="+strings.TrimSpace(req.Name),
		"ORQUESTA_SUBAGENT_DESCRIPTION="+strings.TrimSpace(req.Description),
		"ORQUESTA_SUBAGENT_PROMPT="+req.Prompt,
		"ORQUESTA_SUBAGENT_TYPE="+db.NormalizeSupervisorSubagentType(req.SubagentType),
		"ORQUESTA_SUBAGENT_MODEL="+strings.TrimSpace(req.Model),
		"ORQUESTA_SUBAGENT_STORE="+storePath,
		"CLAWD_AGENT_STORE="+storePath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("launcher de subagente falló: %w: %s", err, strings.TrimSpace(string(out)))
	}
	refresh, err := refreshSupervisorSubagentsFromStore(supervisor, strings.TrimSpace(req.Proyecto))
	if err != nil {
		return nil, err
	}
	return &supervisorSubagentLaunchResult{
		Supervisor: supervisor,
		Proyecto:   strings.TrimSpace(req.Proyecto),
		Launcher:   launcher,
		Store:      refresh.Store,
		Refresh:    refresh,
		Stdout:     strings.TrimSpace(string(out)),
	}, nil
}

func mustCurrentWorkingDir() string {
	if wd, err := os.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
		return wd
	}
	return "."
}
