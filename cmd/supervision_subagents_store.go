package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type claudeSubagentStoreManifest struct {
	AgentID      string  `json:"agentId"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	SubagentType *string `json:"subagentType"`
	Model        *string `json:"model"`
	Status       string  `json:"status"`
	OutputFile   string  `json:"outputFile"`
	ManifestFile string  `json:"manifestFile"`
	CreatedAt    string  `json:"createdAt"`
	StartedAt    *string `json:"startedAt"`
	CompletedAt  *string `json:"completedAt"`
	Error        *string `json:"error"`
}

type supervisorSubagentStoreSummary struct {
	Path             string     `json:"path,omitempty"`
	Exists           bool       `json:"exists"`
	ManifestCount    int        `json:"manifest_count"`
	LatestManifestAt *time.Time `json:"latest_manifest_at,omitempty"`
}

type supervisorSubagentStoreRefreshResult struct {
	Supervisor string                   `json:"supervisor"`
	Proyecto   string                   `json:"proyecto,omitempty"`
	Store      supervisorSubagentStoreSummary `json:"store"`
	Imported   int                      `json:"imported"`
	Subagents  []*db.SupervisorSubagent `json:"subagents,omitempty"`
}

func resolveClaudeSubagentStorePath() string {
	if path := strings.TrimSpace(os.Getenv("CLAWD_AGENT_STORE")); path != "" {
		return path
	}
	if path := strings.TrimSpace(configOrDefault("openclaw_claude_subagent_store", "")); path != "" {
		return path
	}
	if wd, err := os.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
		return filepath.Join(wd, ".clawd-agents")
	}
	return ".clawd-agents"
}

func inspectClaudeSubagentStore() (supervisorSubagentStoreSummary, []string, error) {
	path := resolveClaudeSubagentStorePath()
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return supervisorSubagentStoreSummary{Path: path, Exists: false}, nil, nil
		}
		return supervisorSubagentStoreSummary{}, nil, err
	}
	if !info.IsDir() {
		return supervisorSubagentStoreSummary{}, nil, fmt.Errorf("store de subagentes no es directorio: %s", path)
	}
	entries, err := filepath.Glob(filepath.Join(path, "*.json"))
	if err != nil {
		return supervisorSubagentStoreSummary{}, nil, err
	}
	sort.Strings(entries)
	summary := supervisorSubagentStoreSummary{
		Path:          path,
		Exists:        true,
		ManifestCount: len(entries),
	}
	for _, item := range entries {
		info, err := os.Stat(item)
		if err != nil {
			continue
		}
		mod := info.ModTime().UTC()
		if summary.LatestManifestAt == nil || mod.After(*summary.LatestManifestAt) {
			summary.LatestManifestAt = &mod
		}
	}
	return summary, entries, nil
}

func parseClaudeSubagentManifest(path string) (*claudeSubagentStoreManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var item claudeSubagentStoreManifest
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	if strings.TrimSpace(item.AgentID) == "" {
		return nil, fmt.Errorf("manifest sin agentId: %s", path)
	}
	if strings.TrimSpace(item.ManifestFile) == "" {
		item.ManifestFile = path
	}
	return &item, nil
}

func normalizeClaudeSubagentStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	case "cancelled", "canceled":
		return "cancelled"
	default:
		return "running"
	}
}

func parseManifestTime(raw *string, fallback string) *time.Time {
	value := strings.TrimSpace(fallback)
	if raw != nil && strings.TrimSpace(*raw) != "" {
		value = strings.TrimSpace(*raw)
	}
	if value == "" {
		return nil
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		ts, err = time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return nil
		}
	}
	utc := ts.UTC()
	return &utc
}

func refreshSupervisorSubagentsFromStore(supervisor, proyectoSlug string) (*supervisorSubagentStoreRefreshResult, error) {
	supervisor = resolveSupervisorName(supervisor)
	summary, manifests, err := inspectClaudeSubagentStore()
	if err != nil {
		return nil, err
	}
	result := &supervisorSubagentStoreRefreshResult{
		Supervisor: supervisor,
		Proyecto:   strings.TrimSpace(proyectoSlug),
		Store:      summary,
	}
	if !summary.Exists || len(manifests) == 0 {
		return result, nil
	}
	imported := make([]*db.SupervisorSubagent, 0, len(manifests))
	for _, manifestPath := range manifests {
		item, err := parseClaudeSubagentManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		subType := "general-purpose"
		if item.SubagentType != nil {
			subType = *item.SubagentType
		}
		createdAt := parseManifestTime(nil, item.CreatedAt)
		startedAt := parseManifestTime(item.StartedAt, item.CreatedAt)
		completedAt := parseManifestTime(item.CompletedAt, "")
		metadata := map[string]any{
			"source":      "clawd_store",
			"agent_id":    strings.TrimSpace(item.AgentID),
			"description": strings.TrimSpace(item.Description),
		}
		if item.Model != nil && strings.TrimSpace(*item.Model) != "" {
			metadata["model"] = strings.TrimSpace(*item.Model)
		}
		metadataJSON, _ := json.Marshal(metadata)
		persisted, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:   supervisor,
			ProyectoSlug: strings.TrimSpace(proyectoSlug),
			ThreadID:     strings.TrimSpace(item.AgentID),
			SubagentName: strings.TrimSpace(item.Name),
			SubagentType: subType,
			Status:       normalizeClaudeSubagentStatus(item.Status),
			ManifestPath: strings.TrimSpace(item.ManifestFile),
			OutputPath:   strings.TrimSpace(item.OutputFile),
			ErrorMessage: strings.TrimSpace(derefString(item.Error)),
			MetadataJSON: string(metadataJSON),
			CreatedAt:    createdAt,
			StartedAt:    startedAt,
			CompletedAt:  completedAt,
		})
		if err != nil {
			return nil, err
		}
		imported = append(imported, persisted)
	}
	result.Imported = len(imported)
	result.Subagents = imported
	return result, nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
