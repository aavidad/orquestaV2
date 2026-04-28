package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"orquesta/coordinacion"
	"orquesta/db"
)

type supervisorSubagentLaunchRequest struct {
	Supervisor   string         `json:"supervisor,omitempty"`
	Proyecto     string         `json:"proyecto,omitempty"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description"`
	Prompt       string         `json:"prompt"`
	SubagentType string         `json:"subagent_type,omitempty"`
	Model        string         `json:"model,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type supervisorSubagentLaunchResult struct {
	Supervisor string                                `json:"supervisor"`
	Proyecto   string                                `json:"proyecto,omitempty"`
	Launcher   string                                `json:"launcher"`
	Store      supervisorSubagentStoreSummary        `json:"store"`
	Refresh    *supervisorSubagentStoreRefreshResult `json:"refresh,omitempty"`
	Stdout     string                                `json:"stdout,omitempty"`
	Worktree   map[string]any                        `json:"worktree,omitempty"`
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
	req.Name = resolverNombreSubagenteLaunch(req)
	storePath := resolveClaudeSubagentStorePath()
	cmd := exec.Command("bash", "-lc", launcher)
	cmd.Dir = mustCurrentWorkingDir()
	var worktreeInfo map[string]any
	if strings.TrimSpace(req.Proyecto) != "" && strings.TrimSpace(req.Name) != "" {
		worktree, err := prepararWorktreeSubagente(strings.TrimSpace(req.Proyecto), strings.TrimSpace(req.Name))
		if err != nil {
			worktree = nil
		}
		if worktree != nil {
			cmd.Dir = strings.TrimSpace(worktree.Path)
			worktreeInfo = buildSupervisorSubagentLaunchWorktreeMetadata(worktree)
		}
	}
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
	if worktreeInfo != nil {
		cmd.Env = append(cmd.Env, buildSupervisorSubagentLaunchContractEnv(worktreeInfo)...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("launcher de subagente falló: %w: %s", err, strings.TrimSpace(string(out)))
	}
	refresh, err := refreshSupervisorSubagentsFromStore(supervisor, strings.TrimSpace(req.Proyecto))
	if err != nil {
		return nil, err
	}
	metadataInfo := map[string]any{}
	for key, value := range req.Metadata {
		metadataInfo[key] = value
	}
	for key, value := range worktreeInfo {
		metadataInfo[key] = value
	}
	if len(metadataInfo) > 0 {
		refresh.Subagents = persistirMetadataEnSubagenteLanzado(refresh.Subagents, supervisor, strings.TrimSpace(req.Proyecto), strings.TrimSpace(req.Name), metadataInfo)
	}
	return &supervisorSubagentLaunchResult{
		Supervisor: supervisor,
		Proyecto:   strings.TrimSpace(req.Proyecto),
		Launcher:   launcher,
		Store:      refresh.Store,
		Refresh:    refresh,
		Stdout:     strings.TrimSpace(string(out)),
		Worktree:   worktreeInfo,
	}, nil
}

func persistirMetadataEnSubagenteLanzado(items []*db.SupervisorSubagent, supervisor, proyectoSlug, nombre string, metadata map[string]any) []*db.SupervisorSubagent {
	if len(items) == 0 || strings.TrimSpace(nombre) == "" || len(metadata) == 0 {
		return items
	}
	objetivo := seleccionarSubagentePersistenciaWorktree(items, strings.TrimSpace(nombre))
	if objetivo == nil {
		return items
	}
	metadataJSON := mergeSupervisorSubagentMetadata(objetivo.MetadataJSON, metadata)
	actualizado, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
		Supervisor:      strings.TrimSpace(supervisor),
		ProyectoSlug:    strings.TrimSpace(proyectoSlug),
		SessionID:       strings.TrimSpace(objetivo.SessionID),
		ParentThreadID:  strings.TrimSpace(objetivo.ParentThreadID),
		ThreadID:        strings.TrimSpace(objetivo.ThreadID),
		SubagentName:    strings.TrimSpace(objetivo.SubagentName),
		SubagentType:    strings.TrimSpace(objetivo.SubagentType),
		ToolProfileJSON: strings.TrimSpace(objetivo.ToolProfileJSON),
		Status:          strings.TrimSpace(objetivo.Status),
		ManifestPath:    strings.TrimSpace(objetivo.ManifestPath),
		OutputPath:      strings.TrimSpace(objetivo.OutputPath),
		ErrorMessage:    strings.TrimSpace(objetivo.ErrorMessage),
		MetadataJSON:    metadataJSON,
		CreatedAt:       &objetivo.CreatedAt,
		StartedAt:       &objetivo.StartedAt,
		CompletedAt:     objetivo.CompletedAt,
	})
	if err != nil || actualizado == nil {
		return items
	}
	out := append([]*db.SupervisorSubagent(nil), items...)
	for i, item := range out {
		if item != nil && item.ID == actualizado.ID {
			out[i] = actualizado
			break
		}
	}
	return out
}

func seleccionarSubagentePersistenciaWorktree(items []*db.SupervisorSubagent, nombre string) *db.SupervisorSubagent {
	var elegido *db.SupervisorSubagent
	for _, item := range items {
		if item == nil || !strings.EqualFold(strings.TrimSpace(item.SubagentName), strings.TrimSpace(nombre)) {
			continue
		}
		if elegido == nil || item.UpdatedAt.After(elegido.UpdatedAt) {
			elegido = item
		}
	}
	return elegido
}

func mergeSupervisorSubagentMetadata(base string, worktreeInfo map[string]any) string {
	payload := map[string]any{}
	if strings.TrimSpace(base) != "" {
		_ = json.Unmarshal([]byte(base), &payload)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	for key, value := range worktreeInfo {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				continue
			}
			payload[key] = strings.TrimSpace(v)
		case []string:
			if len(v) == 0 {
				continue
			}
			payload[key] = append([]string(nil), v...)
		default:
			if value == nil {
				continue
			}
			payload[key] = value
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return strings.TrimSpace(base)
	}
	return string(raw)
}

func buildSupervisorSubagentLaunchWorktreeMetadata(worktree *coordinacion.Worktree) map[string]any {
	if worktree == nil {
		return nil
	}
	path := strings.TrimSpace(worktree.Path)
	branch := strings.TrimSpace(worktree.Branch)
	baseRef := strings.TrimSpace(worktree.BaseRef)
	return map[string]any{
		"id":                   worktree.ID,
		"path":                 path,
		"branch":               branch,
		"base_ref":             baseRef,
		"worktree_id":          worktree.ID,
		"ruta_worktree":        path,
		"branch_worktree":      branch,
		"base_ref_worktree":    baseRef,
		"workspace_strategy":   "git_worktree",
		"transport_preference": "tmux",
		"resume_capability":    "session_resume",
		"resume_strategy":      "resumen_y_payload",
		"execution_profile":    "worktree_tmux_resume",
	}
}

func buildSupervisorSubagentLaunchContractEnv(worktreeInfo map[string]any) []string {
	if len(worktreeInfo) == 0 {
		return nil
	}
	worktreeID := int64SupervisorSubagente(worktreeInfo["worktree_id"])
	if worktreeID == 0 {
		worktreeID = int64SupervisorSubagente(worktreeInfo["id"])
	}
	env := []string{
		"ORQUESTA_SUBAGENT_WORKTREE=" + firstNonEmptyLaunch(
			stringFromAnyLaunch(worktreeInfo["ruta_worktree"]),
			stringFromAnyLaunch(worktreeInfo["path"]),
		),
		"ORQUESTA_SUBAGENT_PROJECT_PATH=" + firstNonEmptyLaunch(
			stringFromAnyLaunch(worktreeInfo["ruta_worktree"]),
			stringFromAnyLaunch(worktreeInfo["path"]),
		),
		"ORQUESTA_SUBAGENT_BRANCH=" + firstNonEmptyLaunch(
			stringFromAnyLaunch(worktreeInfo["branch_worktree"]),
			stringFromAnyLaunch(worktreeInfo["branch"]),
		),
		"ORQUESTA_SUBAGENT_BASE_REF=" + firstNonEmptyLaunch(
			stringFromAnyLaunch(worktreeInfo["base_ref_worktree"]),
			stringFromAnyLaunch(worktreeInfo["base_ref"]),
		),
		"ORQUESTA_SUBAGENT_WORKSPACE_STRATEGY=" + stringFromAnyLaunch(worktreeInfo["workspace_strategy"]),
		"ORQUESTA_SUBAGENT_TRANSPORT_PREFERENCE=" + stringFromAnyLaunch(worktreeInfo["transport_preference"]),
		"ORQUESTA_SUBAGENT_RESUME_CAPABILITY=" + stringFromAnyLaunch(worktreeInfo["resume_capability"]),
		"ORQUESTA_SUBAGENT_RESUME_STRATEGY=" + stringFromAnyLaunch(worktreeInfo["resume_strategy"]),
		"ORQUESTA_SUBAGENT_EXECUTION_PROFILE=" + stringFromAnyLaunch(worktreeInfo["execution_profile"]),
	}
	if worktreeID > 0 {
		env = append(env, "ORQUESTA_SUBAGENT_WORKTREE_ID="+strconv.FormatInt(worktreeID, 10))
	}
	return env
}

func resolverNombreSubagenteLaunch(req supervisorSubagentLaunchRequest) string {
	if nombre := strings.TrimSpace(req.Name); nombre != "" {
		return nombre
	}
	supervisor := strings.TrimSpace(req.Supervisor)
	if supervisor == "" {
		supervisor = "OpenClaw"
	}
	tipo := db.NormalizeSupervisorSubagentType(req.SubagentType)
	tipo = strings.ReplaceAll(tipo, "general-purpose", "general")
	tipo = strings.ReplaceAll(tipo, "-", "_")
	return fmt.Sprintf("%s-%s-%d", supervisor, tipo, time.Now().UTC().Unix())
}

func prepararWorktreeSubagente(proyectoRef, agente string) (*coordinacion.Worktree, error) {
	svc := newCoordinationService()
	worktree, err := svc.PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: strings.TrimSpace(proyectoRef),
		Agent:      strings.TrimSpace(agente),
		Reason:     "launch_subagente_externo",
		BaseRef:    "HEAD",
	})
	if err == nil {
		return worktree, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	texto := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(texto, "not a git repository"):
		return nil, nil
	case strings.Contains(texto, "git worktree add"):
		return nil, nil
	case strings.Contains(texto, "cannot lock ref"):
		return nil, nil
	}
	return nil, err
}

func stringFromAnyLaunch(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func firstNonEmptyLaunch(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func mustCurrentWorkingDir() string {
	if wd, err := os.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
		return wd
	}
	return "."
}
