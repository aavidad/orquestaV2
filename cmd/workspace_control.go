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
	Since                 time.Time                         `json:"since"`
	Generated             time.Time                         `json:"generated"`
	ActiveProjects        int                               `json:"active_projects"`
	ProjectControlsFailed int                               `json:"project_controls_failed"`
	TaskCounts            map[string]int                    `json:"task_counts,omitempty"`
	DeudaDispatch         deudaDispatchResumen              `json:"deuda_dispatch"`
	Autonomia             autonomiaResumen                  `json:"autonomia"`
	Operational           workspaceOperationalSummary       `json:"operational"`
	Git                   workspaceControlGitAggregate      `json:"git"`
	AutonomySurface       *autonomySurface                  `json:"autonomy_surface,omitempty"`
	AutonomyHighlights    []string                          `json:"autonomy_highlights,omitempty"`
	AutonomyRecent        []autonomySurfaceRecentItem       `json:"autonomy_recent,omitempty"`
	AutonomyProjects      []workspaceAutonomyProjectSummary `json:"autonomy_projects,omitempty"`
	Agents                []workspaceControlAgentRow        `json:"agents,omitempty"`
	Timeline              []workspaceControlTimelineItem    `json:"timeline,omitempty"`
	ProjectControls       []*projectControlReport           `json:"project_controls,omitempty"`
	WorkersConectados     int                               `json:"workers_conectados"`
	WorkersTrabajando     int                               `json:"workers_trabajando"`
	SupervisoresActivos   int                               `json:"supervisores_activos"`
	Projects              []*apiProyectoCockpit             `json:"projects,omitempty"`
}

type workspaceAutonomyProjectSummary struct {
	Project    string     `json:"project"`
	Events     int        `json:"events"`
	LastAt     *time.Time `json:"last_at,omitempty"`
	Blocking   int        `json:"blocking,omitempty"`
	Highlights []string   `json:"highlights,omitempty"`
}

type workspaceOperationalSummary struct {
	State            string         `json:"state,omitempty"`
	StateReason      string         `json:"state_reason,omitempty"`
	AttentionScore   int            `json:"attention_score"`
	AttentionLabel   string         `json:"attention_label,omitempty"`
	ProjectsByState  map[string]int `json:"projects_by_state,omitempty"`
	BlockingProjects int            `json:"blocking_projects"`
}

type workspaceControlGitAggregate struct {
	TouchedFiles          []string `json:"touched_files,omitempty"`
	FilesChanged          int      `json:"files_changed"`
	PendingAddedLines     int      `json:"pending_added_lines"`
	PendingDeletedLines   int      `json:"pending_deleted_lines"`
	CommittedAddedLines   int      `json:"committed_added_lines"`
	CommittedDeletedLines int      `json:"committed_deleted_lines"`
	Insertions            int      `json:"insertions"`
	Deletions             int      `json:"deletions"`
	LinesNet              int      `json:"lines_net"`
}

type workspaceControlAgentRow struct {
	Project             string     `json:"project,omitempty"`
	Name                string     `json:"name"`
	OperationalState    string     `json:"operational_state,omitempty"`
	OperationalDetail   string     `json:"operational_detail,omitempty"`
	OpenTasks           int        `json:"open_tasks"`
	BlockedTasks        int        `json:"blocked_tasks"`
	MailboxPending      int        `json:"mailbox_pending"`
	LastAutonomyAt      *time.Time `json:"last_autonomy_at,omitempty"`
	ProjectState        string     `json:"project_state,omitempty"`
	ProjectAttention    string     `json:"project_attention,omitempty"`
	ProjectAttentionRaw int        `json:"project_attention_score,omitempty"`
}

type workspaceControlTimelineItem struct {
	Project     string    `json:"project,omitempty"`
	Kind        string    `json:"kind,omitempty"`
	Agent       string    `json:"agent,omitempty"`
	TargetAgent string    `json:"target_agent,omitempty"`
	Supervisor  string    `json:"supervisor,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

var (
	workspaceControlListProjects = func() ([]map[string]any, error) {
		active := true
		return listarProyectosFiltrados("", &active)
	}
	workspaceControlCockpitBuilder = buildProyectoCockpit
	workspaceControlProjectBuilder = buildProjectControlReport
	workspaceAutonomyRecentLimit   = 8
	workspaceAutonomyProjectLimit  = 8
	workspaceControlTimelineLimit  = 24
	workspaceControlDefaultWindow  = 24 * time.Hour
)

func buildWorkspaceControlReport() (*workspaceControlReport, error) {
	since := time.Now().UTC().Add(-workspaceControlDefaultWindow)
	return buildWorkspaceControlReportSince(since)
}

func parseWorkspaceControlSince(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Now().UTC().Add(-workspaceControlDefaultWindow), nil
	}
	return parseStatsSince(value)
}

func buildWorkspaceControlReportSince(since time.Time) (*workspaceControlReport, error) {
	if since.IsZero() {
		since = time.Now().UTC().Add(-workspaceControlDefaultWindow)
	}
	since = since.UTC()
	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
	}
	projectItems, err := workspaceControlListProjects()
	if err != nil {
		return nil, err
	}

	activeProjects := 0
	cockpits := make([]*apiProyectoCockpit, 0, len(projectItems))
	projectControls := make([]*projectControlReport, 0, len(projectItems))
	projectControlsFailed := 0
	for _, item := range projectItems {
		slug := strings.TrimSpace(fmt.Sprint(item["slug"]))
		if slug == "" {
			continue
		}
		activeProjects++
		cockpit, err := workspaceControlCockpitBuilder(slug)
		if err != nil || cockpit == nil {
			cockpit = nil
		}
		if cockpit != nil {
			cockpits = append(cockpits, cockpit)
		}
		if workspaceControlProjectBuilder != nil {
			projectControl, err := workspaceControlProjectBuilder(slug, since)
			if err != nil {
				projectControlsFailed++
			} else if projectControl != nil {
				if projectControl.Cockpit == nil {
					projectControl.Cockpit = cockpit
				}
				projectControls = append(projectControls, projectControl)
			}
		}
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
		Since:                 since,
		Generated:             time.Now().UTC(),
		ActiveProjects:        activeProjects,
		ProjectControlsFailed: projectControlsFailed,
		TaskCounts:            taskCounts,
		DeudaDispatch:         status.DeudaDispatch,
		Autonomia:             status.Autonomia,
		AutonomySurface:       surface,
		WorkersConectados:     status.WorkersConectados,
		WorkersTrabajando:     status.WorkersTrabajando,
		SupervisoresActivos:   status.SupervisoresActivos,
		Projects:              cockpits,
		ProjectControls:       projectControls,
	}
	report.AutonomyProjects = buildWorkspaceAutonomyProjects(cockpits, surface, workspaceAutonomyProjectLimit)
	report.Operational = buildWorkspaceOperationalSummary(projectControls)
	report.Git = buildWorkspaceGitAggregate(projectControls)
	report.Agents = buildWorkspaceControlAgents(projectControls)
	report.Timeline = buildWorkspaceTimeline(projectControls, workspaceControlTimelineLimit)
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

func buildWorkspaceOperationalSummary(projects []*projectControlReport) workspaceOperationalSummary {
	summary := workspaceOperationalSummary{
		ProjectsByState: map[string]int{},
		AttentionLabel:  "estable",
	}
	bestProject := ""
	bestState := ""
	bestReason := ""
	bestScore := -1
	for _, project := range projects {
		if project == nil {
			continue
		}
		state := strings.TrimSpace(project.Progress.State)
		if state == "" {
			state = "desconocido"
		}
		summary.ProjectsByState[state]++
		if state == "bloqueado" || state == "atascado" {
			summary.BlockingProjects++
		}
		score := project.Progress.AttentionScore
		projectSlug := ""
		if project.Project != nil {
			projectSlug = strings.TrimSpace(project.Project.Slug)
		}
		if score > bestScore || (score == bestScore && projectSlug != "" && (bestProject == "" || projectSlug < bestProject)) {
			bestScore = score
			bestProject = projectSlug
			bestState = state
			bestReason = strings.TrimSpace(project.Progress.StateReason)
		}
	}
	if bestScore < 0 {
		summary.ProjectsByState = nil
		return summary
	}
	summary.AttentionScore = bestScore
	summary.AttentionLabel = projectControlAttentionLabel(bestScore)
	summary.State = bestState
	summary.StateReason = bestReason
	if bestProject != "" {
		summary.StateReason = strings.TrimSpace(fmt.Sprintf("proyecto=%s · %s", bestProject, bestReason))
	}
	return summary
}

func buildWorkspaceGitAggregate(projects []*projectControlReport) workspaceControlGitAggregate {
	fileSet := map[string]struct{}{}
	out := workspaceControlGitAggregate{}
	for _, project := range projects {
		if project == nil {
			continue
		}
		out.PendingAddedLines += project.Git.PendingAddedLines
		out.PendingDeletedLines += project.Git.PendingDeletedLines
		out.CommittedAddedLines += project.Git.CommittedAddedLines
		out.CommittedDeletedLines += project.Git.CommittedDeletedLines
		for _, file := range project.Git.TouchedFiles {
			file = strings.TrimSpace(file)
			if file == "" {
				continue
			}
			fileSet[file] = struct{}{}
		}
	}
	out.TouchedFiles = mapKeysSorted(fileSet)
	out.FilesChanged = len(out.TouchedFiles)
	out.Insertions = out.PendingAddedLines + out.CommittedAddedLines
	out.Deletions = out.PendingDeletedLines + out.CommittedDeletedLines
	out.LinesNet = out.Insertions - out.Deletions
	return out
}

func buildWorkspaceControlAgents(projects []*projectControlReport) []workspaceControlAgentRow {
	if len(projects) == 0 {
		return nil
	}
	out := make([]workspaceControlAgentRow, 0)
	for _, project := range projects {
		if project == nil {
			continue
		}
		projectSlug := ""
		if project.Project != nil {
			projectSlug = strings.TrimSpace(project.Project.Slug)
		}
		for _, agent := range project.Agents {
			out = append(out, workspaceControlAgentRow{
				Project:             projectSlug,
				Name:                strings.TrimSpace(agent.Name),
				OperationalState:    strings.TrimSpace(agent.OperationalState),
				OperationalDetail:   strings.TrimSpace(agent.OperationalDetail),
				OpenTasks:           agent.OpenTasks,
				BlockedTasks:        agent.BlockedTasks,
				MailboxPending:      agent.MailboxPending,
				LastAutonomyAt:      agent.LastAutonomyAt,
				ProjectState:        strings.TrimSpace(project.Progress.State),
				ProjectAttention:    strings.TrimSpace(project.Progress.AttentionLabel),
				ProjectAttentionRaw: project.Progress.AttentionScore,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(out[i].Project) + "\x00" + strings.TrimSpace(out[i].Name))
		right := strings.ToLower(strings.TrimSpace(out[j].Project) + "\x00" + strings.TrimSpace(out[j].Name))
		return left < right
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildWorkspaceTimeline(projects []*projectControlReport, limit int) []workspaceControlTimelineItem {
	if len(projects) == 0 {
		return nil
	}
	items := make([]workspaceControlTimelineItem, 0)
	for _, project := range projects {
		if project == nil {
			continue
		}
		projectSlug := ""
		if project.Project != nil {
			projectSlug = strings.TrimSpace(project.Project.Slug)
		}
		for _, item := range project.Autonomy {
			if item.CreatedAt.IsZero() {
				continue
			}
			items = append(items, workspaceControlTimelineItem{
				Project:     projectSlug,
				Kind:        strings.TrimSpace(item.Kind),
				Agent:       strings.TrimSpace(item.Agent),
				TargetAgent: strings.TrimSpace(item.TargetAgent),
				Supervisor:  strings.TrimSpace(item.Supervisor),
				Reason:      strings.TrimSpace(item.Reason),
				CreatedAt:   item.CreatedAt.UTC(),
			})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		left := strings.ToLower(items[i].Project + "\x00" + items[i].Kind)
		right := strings.ToLower(items[j].Project + "\x00" + items[j].Kind)
		return left < right
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	if len(items) == 0 {
		return nil
	}
	return items
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
