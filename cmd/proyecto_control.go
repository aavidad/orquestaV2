package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/memoriaproyecto"
)

type projectControlReport struct {
	Project              *db.Proyecto                     `json:"project,omitempty"`
	Overview             *memoriaproyecto.ProjectOverview `json:"overview,omitempty"`
	Cockpit              *apiProyectoCockpit              `json:"cockpit,omitempty"`
	Status               apiStatusResponse                `json:"status"`
	Since                time.Time                        `json:"since"`
	Generated            time.Time                        `json:"generated"`
	Tasks                []*db.Tarea                      `json:"tasks,omitempty"`
	TaskCounts           map[string]int                   `json:"task_counts,omitempty"`
	CompletionPct        int                              `json:"completion_pct"`
	StartedAt            *time.Time                       `json:"started_at,omitempty"`
	LastActivityAt       *time.Time                       `json:"last_activity_at,omitempty"`
	Progress             projectControlProgress           `json:"progress"`
	Agents               []projectControlAgentRow         `json:"agents,omitempty"`
	AutonomyEvents       int                              `json:"autonomy_events"`
	AutonomyByKind       map[string]int                   `json:"autonomy_by_kind,omitempty"`
	AutonomyLastAt       *time.Time                       `json:"autonomy_last_at,omitempty"`
	Autonomy             []autonomyEventSummary           `json:"autonomy,omitempty"`
	AutonomyHighlights   []string                         `json:"autonomy_highlights,omitempty"`
	IntegrationRisk      string                           `json:"integration_risk,omitempty"`
	IntegrationRiskScore int                              `json:"integration_risk_score,omitempty"`
	Git                  projectControlGitAggregate       `json:"git"`
}

type projectControlProgress struct {
	TotalTasks         int    `json:"total_tasks"`
	OpenTasks          int    `json:"open_tasks"`
	CompletedTasks     int    `json:"completed_tasks"`
	BlockedTasks       int    `json:"blocked_tasks"`
	ActiveTasks        int    `json:"active_tasks"`
	ReservedTasks      int    `json:"reserved_tasks"`
	UnassignedTasks    int    `json:"unassigned_tasks"`
	ActivityLagMinutes int    `json:"activity_lag_minutes"`
	State              string `json:"state,omitempty"`
	StateReason        string `json:"state_reason,omitempty"`
	AttentionScore     int    `json:"attention_score"`
	AttentionLabel     string `json:"attention_label,omitempty"`
}

type projectControlAgentRow struct {
	Name              string     `json:"name"`
	OperationalState  string     `json:"operational_state,omitempty"`
	OperationalDetail string     `json:"operational_detail,omitempty"`
	OpenTasks         int        `json:"open_tasks"`
	BlockedTasks      int        `json:"blocked_tasks"`
	MailboxPending    int        `json:"mailbox_pending"`
	LastAutonomyAt    *time.Time `json:"last_autonomy_at,omitempty"`
}

type projectControlGitAggregate struct {
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

var proyectoControlCmd = &cobra.Command{
	Use:   "control <slug|id>",
	Short: "Resume el estado total operativo de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sinceRaw, _ := cmd.Flags().GetString("desde")
		jsonOut, _ := cmd.Flags().GetBool("json")
		since, err := parseStatsSince(strings.TrimSpace(sinceRaw))
		if err != nil {
			return err
		}
		report, err := loadProjectControlReport(strings.TrimSpace(args[0]), since)
		if err != nil {
			return err
		}
		if jsonOut {
			return imprimirJSON(apiProyectoControlResponse{Control: report})
		}
		return imprimirProyectoControl(report)
	},
}

func init() {
	proyectoControlCmd.Flags().String("desde", "24h", "Ventana temporal o timestamp RFC3339")
	proyectoControlCmd.Flags().Bool("json", false, "Salida JSON")
	proyectoCmd.AddCommand(proyectoControlCmd)
}

func loadProjectControlReport(projectRef string, since time.Time) (*projectControlReport, error) {
	report, ok, err := cargarProyectoControlDesdeAPI(projectRef, since)
	if err != nil {
		return nil, err
	}
	if !ok || report == nil {
		return nil, serverFirstCommandError("proyecto control")
	}
	return report, nil
}

func buildProjectControlReport(ref string, since time.Time) (*projectControlReport, error) {
	project, err := apiGetProyectoConRutaEfectivaTimeboxed(strings.TrimSpace(ref), "")
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, nil
	}
	overview, err := memoriaproyecto.NewService(db.ProjectMemoryRepository{}).Overview(strings.TrimSpace(project.Slug))
	if err != nil {
		return nil, err
	}
	cockpit, err := buildProyectoCockpit(strings.TrimSpace(project.Slug))
	if err != nil {
		return nil, err
	}
	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
	}
	panelRows, err := fetchAgentPanelRowsCached(statusRowsTimeout)
	if err != nil {
		panelRows, err = agentesService.BuildPanelRows()
		if err != nil {
			return nil, err
		}
	}
	tasks, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &project.ID})
	if err != nil {
		return nil, err
	}

	report := &projectControlReport{
		Project:       project,
		Overview:      overview,
		Cockpit:       cockpit,
		Status:        status,
		Since:         since.UTC(),
		Generated:     time.Now().UTC(),
		Tasks:         tasks,
		TaskCounts:    map[string]int{},
		CompletionPct: projectControlCompletionPct(tasks),
	}
	report.StartedAt = projectControlStartedAt(project, tasks)
	autonomy, err := buildProjectAutonomyEventSummaries(project.ID, since, 50)
	if err != nil {
		return nil, err
	}
	report.Autonomy, report.AutonomyByKind, report.AutonomyLastAt, report.AutonomyEvents = compactAutonomyEventSummaries(autonomy, 8)
	report.AutonomyHighlights = buildAutonomyHighlights(report.AutonomyByKind, report.Autonomy, report.AutonomyLastAt, 4)
	report.IntegrationRiskScore, report.IntegrationRisk, report.AutonomyHighlights = projectControlIntegrationRisk(cockpit, report.AutonomyHighlights)
	report.LastActivityAt = maxTimePtr(report.LastActivityAt, report.AutonomyLastAt)

	agents := filterProjectPanelRows(panelRows, project.Slug)
	sort.Slice(agents, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(agents[i].Agente.Nombre)) < strings.ToLower(strings.TrimSpace(agents[j].Agente.Nombre))
	})

	fileSet := map[string]struct{}{}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		report.TaskCounts[strings.TrimSpace(string(task.Estado))]++
	}
	for _, row := range agents {
		activity, err := buildAgentActivityReportLocal(strings.TrimSpace(row.Agente.Nombre), project.Slug, since, 200, 400)
		if err != nil {
			return nil, err
		}
		report.Agents = append(report.Agents, projectControlAgentRow{
			Name:              strings.TrimSpace(row.Agente.Nombre),
			OperationalState:  strings.TrimSpace(row.EstadoOperativo),
			OperationalDetail: strings.TrimSpace(row.DetalleOperativo),
			OpenTasks:         row.OpenTasks,
			BlockedTasks:      row.BlockedTasks,
			MailboxPending:    row.MailboxPending,
			LastAutonomyAt:    row.LastAutonomyMoment,
		})
		if activity != nil {
			report.LastActivityAt = maxTimePtr(report.LastActivityAt, projectActivityMoment(activity))
		}
		if activity != nil && activity.Git != nil {
			report.Git.PendingAddedLines += activity.Git.PendingAddedLines
			report.Git.PendingDeletedLines += activity.Git.PendingDeletedLines
			report.Git.CommittedAddedLines += activity.Git.CommittedAddedLines
			report.Git.CommittedDeletedLines += activity.Git.CommittedDeletedLines
			for _, file := range activity.Git.TouchedFiles {
				fileSet[file] = struct{}{}
			}
		}
	}
	report.Git.TouchedFiles = mapKeysSorted(fileSet)
	report.Git = canonicalProjectControlGitAggregate(report.Git)
	report.Progress = buildProjectControlProgress(report)
	return report, nil
}

func filterProjectPanelRows(rows []agentesapp.Row, projectSlug string) []agentesapp.Row {
	out := make([]agentesapp.Row, 0)
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		if row.Asignacion != nil && strings.EqualFold(strings.TrimSpace(row.Asignacion.ProyectoSlug), projectSlug) {
			out = append(out, row)
			continue
		}
		if row.Sesion != nil && strings.EqualFold(strings.TrimSpace(row.Sesion.ProyectoSlug), projectSlug) {
			out = append(out, row)
			continue
		}
	}
	return out
}

func projectControlCompletionPct(tasks []*db.Tarea) int {
	total := 0
	done := 0
	for _, task := range tasks {
		if task == nil {
			continue
		}
		switch task.Estado {
		case db.TareaCancelada:
			continue
		case db.TareaCompletada:
			total++
			done++
		default:
			total++
		}
	}
	if total == 0 {
		return 0
	}
	return int((100 * done) / total)
}

func projectControlStartedAt(project *db.Proyecto, tasks []*db.Tarea) *time.Time {
	var started *time.Time
	if project != nil {
		value := project.CreatedAt.UTC()
		started = &value
	}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if started == nil || task.CreatedAt.UTC().Before(*started) {
			value := task.CreatedAt.UTC()
			started = &value
		}
	}
	return started
}

func projectActivityMoment(report *agentActivityReport) *time.Time {
	if report == nil {
		return nil
	}
	var latest *time.Time
	for _, item := range report.Audit {
		value := item.CreatedAt.UTC()
		latest = maxTimePtr(latest, &value)
	}
	for _, item := range report.Transcript {
		if item == nil {
			continue
		}
		value := item.CreatedAt.UTC()
		latest = maxTimePtr(latest, &value)
	}
	return latest
}

func maxTimePtr(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		value := candidate.UTC()
		return &value
	}
	return current
}

func imprimirProyectoControl(report *projectControlReport) error {
	if report == nil || report.Project == nil {
		return fmt.Errorf("control de proyecto vacío")
	}
	git := canonicalProjectControlGitAggregate(report.Git)
	fmt.Printf("Proyecto:   %s\n", strings.TrimSpace(report.Project.Slug))
	fmt.Printf("Nombre:     %s\n", strings.TrimSpace(report.Project.Nombre))
	if report.StartedAt != nil {
		fmt.Printf("Inicio:     %s\n", report.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if report.LastActivityAt != nil {
		fmt.Printf("Actividad:  %s\n", report.LastActivityAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("Avance:     %d%%\n", report.CompletionPct)
	fmt.Printf("Control:    estado=%s", strings.TrimSpace(report.Progress.State))
	if report.Progress.StateReason != "" {
		fmt.Printf(" · %s", report.Progress.StateReason)
	}
	fmt.Printf(" · atención=%s(%d) · abiertas=%d activas=%d bloqueadas=%d libres=%d",
		strings.TrimSpace(report.Progress.AttentionLabel),
		report.Progress.AttentionScore,
		report.Progress.OpenTasks,
		report.Progress.ActiveTasks,
		report.Progress.BlockedTasks,
		report.Progress.UnassignedTasks,
	)
	if report.Progress.ActivityLagMinutes > 0 {
		fmt.Printf(" · lag=%s", formatProjectControlLag(report.Progress.ActivityLagMinutes))
	}
	fmt.Println()
	fmt.Printf("Autonomía:  supervisor=%d continuando=%d pendiente=%d confirmada=%d\n",
		report.Status.Autonomia.Supervisando,
		report.Status.Autonomia.Continuando,
		report.Status.Autonomia.ContinuidadPendiente,
		report.Status.Autonomia.WorkConfirmed,
	)
	if report.Cockpit != nil {
		fmt.Printf("Cockpit:    asignaciones=%d propuestas=%d review=%d mailbox_rt=%d orders_rt=%d drift=%d\n",
			report.Cockpit.AsignacionesActivas,
			report.Cockpit.PropuestasAbiertas,
			report.Cockpit.ReviewGatesAbiertas,
			report.Cockpit.RuntimeMailboxPendiente,
			report.Cockpit.RuntimeOrdersAbiertas,
			len(report.Cockpit.WorktreeDrift),
		)
	}
	if report.IntegrationRisk != "" && report.IntegrationRiskScore > 0 {
		fmt.Printf("Integración: riesgo=%s · integracion_bloqueada=%d\n",
			report.IntegrationRisk,
			report.IntegrationRiskScore,
		)
	}
	fmt.Printf("Tareas:     ")
	first := true
	keys := make([]string, 0, len(report.TaskCounts))
	for key := range report.TaskCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !first {
			fmt.Printf(" · ")
		}
		first = false
		fmt.Printf("%s=%d", key, report.TaskCounts[key])
	}
	fmt.Println()
	fmt.Printf("Agentes:    %d\n", len(report.Agents))
	for _, item := range report.Agents {
		fmt.Printf("  - %s: %s", item.Name, item.OperationalState)
		if item.OperationalDetail != "" {
			fmt.Printf(" — %s", item.OperationalDetail)
		}
		fmt.Printf(" · open=%d blocked=%d mailbox=%d\n", item.OpenTasks, item.BlockedTasks, item.MailboxPending)
	}
	fmt.Printf("Autonomy:   %d evento(s)\n", report.AutonomyEvents)
	for _, item := range topCountPairs(report.AutonomyByKind, 5) {
		fmt.Printf("  - %s: %d\n", item.Key, item.Count)
	}
	if report.AutonomyLastAt != nil {
		fmt.Printf("Último aut: %s\n", report.AutonomyLastAt.Format("2006-01-02 15:04:05"))
	}
	if len(report.AutonomyHighlights) > 0 {
		highlights := report.AutonomyHighlights
		if report.IntegrationRiskScore > 0 {
			if compact := compactProjectControlIntegrationHighlights(highlights); len(compact) > 0 {
				highlights = compact
			}
		}
		fmt.Printf("Highlights: %s\n", strings.Join(highlights, " | "))
	}
	if len(report.Autonomy) > 0 {
		fmt.Printf("Recientes:\n")
		for _, item := range report.Autonomy {
			fmt.Printf("  - %s", item.Kind)
			if item.Agent != "" {
				fmt.Printf(" · agente=%s", item.Agent)
			}
			if item.TargetAgent != "" {
				fmt.Printf(" · destino=%s", item.TargetAgent)
			}
			if item.Supervisor != "" {
				fmt.Printf(" · supervisor=%s", item.Supervisor)
			}
			if item.Reason != "" {
				fmt.Printf(" · %s", item.Reason)
			}
			if len(item.Artifacts) > 0 {
				fmt.Printf(" · artifacts=%s", strings.Join(item.Artifacts, ", "))
				if item.ArtifactsMore > 0 {
					fmt.Printf(" (+%d)", item.ArtifactsMore)
				}
			}
			fmt.Printf(" · %s\n", item.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}
	fmt.Printf("Git:        archivos=%d pending +%d/-%d commits +%d/-%d total +%d/-%d net=%+d\n",
		git.FilesChanged,
		git.PendingAddedLines,
		git.PendingDeletedLines,
		git.CommittedAddedLines,
		git.CommittedDeletedLines,
		git.Insertions,
		git.Deletions,
		git.LinesNet,
	)
	for _, file := range git.TouchedFiles {
		fmt.Printf("  - %s\n", file)
	}
	return nil
}

func canonicalProjectControlGitAggregate(in projectControlGitAggregate) projectControlGitAggregate {
	in.TouchedFiles = normalizeGitFileList(in.TouchedFiles)
	totals := buildGitChangeTotals(
		in.TouchedFiles,
		in.PendingAddedLines,
		in.PendingDeletedLines,
		in.CommittedAddedLines,
		in.CommittedDeletedLines,
	)
	in.FilesChanged = totals.FilesChanged
	in.Insertions = totals.Insertions
	in.Deletions = totals.Deletions
	in.LinesNet = totals.LinesNet
	return in
}

func projectControlIntegrationRisk(cockpit *apiProyectoCockpit, base []string) (int, string, []string) {
	riskHighlights, score := buildWorkspaceBlockingHighlights(cockpit)
	if score <= 0 {
		return 0, "", base
	}
	return score, workspaceIntegrationRiskLabel(score), mergeWorkspaceProjectHighlights(base, riskHighlights)
}

func compactProjectControlIntegrationHighlights(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "riesgo=") || strings.HasPrefix(value, "integracion_bloqueada=") {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func projectAutonomyActivityMoment(items []autonomyEventSummary) *time.Time {
	var latest *time.Time
	for _, item := range items {
		value := item.CreatedAt.UTC()
		latest = maxTimePtr(latest, &value)
	}
	return latest
}

func buildProjectControlProgress(report *projectControlReport) projectControlProgress {
	progress := projectControlProgress{
		State:          "sin_tareas",
		StateReason:    "sin backlog visible",
		AttentionLabel: "estable",
	}
	if report == nil {
		return progress
	}
	for _, task := range report.Tasks {
		if task == nil {
			continue
		}
		switch task.Estado {
		case db.TareaCancelada:
			continue
		case db.TareaCompletada:
			progress.TotalTasks++
			progress.CompletedTasks++
		case db.TareaBloqueada:
			progress.TotalTasks++
			progress.OpenTasks++
			progress.BlockedTasks++
		case db.TareaEnProgreso:
			progress.TotalTasks++
			progress.OpenTasks++
			progress.ActiveTasks++
		case db.TareaAsignada:
			progress.TotalTasks++
			progress.OpenTasks++
			progress.ReservedTasks++
		case db.TareaLibre:
			progress.TotalTasks++
			progress.OpenTasks++
			progress.UnassignedTasks++
		default:
			progress.TotalTasks++
			progress.OpenTasks++
		}
	}
	if generated := report.Generated.UTC(); !generated.IsZero() && report.LastActivityAt != nil {
		if lag := int(generated.Sub(report.LastActivityAt.UTC()).Minutes()); lag > 0 {
			progress.ActivityLagMinutes = lag
		}
	}
	progress.AttentionScore = projectControlAttentionScore(report, progress)
	progress.AttentionLabel = projectControlAttentionLabel(progress.AttentionScore)
	progress.State, progress.StateReason = projectControlState(progress, report)
	return progress
}

func projectControlAttentionScore(report *projectControlReport, progress projectControlProgress) int {
	score := 0
	if progress.BlockedTasks > 0 {
		score += minInt(progress.BlockedTasks*2, 4)
	}
	if progress.OpenTasks > 0 && progress.ActiveTasks == 0 {
		score += 2
	}
	if progress.UnassignedTasks > 0 {
		score++
	}
	if report != nil && report.Cockpit != nil {
		if report.Cockpit.ReviewGatesAbiertas > 0 {
			score++
		}
		if report.Cockpit.RuntimeMailboxPendiente > 0 {
			score++
		}
		if report.Cockpit.RuntimeOrdersAbiertas > 0 {
			score++
		}
		if report.Cockpit.PropuestasAbiertas > 0 {
			score++
		}
	}
	switch {
	case progress.ActivityLagMinutes >= 24*60:
		score += 4
	case progress.ActivityLagMinutes >= 6*60:
		score += 2
	case progress.ActivityLagMinutes >= 2*60:
		score++
	}
	return score
}

func projectControlAttentionLabel(score int) string {
	switch {
	case score >= 8:
		return "critico"
	case score >= 4:
		return "alto"
	case score > 0:
		return "bajo"
	default:
		return "estable"
	}
}

func projectControlState(progress projectControlProgress, report *projectControlReport) (string, string) {
	switch {
	case progress.TotalTasks == 0:
		return "sin_tareas", "sin backlog visible"
	case progress.OpenTasks == 0:
		return "completado", "sin trabajo abierto"
	case progress.BlockedTasks > 0 && progress.ActiveTasks == 0 && progress.ReservedTasks == 0 && progress.ActivityLagMinutes >= 6*60:
		return "atascado", "bloqueos sin ejecución ni actividad reciente"
	case progress.BlockedTasks > 0 && progress.ActiveTasks == 0 && progress.ReservedTasks == 0:
		return "bloqueado", "bloqueos abiertos sin ejecución activa"
	case progress.OpenTasks > 0 && report != nil && report.LastActivityAt == nil:
		return "frio", "backlog abierto sin actividad registrada"
	case progress.OpenTasks > 0 && progress.ActivityLagMinutes >= 24*60:
		return "atascado", "sin actividad reciente en la última jornada"
	case progress.ActiveTasks > 0:
		return "activo", "hay trabajo en progreso"
	case progress.ReservedTasks > 0 || projectControlHasStartupPressure(report):
		return "arrancando", "hay trabajo reservado o runtime pendiente"
	default:
		return "vigilancia", "backlog abierto sin ejecución actual"
	}
}

func projectControlHasStartupPressure(report *projectControlReport) bool {
	if report == nil || report.Cockpit == nil {
		return false
	}
	return report.Cockpit.RuntimeMailboxPendiente > 0 || report.Cockpit.RuntimeOrdersAbiertas > 0
}

func formatProjectControlLag(minutes int) string {
	if minutes <= 0 {
		return "0m"
	}
	hours := minutes / 60
	remain := minutes % 60
	if hours <= 0 {
		return fmt.Sprintf("%dm", remain)
	}
	if remain == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%02dm", hours, remain)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
