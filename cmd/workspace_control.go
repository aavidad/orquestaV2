package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type apiWorkspaceControlResponse struct {
	Control *workspaceControlReport `json:"control"`
}

type workspaceControlReport struct {
	Generated           time.Time                         `json:"generated"`
	ActiveProjects      int                               `json:"active_projects"`
	TaskCounts          map[string]int                    `json:"task_counts,omitempty"`
	DeudaDispatch       deudaDispatchResumen              `json:"deuda_dispatch"`
	Autonomia           autonomiaResumen                  `json:"autonomia"`
	AutonomySurface     *autonomySurface                  `json:"autonomy_surface,omitempty"`
	AutonomyHighlights  []string                          `json:"autonomy_highlights,omitempty"`
	AutonomyRecent      []autonomySurfaceRecentItem       `json:"autonomy_recent,omitempty"`
	AutonomyProjects    []workspaceAutonomyProjectSummary `json:"autonomy_projects,omitempty"`
	WorkersConectados   int                               `json:"workers_conectados"`
	WorkersTrabajando   int                               `json:"workers_trabajando"`
	SupervisoresActivos int                               `json:"supervisores_activos"`
	Projects            []*apiProyectoCockpit             `json:"projects,omitempty"`
}

type workspaceAutonomyProjectSummary struct {
	Project    string     `json:"project"`
	Events     int        `json:"events"`
	LastAt     *time.Time `json:"last_at,omitempty"`
	Blocking   int        `json:"blocking,omitempty"`
	Highlights []string   `json:"highlights,omitempty"`
}

var (
	workspaceControlListProjects = func() ([]map[string]any, error) {
		active := true
		return listarProyectosFiltrados("", &active)
	}
	workspaceControlCockpitBuilder = buildProyectoCockpit
	workspaceAutonomyRecentLimit   = 8
	workspaceAutonomyProjectLimit  = 8
)

func buildWorkspaceControlReport() (*workspaceControlReport, error) {
	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
	}
	projectItems, err := workspaceControlListProjects()
	if err != nil {
		return nil, err
	}

	cockpits := make([]*apiProyectoCockpit, 0, len(projectItems))
	for _, item := range projectItems {
		slug := strings.TrimSpace(fmt.Sprint(item["slug"]))
		if slug == "" {
			continue
		}
		cockpit, err := workspaceControlCockpitBuilder(slug)
		if err != nil || cockpit == nil {
			continue
		}
		cockpits = append(cockpits, cockpit)
	}
	sort.SliceStable(cockpits, func(i, j int) bool {
		left := ""
		right := ""
		if cockpit := cockpits[i]; cockpit != nil && cockpit.Proyecto != nil {
			left = strings.ToLower(strings.TrimSpace(cockpit.Proyecto.Slug))
		}
		if cockpit := cockpits[j]; cockpit != nil && cockpit.Proyecto != nil {
			right = strings.ToLower(strings.TrimSpace(cockpit.Proyecto.Slug))
		}
		return left < right
	})

	taskCounts := make(map[string]int, len(status.TareasPorEstado))
	for estado, count := range status.TareasPorEstado {
		taskCounts[estado] = count
	}

	surface := buildAutonomySurfaceFromCockpits(cockpits, workspaceAutonomyRecentLimit)
	report := &workspaceControlReport{
		Generated:           time.Now().UTC(),
		ActiveProjects:      len(cockpits),
		TaskCounts:          taskCounts,
		DeudaDispatch:       status.DeudaDispatch,
		Autonomia:           status.Autonomia,
		AutonomySurface:     surface,
		WorkersConectados:   status.WorkersConectados,
		WorkersTrabajando:   status.WorkersTrabajando,
		SupervisoresActivos: status.SupervisoresActivos,
		Projects:            cockpits,
	}
	report.AutonomyProjects = buildWorkspaceAutonomyProjects(cockpits, surface, workspaceAutonomyProjectLimit)
	if surface != nil {
		report.AutonomyHighlights = append([]string(nil), surface.Highlights...)
		report.AutonomyRecent = buildWorkspaceAutonomyRecent(surface, workspaceAutonomyRecentLimit)
	}
	topRisk, hasTopRisk := workspaceTopRiskProject(report.AutonomyProjects)
	if blocking := workspaceBlockingProjectsCount(report.AutonomyProjects); blocking > 0 {
		report.AutonomyHighlights = appendWorkspaceHighlight(report.AutonomyHighlights, fmt.Sprintf("frentes_bloqueantes=%d", blocking))
	}
	if hasTopRisk {
		report.AutonomyHighlights = appendWorkspaceHighlight(report.AutonomyHighlights, fmt.Sprintf("integracion_bloqueada=%d", topRisk.Blocking))
		report.AutonomyHighlights = appendWorkspaceHighlight(report.AutonomyHighlights, fmt.Sprintf("riesgo_top=%s(%d)", topRisk.Project, topRisk.Blocking))
	}
	return report, nil
}

func buildWorkspaceAutonomyRecent(surface *autonomySurface, limit int) []autonomySurfaceRecentItem {
	if surface == nil || len(surface.Recent) == 0 {
		return nil
	}
	items := append([]autonomySurfaceRecentItem(nil), surface.Recent...)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func buildWorkspaceAutonomyProjects(cockpits []*apiProyectoCockpit, surface *autonomySurface, limit int) []workspaceAutonomyProjectSummary {
	if len(cockpits) == 0 && (surface == nil || len(surface.Projects) == 0) {
		return nil
	}
	byProject := map[string]autonomyProjectSurface{}
	if surface != nil {
		for _, item := range surface.Projects {
			project := strings.TrimSpace(item.Project)
			if project == "" {
				continue
			}
			byProject[project] = item
		}
	}

	items := make([]workspaceAutonomyProjectSummary, 0, max(len(cockpits), len(byProject)))
	for _, cockpit := range cockpits {
		if cockpit == nil || cockpit.Proyecto == nil {
			continue
		}
		project := strings.TrimSpace(cockpit.Proyecto.Slug)
		if project == "" {
			continue
		}
		surfaceItem, ok := byProject[project]
		blockingHighlights, blockingCount := buildWorkspaceBlockingHighlights(cockpit)
		highlights := mergeWorkspaceProjectHighlights(surfaceItem.Highlights, blockingHighlights)
		events := 0
		var lastAt *time.Time
		if ok {
			events = surfaceItem.Events
			lastAt = surfaceItem.LastAt
		}
		if events <= 0 && blockingCount <= 0 {
			delete(byProject, project)
			continue
		}
		items = append(items, workspaceAutonomyProjectSummary{
			Project:    project,
			Events:     events,
			LastAt:     lastAt,
			Blocking:   blockingCount,
			Highlights: highlights,
		})
		delete(byProject, project)
	}
	for project, item := range byProject {
		project = strings.TrimSpace(project)
		if project == "" || item.Events <= 0 {
			continue
		}
		items = append(items, workspaceAutonomyProjectSummary{
			Project:    project,
			Events:     item.Events,
			LastAt:     item.LastAt,
			Highlights: append([]string(nil), item.Highlights...),
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Blocking != right.Blocking {
			return left.Blocking > right.Blocking
		}
		switch {
		case left.LastAt == nil && right.LastAt == nil:
		case left.LastAt == nil:
			return false
		case right.LastAt == nil:
			return true
		case !left.LastAt.Equal(*right.LastAt):
			return left.LastAt.After(*right.LastAt)
		}
		if left.Events != right.Events {
			return left.Events > right.Events
		}
		return left.Project < right.Project
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func buildWorkspaceBlockingHighlights(cockpit *apiProyectoCockpit) ([]string, int) {
	if cockpit == nil {
		return nil, 0
	}
	highlights := make([]string, 0, 6)
	risk := 0
	if blocked := cockpit.TareasPorEstado[string(db.TareaBloqueada)]; blocked > 0 {
		highlights = append(highlights, fmt.Sprintf("bloqueadas=%d", blocked))
		risk += 5
	}
	if gates := cockpit.ReviewGatesAbiertas; gates > 0 {
		highlights = append(highlights, fmt.Sprintf("review_gates=%d", gates))
		risk += 4
	}
	if drift := len(cockpit.WorktreeDrift); drift > 0 {
		highlights = append(highlights, fmt.Sprintf("drift=%d", drift))
		risk += 1
	}
	if runtimeOrders := cockpit.RuntimeOrdersAbiertas; runtimeOrders > 0 {
		highlights = append(highlights, fmt.Sprintf("runtime_orders=%d", runtimeOrders))
		risk += 3
	}
	if runtimeMailbox := cockpit.RuntimeMailboxPendiente; runtimeMailbox > 0 {
		highlights = append(highlights, fmt.Sprintf("mailbox_rt=%d", runtimeMailbox))
		risk += 2
	}
	if proposals := cockpit.PropuestasAbiertas; proposals > 0 {
		highlights = append(highlights, fmt.Sprintf("propuestas_abiertas=%d", proposals))
		risk += 1
	}
	if risk > 0 {
		highlights = append([]string{
			fmt.Sprintf("riesgo=%s", workspaceIntegrationRiskLabel(risk)),
			fmt.Sprintf("integracion_bloqueada=%d", risk),
		}, highlights...)
	}
	return highlights, risk
}

func workspaceIntegrationRiskLabel(score int) string {
	switch {
	case score >= 9:
		return "critico"
	case score >= 5:
		return "alto"
	case score > 0:
		return "medio"
	default:
		return ""
	}
}

func mergeWorkspaceProjectHighlights(base []string, extra []string) []string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := make([]string, 0, len(base)+len(extra))
	seen := map[string]struct{}{}
	for _, item := range append(append([]string(nil), base...), extra...) {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func appendWorkspaceHighlight(highlights []string, item string) []string {
	merged := mergeWorkspaceProjectHighlights(highlights, []string{item})
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func workspaceBlockingProjectsCount(items []workspaceAutonomyProjectSummary) int {
	total := 0
	for _, item := range items {
		if item.Blocking > 0 {
			total++
		}
	}
	return total
}

func workspaceTopRiskProject(items []workspaceAutonomyProjectSummary) (workspaceAutonomyProjectSummary, bool) {
	var best workspaceAutonomyProjectSummary
	found := false
	for _, item := range items {
		if item.Blocking <= 0 || strings.TrimSpace(item.Project) == "" {
			continue
		}
		if !found || item.Blocking > best.Blocking || (item.Blocking == best.Blocking && strings.TrimSpace(item.Project) < strings.TrimSpace(best.Project)) {
			best = item
			found = true
		}
	}
	return best, found
}
