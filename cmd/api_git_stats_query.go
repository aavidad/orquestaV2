package cmd

import (
	"fmt"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func gitStatsResponseFromQuery(scopeRef, projectRef, agentRef, path, cwd string, since time.Time) (apiGitStatsResponse, error) {
	scope := strings.ToLower(strings.TrimSpace(scopeRef))
	project := strings.TrimSpace(projectRef)
	agent := strings.TrimSpace(agentRef)
	path = normalizeGitStatsPath(firstNonEmpty(path, cwd))

	switch {
	case scope == "global":
		return gitStatsGlobalResponse(since)
	case agent != "":
		return gitStatsAgentResponse(agent, project, since)
	case project != "":
		return gitStatsProjectResponse(project, path, since)
	case path != "":
		return gitStatsPathResponse(path, since)
	default:
		return apiGitStatsResponse{}, fmt.Errorf("scope=global, agente, proyecto o path/cwd es obligatorio")
	}
}

func gitStatsPathResponse(path string, since time.Time) (apiGitStatsResponse, error) {
	stats, err := apiGitStatsService.Collect(path, since)
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	return apiGitStatsResponse{
		OK:        true,
		Scope:     "path",
		Path:      path,
		Since:     since.UTC(),
		Generated: time.Now().UTC(),
		Stats:     stats,
	}, nil
}

func gitStatsProjectResponse(projectRef, explicitPath string, since time.Time) (apiGitStatsResponse, error) {
	if explicitPath != "" {
		return apiGitStatsResponse{}, fmt.Errorf("usa proyecto o path/cwd, pero no ambos")
	}
	project, err := apiGitStatsProjectLookupFn(projectRef)
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	if project == nil || strings.TrimSpace(project.RutaAbs) == "" {
		return apiGitStatsResponse{}, fmt.Errorf("proyecto sin ruta git: %s", projectRef)
	}
	path := normalizeGitStatsPath(project.RutaAbs)
	stats, err := apiGitStatsService.Collect(path, since)
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	return apiGitStatsResponse{
		OK:        true,
		Scope:     "proyecto",
		Project:   strings.TrimSpace(project.Slug),
		Path:      path,
		Since:     since.UTC(),
		Generated: time.Now().UTC(),
		Stats:     stats,
	}, nil
}

func gitStatsAgentResponse(agentRef, projectRef string, since time.Time) (apiGitStatsResponse, error) {
	project, err := gitStatsProjectSlugForAgentQuery(projectRef)
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	detail, err := apiGitStatsAgentDetailFn(agentRef)
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	path := latestGitStatsSessionCWD(detail, project)
	if path == "" {
		return apiGitStatsResponse{}, fmt.Errorf("agente sin sesion con cwd git: %s", agentRef)
	}
	stats, err := apiGitStatsService.Collect(path, since)
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	return apiGitStatsResponse{
		OK:        true,
		Scope:     "agente",
		Project:   project,
		Agent:     strings.TrimSpace(agentRef),
		Path:      path,
		Since:     since.UTC(),
		Generated: time.Now().UTC(),
		Stats:     stats,
	}, nil
}

func gitStatsGlobalResponse(since time.Time) (apiGitStatsResponse, error) {
	rows, err := apiGitStatsPanelRowsFn()
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	worktrees := make([]apiGitStatsWorktree, 0, len(rows))
	for _, row := range rows {
		agent := gitStatsRowAgent(row)
		if agent == "" {
			continue
		}
		project := gitStatsRowProject(row)
		detail, err := apiGitStatsAgentDetailFn(agent)
		if err != nil {
			return apiGitStatsResponse{}, err
		}
		path := latestGitStatsSessionCWD(detail, project)
		if path == "" {
			continue
		}
		stats, err := apiGitStatsService.Collect(path, since)
		if err != nil {
			return apiGitStatsResponse{}, err
		}
		worktrees = append(worktrees, apiGitStatsWorktree{
			Agent:   agent,
			Project: project,
			Path:    path,
			Stats:   stats,
		})
	}
	return apiGitStatsResponse{
		OK:        true,
		Scope:     "global",
		Since:     since.UTC(),
		Generated: time.Now().UTC(),
		Stats:     aggregateGitOperationalStats(worktrees),
		Worktrees: worktrees,
	}, nil
}

func gitStatsProjectSlugForAgentQuery(projectRef string) (string, error) {
	projectRef = strings.TrimSpace(projectRef)
	if projectRef == "" {
		return "", nil
	}
	project, err := apiGitStatsProjectLookupFn(projectRef)
	if err != nil {
		return "", err
	}
	if project == nil {
		return "", fmt.Errorf("proyecto no encontrado: %s", projectRef)
	}
	if slug := strings.TrimSpace(project.Slug); slug != "" {
		return slug, nil
	}
	return projectRef, nil
}

func latestGitStatsSessionCWD(detail *agentesapp.Detail, project string) string {
	if detail == nil {
		return ""
	}
	var selected *db.Sesion
	for _, session := range detail.Sesiones {
		if session == nil || strings.TrimSpace(session.CWD) == "" {
			continue
		}
		if project != "" && !strings.EqualFold(strings.TrimSpace(session.ProyectoSlug), project) {
			continue
		}
		if selected == nil || session.Inicio.After(selected.Inicio) {
			selected = session
		}
	}
	if selected == nil {
		return ""
	}
	return normalizeGitStatsPath(selected.CWD)
}

func gitStatsRowAgent(row agentesapp.Row) string {
	if row.Agente == nil {
		return ""
	}
	return strings.TrimSpace(row.Agente.Nombre)
}

func gitStatsRowProject(row agentesapp.Row) string {
	if row.Asignacion != nil {
		return strings.TrimSpace(row.Asignacion.ProyectoSlug)
	}
	if row.Sesion != nil {
		return strings.TrimSpace(row.Sesion.ProyectoSlug)
	}
	return ""
}

func aggregateGitOperationalStats(worktrees []apiGitStatsWorktree) *gitOperationalStats {
	if len(worktrees) == 0 {
		return nil
	}
	touched := make([]string, 0)
	agg := &gitOperationalStats{}
	for _, item := range worktrees {
		if item.Stats == nil {
			continue
		}
		stats := item.Stats
		touched = append(touched, stats.TouchedFiles...)
		agg.PendingAddedLines += stats.PendingAddedLines
		agg.PendingDeletedLines += stats.PendingDeletedLines
		agg.CommittedAddedLines += stats.CommittedAddedLines
		agg.CommittedDeletedLines += stats.CommittedDeletedLines
		agg.RecentCommitCount += stats.RecentCommitCount
	}
	totals := buildGitChangeTotals(
		touched,
		agg.PendingAddedLines,
		agg.PendingDeletedLines,
		agg.CommittedAddedLines,
		agg.CommittedDeletedLines,
	)
	agg.TouchedFiles = normalizeGitFileList(touched)
	agg.FilesChanged = totals.FilesChanged
	agg.Insertions = totals.Insertions
	agg.Deletions = totals.Deletions
	agg.LinesNet = totals.LinesNet
	return agg
}
