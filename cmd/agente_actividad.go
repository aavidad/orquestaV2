package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/gitestadisticasapp"
)

type agentActivitySummary struct {
	AuditEntries          int            `json:"audit_entries"`
	TranscriptEntries     int            `json:"transcript_entries"`
	AutonomyEvents        int            `json:"autonomy_events"`
	LastAutonomyEventAt   *time.Time     `json:"last_autonomy_event_at,omitempty"`
	LastTestSignalAt      *time.Time     `json:"last_test_signal_at,omitempty"`
	LastGitSignalAt       *time.Time     `json:"last_git_signal_at,omitempty"`
	AuditByAction         map[string]int `json:"audit_by_action,omitempty"`
	TranscriptBySignal    map[string]int `json:"transcript_by_signal,omitempty"`
	TranscriptByStream    map[string]int `json:"transcript_by_stream,omitempty"`
	AutonomyByKind        map[string]int `json:"autonomy_by_kind,omitempty"`
	OpenTasks             int            `json:"open_tasks"`
	BlockedTasks          int            `json:"blocked_tasks"`
	MailboxPending        int            `json:"mailbox_pending"`
	OrdersOpen            int            `json:"orders_open"`
	OrdersFailed          int            `json:"orders_failed"`
	LastAutonomyAction    string         `json:"last_autonomy_action,omitempty"`
	LastAutonomySource    string         `json:"last_autonomy_source,omitempty"`
	CurrentOperational    string         `json:"current_operational,omitempty"`
	CurrentOperationalWhy string         `json:"current_operational_detail,omitempty"`
	CurrentTasks          []string       `json:"current_tasks,omitempty"`
	CommitCadence         string         `json:"commit_cadence,omitempty"`
	CommitCadenceDetail   string         `json:"commit_cadence_detail,omitempty"`
	AutonomyHighlights    []string       `json:"autonomy_highlights,omitempty"`
	IntegrationRisk       string         `json:"integration_risk,omitempty"`
	IntegrationRiskScore  int            `json:"integration_risk_score,omitempty"`
	IntegrationHighlights []string       `json:"integration_highlights,omitempty"`
}

type agentGitActivityStats = gitestadisticasapp.Stats

type agentActivityReport struct {
	Agent      string                       `json:"agent"`
	Project    string                       `json:"project,omitempty"`
	Since      time.Time                    `json:"since"`
	Generated  time.Time                    `json:"generated"`
	Detail     *agentesapp.Detail           `json:"detail,omitempty"`
	Audit      []db.AuditEntry              `json:"audit,omitempty"`
	Transcript []*db.RuntimeTranscriptEntry `json:"transcript,omitempty"`
	Git        *agentGitActivityStats       `json:"git,omitempty"`
	Autonomy   []autonomyEventSummary       `json:"autonomy,omitempty"`
	Summary    agentActivitySummary         `json:"summary"`
}

var agentGitStatsService = gitestadisticasapp.NewService()
var resolveAgentActivityWorktree = func(project, agent string) string {
	if worktree, err := gitService.ResolveActiveWorktree(strings.TrimSpace(project), strings.TrimSpace(agent)); err == nil && worktree != nil {
		return strings.TrimSpace(worktree.RutaAbs)
	}
	return ""
}

var agenteActividadCmd = &cobra.Command{
	Use:   "actividad <agente>",
	Short: "Resume actividad operativa y git de un agente en una ventana temporal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agenteNombre := strings.TrimSpace(args[0])
		proyecto, _ := cmd.Flags().GetString("proyecto")
		desdeRaw, _ := cmd.Flags().GetString("desde")
		jsonOut, _ := cmd.Flags().GetBool("json")
		auditLimit, _ := cmd.Flags().GetInt("audit-limit")
		transcriptLimit, _ := cmd.Flags().GetInt("transcript-limit")

		desde, err := parseStatsSince(strings.TrimSpace(desdeRaw))
		if err != nil {
			return err
		}
		report, err := loadAgentActivityReport(agenteNombre, strings.TrimSpace(proyecto), desde, auditLimit, transcriptLimit)
		if err != nil {
			return err
		}
		if jsonOut {
			return imprimirJSON(report)
		}
		return imprimirAgenteActividad(report)
	},
}

func init() {
	agenteActividadCmd.Flags().String("proyecto", "", "Proyecto al que acotar la actividad")
	agenteActividadCmd.Flags().String("desde", "1h", "Ventana temporal o timestamp RFC3339 (ej: 1h, 24h, 2026-04-23T10:00:00Z)")
	agenteActividadCmd.Flags().Int("audit-limit", 200, "Número máximo de eventos de auditoría a recuperar")
	agenteActividadCmd.Flags().Int("transcript-limit", 400, "Número máximo de líneas de transcript a recuperar")
	agenteActividadCmd.Flags().Bool("json", false, "Salida JSON")
	agenteCmd.AddCommand(agenteActividadCmd)
}

func parseStatsSince(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Now().UTC().Add(-time.Hour), nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		return time.Now().UTC().Add(-d), nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("valor --desde inválido: %q", raw)
}

func loadAgentActivityReport(agent, project string, since time.Time, auditLimit, transcriptLimit int) (*agentActivityReport, error) {
	if report, ok, err := cargarAgenteActividadDesdeAPI(agent, project, since); err != nil {
		return nil, err
	} else if ok && report != nil {
		return report, nil
	}
	return buildAgentActivityReportLocal(agent, project, since, auditLimit, transcriptLimit)
}

func buildAgentActivityReportLocal(agent, project string, since time.Time, auditLimit, transcriptLimit int) (*agentActivityReport, error) {
	detail, err := agentesService.BuildDetailCompact(strings.TrimSpace(agent))
	if err != nil {
		return nil, err
	}
	var projectEntity *db.Proyecto
	audit := make([]db.AuditEntry, 0)
	if auditLimit > 0 {
		logs, err := db.ListarAuditoria(db.FiltroAuditoria{
			Agente: &agent,
			Desde:  &since,
			Limite: auditLimit,
		})
		if err != nil {
			return nil, err
		}
		audit = make([]db.AuditEntry, 0, len(logs))
		for _, item := range logs {
			if item == nil {
				continue
			}
			audit = append(audit, db.AuditEntry{
				Agente:    item.Agente,
				Accion:    item.Accion,
				Entidad:   item.Entidad,
				EntidadID: item.EntidadID,
				Detalle:   item.Detalle,
				CreatedAt: item.CreatedAt,
			})
		}
	}
	project, projectEntity, err = resolveAgentActivityProject(detail, project)
	if err != nil {
		return nil, err
	}
	transcript := make([]*db.RuntimeTranscriptEntry, 0)
	if transcriptLimit > 0 {
		filter := db.FiltroRuntimeTranscript{Agente: &agent, Desde: &since, Limit: transcriptLimit}
		if projectEntity != nil {
			filter.ProyectoID = &projectEntity.ID
		}
		transcript, err = db.ListarRuntimeTranscript(filter)
		if err != nil {
			return nil, err
		}
	}
	cwd := resolveAgentActivityCWD(detail, project)
	gitStats, err := collectAgentGitActivityStats(cwd, since)
	if err != nil {
		return nil, err
	}
	report := &agentActivityReport{
		Agent:      agent,
		Project:    project,
		Since:      since.UTC(),
		Generated:  time.Now().UTC(),
		Detail:     detail,
		Audit:      audit,
		Transcript: transcript,
		Git:        gitStats,
	}
	autonomy := make([]autonomyEventSummary, 0)
	if report.Detail != nil && report.Detail.Row.Asignacion != nil && report.Detail.Row.Asignacion.ProyectoID > 0 {
		projectID := report.Detail.Row.Asignacion.ProyectoID
		autonomy, err = buildAgentAutonomyEventSummaries(agent, &projectID, since, 50)
		if err != nil {
			return nil, err
		}
	} else if projectEntity != nil {
		autonomy, err = buildAgentAutonomyEventSummaries(agent, &projectEntity.ID, since, 50)
		if err != nil {
			return nil, err
		}
	} else {
		autonomy, err = buildAgentAutonomyEventSummaries(agent, nil, since, 50)
		if err != nil {
			return nil, err
		}
	}
	report.Autonomy, _, _, _ = compactAutonomyEventSummaries(autonomy, 8)
	report.Summary = summarizeAgentActivity(detail, audit, transcript, autonomy)
	report.Summary.CommitCadence, report.Summary.CommitCadenceDetail = summarizeAgentCommitCadence(report.Git)
	if projectEntity != nil {
		projectAutonomy, err := buildProjectAutonomyEventSummaries(projectEntity.ID, since, 50)
		if err != nil {
			return nil, err
		}
		_, projectAutonomyByKind, projectAutonomyLastAt, _ := compactAutonomyEventSummaries(projectAutonomy, 8)
		report.Summary.AutonomyHighlights = buildAutonomyHighlights(projectAutonomyByKind, projectAutonomy, projectAutonomyLastAt, 4)
		if cockpit, err := buildProyectoCockpit(projectEntity.Slug); err == nil && cockpit != nil {
			report.Summary.IntegrationRiskScore, report.Summary.IntegrationRisk, report.Summary.IntegrationHighlights = projectControlIntegrationRisk(cockpit, nil)
			report.Summary.IntegrationHighlights = compactProjectControlIntegrationHighlights(report.Summary.IntegrationHighlights)
		}
	}
	return report, nil
}

func fetchAgentOverviewForActivity(agent string) (*agentesapp.Detail, error) {
	var out apiAgenteOverviewResponse
	if ok, err := apiGet(fmt.Sprintf("/api/agentes/%s/overview", url.PathEscape(agent)), &out); err != nil {
		return nil, err
	} else if ok {
		if out.Detail == nil {
			return nil, fmt.Errorf("overview vacío")
		}
		return out.Detail, nil
	}
	return nil, serverFirstCommandError("agente actividad")
}

func fetchAgentAuditForActivity(agent string, since time.Time, limit int) ([]db.AuditEntry, error) {
	query := url.Values{
		"agente": []string{strings.TrimSpace(agent)},
		"desde":  []string{since.UTC().Format(time.RFC3339)},
		"limit":  []string{strconv.Itoa(limit)},
	}
	var resp apiAuditResponse
	if ok, err := apiGetQuery("/api/audit", query, &resp); err != nil {
		return nil, err
	} else if ok {
		return resp.Audit, nil
	}
	return nil, serverFirstCommandError("agente actividad")
}

func fetchAgentTranscriptForActivity(agent, project string, since time.Time, limit int) ([]*db.RuntimeTranscriptEntry, error) {
	query := url.Values{
		"agente": []string{strings.TrimSpace(agent)},
		"desde":  []string{since.UTC().Format(time.RFC3339)},
		"limit":  []string{strconv.Itoa(limit)},
	}
	if strings.TrimSpace(project) != "" {
		query.Set("proyecto", strings.TrimSpace(project))
	}
	var resp apiRuntimeTranscriptResponse
	if ok, err := apiGetQuery("/api/runtime-transcript", query, &resp); err != nil {
		return nil, err
	} else if ok {
		return resp.Transcript, nil
	}
	return nil, serverFirstCommandError("agente actividad")
}

func summarizeAgentActivity(detail *agentesapp.Detail, audit []db.AuditEntry, transcript []*db.RuntimeTranscriptEntry, autonomy []autonomyEventSummary) agentActivitySummary {
	out := agentActivitySummary{
		AuditEntries:       len(audit),
		TranscriptEntries:  len(transcript),
		AutonomyEvents:     len(autonomy),
		AuditByAction:      map[string]int{},
		TranscriptBySignal: map[string]int{},
		TranscriptByStream: map[string]int{},
		AutonomyByKind:     map[string]int{},
	}
	for _, item := range audit {
		key := strings.TrimSpace(item.Accion)
		if key == "" {
			key = "sin_accion"
		}
		out.AuditByAction[key]++
	}
	for _, item := range transcript {
		if item == nil {
			continue
		}
		signal := classifyAgentActivityTranscriptSignal(item)
		stream := strings.TrimSpace(item.Stream)
		if stream == "" {
			stream = "unknown"
		}
		out.TranscriptBySignal[signal]++
		out.TranscriptByStream[stream]++
		if !item.CreatedAt.IsZero() {
			value := item.CreatedAt.UTC()
			switch signal {
			case "tests":
				out.LastTestSignalAt = maxTimePtr(out.LastTestSignalAt, &value)
			case "git_activity":
				out.LastGitSignalAt = maxTimePtr(out.LastGitSignalAt, &value)
			}
		}
	}
	for _, item := range autonomy {
		key := strings.TrimSpace(item.Kind)
		if key == "" {
			key = "sin_kind"
		}
		out.AutonomyByKind[key]++
		value := item.CreatedAt.UTC()
		out.LastAutonomyEventAt = maxTimePtr(out.LastAutonomyEventAt, &value)
	}
	if detail != nil {
		out.OpenTasks = detail.Row.OpenTasks
		out.BlockedTasks = detail.Row.BlockedTasks
		out.MailboxPending = detail.Row.MailboxPending
		out.OrdersOpen = detail.Row.OrdersOpen
		out.OrdersFailed = detail.Row.OrdersFailed
		out.LastAutonomyAction = strings.TrimSpace(detail.Row.LastAutonomyAction)
		out.LastAutonomySource = strings.TrimSpace(detail.Row.LastAutonomySource)
		out.CurrentOperational = strings.TrimSpace(detail.Row.EstadoOperativo)
		out.CurrentOperationalWhy = strings.TrimSpace(detail.Row.DetalleOperativo)
		out.CurrentTasks = buildAgentActivityCurrentTasks(detail)
	}
	return out
}

func buildAgentActivityCurrentTasks(detail *agentesapp.Detail) []string {
	if detail == nil || detail.Entity == nil || len(detail.Entity.Leases) == 0 {
		return nil
	}
	out := make([]string, 0, len(detail.Entity.Leases))
	for _, lease := range detail.Entity.Leases {
		label := formatAgentActivityLease(lease)
		if strings.TrimSpace(label) == "" {
			continue
		}
		out = append(out, label)
	}
	return out
}

func formatAgentActivityLease(lease agentesapp.WorkLease) string {
	if lease.TaskID <= 0 {
		return ""
	}
	parts := []string{fmt.Sprintf("#%d", lease.TaskID)}
	if state := strings.TrimSpace(string(lease.State)); state != "" {
		parts = append(parts, "["+state+"]")
	}
	if title := strings.TrimSpace(lease.Title); title != "" {
		parts = append(parts, title)
	}
	if module := strings.TrimSpace(lease.Module); module != "" {
		parts = append(parts, "modulo="+module)
	}
	return strings.Join(parts, " ")
}

func summarizeAgentCommitCadence(stats *agentGitActivityStats) (string, string) {
	if stats == nil {
		return "", ""
	}
	pendingFiles := len(stats.PendingFiles)
	pendingLines := stats.PendingAddedLines + stats.PendingDeletedLines
	switch {
	case stats.RecentCommitCount > 0 && pendingFiles == 0:
		return "fluida", fmt.Sprintf("%d commit(s) recientes y sin diff pendiente", stats.RecentCommitCount)
	case stats.RecentCommitCount > 0:
		return "activa", fmt.Sprintf("%d commit(s) recientes; diff pendiente=%d fichero(s) +%d/-%d",
			stats.RecentCommitCount, pendingFiles, stats.PendingAddedLines, stats.PendingDeletedLines)
	case pendingFiles == 0:
		return "sin_diff", "sin diff pendiente ni commits recientes en la ventana"
	case pendingFiles >= 8 || pendingLines >= 400:
		return "atrasada", fmt.Sprintf("diff pendiente grande (%d fichero(s), +%d/-%d) sin commits recientes",
			pendingFiles, stats.PendingAddedLines, stats.PendingDeletedLines)
	default:
		return "sin_checkpoint", fmt.Sprintf("diff pendiente (%d fichero(s), +%d/-%d) sin commits recientes",
			pendingFiles, stats.PendingAddedLines, stats.PendingDeletedLines)
	}
}

func classifyAgentActivityTranscriptSignal(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return "sin_clasificar"
	}
	if signal := strings.TrimSpace(item.Classification); signal != "" {
		return signal
	}
	text := strings.ToLower(strings.TrimSpace(firstNonEmpty(item.NormalizedText, item.Text)))
	stream := strings.ToLower(strings.TrimSpace(item.Stream))
	switch {
	case text == "" && stream == "":
		return "sin_clasificar"
	case strings.Contains(text, "go test") || strings.Contains(text, "npm test") || strings.Contains(text, "pytest") || strings.Contains(text, "tests pass") || strings.Contains(text, "ok  \t"):
		return "tests"
	case strings.Contains(text, "git commit") || strings.Contains(text, "git push") || strings.Contains(text, "git status") || strings.Contains(text, "files changed"):
		return "git_activity"
	case strings.Contains(text, "panic:") || strings.Contains(text, "error:") || strings.Contains(text, "failed") || strings.Contains(text, "traceback"):
		return "runtime_error"
	case strings.Contains(text, "apply_patch") || strings.Contains(text, "update file:") || strings.Contains(text, "add file:") || strings.Contains(text, "diff --git"):
		return "code_change"
	case strings.Contains(text, "progress") || strings.Contains(text, "avance") || strings.Contains(text, "implement") || strings.Contains(text, "validado"):
		return "progress_update"
	case stream == "stderr":
		return "stderr"
	case stream == "stdout":
		return "stdout"
	case stream != "":
		return stream
	default:
		return "sin_clasificar"
	}
}

func resolveAgentActivityProject(detail *agentesapp.Detail, project string) (string, *db.Proyecto, error) {
	project = strings.TrimSpace(project)
	if project != "" {
		entity, err := apiGetProyectoConRutaEfectivaTimeboxed(project, "")
		if err != nil {
			return "", nil, err
		}
		return strings.TrimSpace(entity.Slug), entity, nil
	}
	if detail == nil {
		return "", nil, nil
	}
	candidates := make([]string, 0, 4)
	if detail.Row.Asignacion != nil {
		if slug := strings.TrimSpace(detail.Row.Asignacion.ProyectoSlug); slug != "" {
			candidates = append(candidates, slug)
		}
		if detail.Row.Asignacion.ProyectoID > 0 {
			candidates = append(candidates, strconv.FormatInt(detail.Row.Asignacion.ProyectoID, 10))
		}
	}
	if detail.Row.Sesion != nil {
		if slug := strings.TrimSpace(detail.Row.Sesion.ProyectoSlug); slug != "" {
			candidates = append(candidates, slug)
		}
		if detail.Row.Sesion.ProyectoID != nil && *detail.Row.Sesion.ProyectoID > 0 {
			candidates = append(candidates, strconv.FormatInt(*detail.Row.Sesion.ProyectoID, 10))
		}
	}
	seen := map[string]struct{}{}
	for _, ref := range candidates {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		key := strings.ToLower(ref)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		entity, err := db.GetProyecto(ref)
		if err != nil || entity == nil {
			continue
		}
		return strings.TrimSpace(entity.Slug), entity, nil
	}
	return "", nil, nil
}

func resolveAgentActivityCWD(detail *agentesapp.Detail, project string) string {
	if detail == nil {
		return ""
	}
	project = strings.TrimSpace(project)
	if detail.Row.Agente != nil {
		if path := strings.TrimSpace(resolveAgentActivityWorktree(project, strings.TrimSpace(detail.Row.Agente.Nombre))); path != "" {
			return path
		}
	}
	sessions := make([]*db.Sesion, 0, len(detail.Sesiones)+1)
	if detail.Row.Sesion != nil {
		sessions = append(sessions, detail.Row.Sesion)
	}
	for _, session := range detail.Sesiones {
		if session == nil || session == detail.Row.Sesion {
			continue
		}
		sessions = append(sessions, session)
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessionActivityMoment(sessions[i]).After(sessionActivityMoment(sessions[j]))
	})
	for _, session := range sessions {
		if session == nil || strings.TrimSpace(session.CWD) == "" {
			continue
		}
		if project != "" && !strings.EqualFold(strings.TrimSpace(session.ProyectoSlug), project) {
			continue
		}
		return strings.TrimSpace(session.CWD)
	}
	return ""
}

func sessionActivityMoment(session *db.Sesion) time.Time {
	if session == nil {
		return time.Time{}
	}
	if session.HeartbeatAt != nil {
		return session.HeartbeatAt.UTC()
	}
	return session.Inicio.UTC()
}

func collectAgentGitActivityStats(cwd string, since time.Time) (*agentGitActivityStats, error) {
	return agentGitStatsService.Collect(cwd, since)
}

func imprimirAgenteActividad(report *agentActivityReport) error {
	if report == nil {
		return fmt.Errorf("actividad vacía")
	}
	fmt.Printf("Agente:    %s\n", report.Agent)
	if strings.TrimSpace(report.Project) != "" {
		fmt.Printf("Proyecto:  %s\n", report.Project)
	}
	fmt.Printf("Desde:     %s\n", report.Since.Format("2006-01-02 15:04:05"))
	fmt.Printf("Generado:  %s\n", report.Generated.Format("2006-01-02 15:04:05"))
	if strings.TrimSpace(report.Summary.CurrentOperational) != "" {
		fmt.Printf("Operativo: %s", report.Summary.CurrentOperational)
		if report.Summary.CurrentOperationalWhy != "" {
			fmt.Printf(" — %s", report.Summary.CurrentOperationalWhy)
		}
		fmt.Println()
	}
	fmt.Printf("Tareas:    abiertas=%d bloqueadas=%d mailbox=%d ordenes=%d fallidas=%d\n",
		report.Summary.OpenTasks,
		report.Summary.BlockedTasks,
		report.Summary.MailboxPending,
		report.Summary.OrdersOpen,
		report.Summary.OrdersFailed,
	)
	if len(report.Summary.CurrentTasks) > 0 {
		fmt.Printf("Actuales:\n")
		for _, task := range report.Summary.CurrentTasks {
			fmt.Printf("  - %s\n", task)
		}
	}
	if report.Summary.IntegrationRiskScore > 0 {
		fmt.Printf("Riesgo:    %s · integracion_bloqueada=%d", report.Summary.IntegrationRisk, report.Summary.IntegrationRiskScore)
		if len(report.Summary.IntegrationHighlights) > 0 {
			fmt.Printf(" · causas %s", strings.Join(report.Summary.IntegrationHighlights, " | "))
		}
		fmt.Println()
	}
	if len(report.Summary.AutonomyHighlights) > 0 {
		fmt.Printf("Autonomía: %s\n", strings.Join(report.Summary.AutonomyHighlights, " | "))
	}
	fmt.Printf("Audit:     %d evento(s)\n", report.Summary.AuditEntries)
	for _, item := range topCountPairs(report.Summary.AuditByAction, 5) {
		fmt.Printf("  - %s: %d\n", item.Key, item.Count)
	}
	fmt.Printf("Transcript:%d linea(s)\n", report.Summary.TranscriptEntries)
	for _, item := range topCountPairs(report.Summary.TranscriptBySignal, 5) {
		fmt.Printf("  - %s: %d\n", item.Key, item.Count)
	}
	fmt.Printf("Autonomy:  %d evento(s)\n", report.Summary.AutonomyEvents)
	if report.Summary.LastAutonomyEventAt != nil {
		fmt.Printf("Último:    %s\n", report.Summary.LastAutonomyEventAt.Format("2006-01-02 15:04:05"))
	}
	for _, item := range topCountPairs(report.Summary.AutonomyByKind, 5) {
		fmt.Printf("  - %s: %d\n", item.Key, item.Count)
	}
	if report.Git != nil {
		fmt.Printf("Git:       rama=%s\n", valorVacio(report.Git.Branch))
		if report.Git.RepoRoot != "" {
			fmt.Printf("Repo:      %s\n", report.Git.RepoRoot)
		}
		if head := formatAgentGitHeadCommit(report.Git.HeadCommit); head != "" {
			fmt.Printf("HEAD:      %s\n", head)
		}
		if report.Git.PendingShortStat != "" {
			fmt.Printf("Diff:      %s\n", report.Git.PendingShortStat)
		}
		fmt.Printf("Líneas:    pending +%d/-%d · commits +%d/-%d · commits=%d\n",
			report.Git.PendingAddedLines,
			report.Git.PendingDeletedLines,
			report.Git.CommittedAddedLines,
			report.Git.CommittedDeletedLines,
			report.Git.RecentCommitCount,
		)
		if report.Summary.CommitCadence != "" {
			fmt.Printf("Cadencia:  %s", report.Summary.CommitCadence)
			if report.Summary.CommitCadenceDetail != "" {
				fmt.Printf(" · %s", report.Summary.CommitCadenceDetail)
			}
			fmt.Println()
		}
		if len(report.Git.TouchedFiles) > 0 {
			fmt.Printf("Ficheros tocados (%d):\n", len(report.Git.TouchedFiles))
			for _, file := range report.Git.TouchedFiles {
				fmt.Printf("  - %s\n", file)
			}
		}
	}
	if tests := report.Summary.TranscriptBySignal["tests"]; tests > 0 {
		fmt.Printf("Tests:     señales=%d", tests)
		if report.Summary.LastTestSignalAt != nil {
			fmt.Printf(" · ultimo=%s", report.Summary.LastTestSignalAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}
	if len(report.Autonomy) > 0 {
		fmt.Printf("Eventos autonomía recientes:\n")
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
	return nil
}

func formatAgentGitHeadCommit(commit *gitestadisticasapp.CommitSummary) string {
	if commit == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	headline := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(commit.HashShort),
		strings.TrimSpace(commit.Subject),
	}, " "))
	if headline != "" {
		parts = append(parts, headline)
	}
	if commit.CommittedAt != nil && !commit.CommittedAt.IsZero() {
		parts = append(parts, commit.CommittedAt.UTC().Format("2006-01-02 15:04:05Z07:00"))
	}
	return strings.Join(parts, " · ")
}

func compactAutonomyEventSummaries(items []autonomyEventSummary, recentLimit int) ([]autonomyEventSummary, map[string]int, *time.Time, int) {
	if len(items) == 0 {
		return nil, map[string]int{}, nil, 0
	}
	ordered := append([]autonomyEventSummary(nil), items...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
			return ordered[i].Kind < ordered[j].Kind
		}
		return ordered[i].CreatedAt.After(ordered[j].CreatedAt)
	})
	byKind := make(map[string]int, len(ordered))
	var lastAt *time.Time
	for _, item := range ordered {
		key := strings.TrimSpace(item.Kind)
		if key == "" {
			key = "sin_kind"
		}
		byKind[key]++
		value := item.CreatedAt.UTC()
		lastAt = maxTimePtr(lastAt, &value)
	}
	if recentLimit > 0 && len(ordered) > recentLimit {
		ordered = ordered[:recentLimit]
	}
	return ordered, byKind, lastAt, len(items)
}

func buildAutonomyHighlights(byKind map[string]int, recent []autonomyEventSummary, lastAt *time.Time, maxItems int) []string {
	if maxItems <= 0 {
		maxItems = 3
	}
	highlights := make([]string, 0, maxItems)
	seen := make(map[string]struct{}, maxItems)
	appendHighlight := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		if len(highlights) < maxItems {
			highlights = append(highlights, value)
		}
	}

	for _, pair := range topCountPairs(byKind, maxItems) {
		appendHighlight(fmt.Sprintf("%s=%d", strings.TrimSpace(pair.Key), pair.Count))
	}
	for _, item := range recent {
		if len(highlights) >= maxItems {
			break
		}
		parts := []string{strings.TrimSpace(item.Kind)}
		if target := firstNonEmpty(strings.TrimSpace(item.TargetAgent), strings.TrimSpace(item.Agent)); target != "" {
			parts = append(parts, target)
		}
		if len(item.Artifacts) > 0 {
			parts = append(parts, item.Artifacts[0])
		}
		appendHighlight(strings.Join(parts, " · "))
	}
	if len(highlights) < maxItems && lastAt != nil && !lastAt.IsZero() {
		appendHighlight("último " + lastAt.UTC().Format("2006-01-02 15:04:05"))
	}
	return highlights
}

type countPair struct {
	Key   string
	Count int
}

func topCountPairs(items map[string]int, max int) []countPair {
	pairs := make([]countPair, 0, len(items))
	for key, count := range items {
		pairs = append(pairs, countPair{Key: key, Count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count == pairs[j].Count {
			return pairs[i].Key < pairs[j].Key
		}
		return pairs[i].Count > pairs[j].Count
	})
	if max > 0 && len(pairs) > max {
		return pairs[:max]
	}
	return pairs
}

func mapKeysSorted(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (r *agentActivityReport) MarshalJSON() ([]byte, error) {
	type alias agentActivityReport
	return json.Marshal((*alias)(r))
}
