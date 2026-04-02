package agentesapp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type Store interface {
	RegisterAgent(nombre, rol string) error
	RegisterAgentAuto(proveedor, rol string) (string, error)
	ObserveAgentIdentity(nombre, email, usuario, fuente string, observedAt *time.Time) error
	RetireAgent(nombre string) error
	RehabilitateAgent(nombre string) error
	ResetReanimation(nombre string) error
	DeleteAgent(nombre string) error
	MergeAgents(origen, destino string) (*db.FusionAgentesResultado, error)
	Audit(agente, accion, entidad string, entidadID int64, detalle string)
	GetAgent(nombre string) (*db.Agente, error)
	GetProject(ref string) (*db.Proyecto, error)
	GetConnector(ref string) (*db.Conector, error)
	ListAgents() ([]*db.Agente, error)
	ListAssignments(filtro db.FiltroAsignaciones) ([]*db.Asignacion, error)
	ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error)
	GetLastSession(agente string, proyectoID *int64) (*db.Sesion, error)
	GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error)
	SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error
	ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error)
	ListRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error)
	ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error)
	ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error)
	ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error)
	ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error)
	ListProjectPendingVotes(agente string, proyectoID int64) ([]*db.Propuesta, error)
	ListProjectOpenProposals(proyectoID int64) ([]*db.Propuesta, error)
	GetLatestAgentBudget(agente string) (*db.PresupuestoSesion, *db.Sesion, error)
	ResolveGovernanceCatalog(rol string, proyectoID *int64) (*db.GovernanceCatalog, error)
	ResolveGovernanceCatalogForContext(rol string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error)
	ListRules(rol string) ([]*db.Regla, error)
	ListSkills(rol string) ([]*db.Skill, error)
	ListWorkflows(rol string) ([]*db.Workflow, error)
	ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error)
	ConfigGet(clave string) (string, error)
	PauseAgent(nombre string, minutos int, motivo string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type Row struct {
	Agente         *db.Agente
	Asignacion     *db.Asignacion
	Sesion         *db.Sesion
	Runtime        *db.RuntimeInstance
	Handle         *db.RuntimeHandle
	OrdersOpen     int
	OrdersFailed   int
	MailboxPending int
	MailboxTotal   int
	Checkpoints    int
	LastCheckpoint *db.RuntimeCheckpoint
	OpenTasks      int
}

type Detail struct {
	Row          Row
	Asignaciones []*db.Asignacion
	Sesiones     []*db.Sesion
	Runtimes     []*db.RuntimeInstance
	Handles      []*db.RuntimeHandle
	Transcript   []*db.RuntimeTranscriptEntry
	Orders       []*db.RuntimeOrder
	Mailbox      []*db.RuntimeMailboxMessage
	Checkpoints  []*db.RuntimeCheckpoint
}

type InvestigationMatch struct {
	Transcript    *db.RuntimeTranscriptEntry `json:"transcript"`
	Runtime       *db.RuntimeInstance        `json:"runtime,omitempty"`
	TraceDir      string                     `json:"trace_dir,omitempty"`
	TraceManifest string                     `json:"trace_manifest,omitempty"`
	LogPath       string                     `json:"log_path,omitempty"`
	WorkingDir    string                     `json:"working_dir,omitempty"`
}

type InvestigationAgentResult struct {
	Agente         *db.Agente            `json:"agente"`
	Runtime        *db.RuntimeInstance   `json:"runtime,omitempty"`
	LastCheckpoint *db.RuntimeCheckpoint `json:"last_checkpoint,omitempty"`
	OpenTasks      int                   `json:"open_tasks"`
	Matches        []*InvestigationMatch `json:"matches"`
	handles        []*db.RuntimeHandle
	runtimes       []*db.RuntimeInstance
}

type InvestigationReport struct {
	Query        string                      `json:"query"`
	Proyecto     *db.Proyecto                `json:"proyecto,omitempty"`
	Results      []*InvestigationAgentResult `json:"results"`
	TotalMatches int                         `json:"total_matches"`
}

func (s *Service) RegisterAgent(nombre, rol string) error {
	return s.store.RegisterAgent(strings.TrimSpace(nombre), strings.TrimSpace(rol))
}

func (s *Service) RegisterAgentAuto(proveedor, rol string) (string, error) {
	return s.store.RegisterAgentAuto(strings.TrimSpace(proveedor), strings.TrimSpace(rol))
}

func (s *Service) ObserveAgentIdentity(nombre, email, usuario, fuente string, observedAt *time.Time) error {
	return s.store.ObserveAgentIdentity(strings.TrimSpace(nombre), strings.TrimSpace(email), strings.TrimSpace(usuario), strings.TrimSpace(fuente), observedAt)
}

func (s *Service) GetAgent(nombre string) (*db.Agente, error) {
	return s.store.GetAgent(strings.TrimSpace(nombre))
}

func (s *Service) ListAgents() ([]*db.Agente, error) {
	return s.store.ListAgents()
}

func (s *Service) ApplyStateAction(nombre, accion string) error {
	nombre = strings.TrimSpace(nombre)
	switch strings.TrimSpace(accion) {
	case "retirar":
		return s.retireAgent(nombre)
	case "rehabilitar":
		return s.store.RehabilitateAgent(nombre)
	case "reset-reanimacion":
		return s.store.ResetReanimation(nombre)
	default:
		return fmt.Errorf("acción de agente no soportada: %s", accion)
	}
}

func (s *Service) retireAgent(nombre string) error {
	if err := s.enqueueRetirementPauseIfNeeded(nombre); err != nil {
		return err
	}
	return s.store.RetireAgent(nombre)
}

func (s *Service) enqueueRetirementPauseIfNeeded(nombre string) error {
	handles, err := s.store.ListRuntimeHandles(&nombre)
	if err != nil {
		return err
	}

	var (
		needsPause bool
		proyectoID *int64
	)
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if strings.TrimSpace(handle.Estado) != "activo" {
			continue
		}
		needsPause = true
		if proyectoID == nil && handle.ProyectoID != nil {
			id := *handle.ProyectoID
			proyectoID = &id
		}
	}
	if !needsPause {
		return nil
	}

	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &nombre})
	if err != nil {
		return err
	}
	for _, order := range orders {
		if order == nil {
			continue
		}
		if order.Tipo != "pause" && order.Tipo != "stop" {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			return nil
		}
	}

	payloadJSON, err := json.Marshal(map[string]any{
		"accion": "pause",
		"motivo": "retiro_agente",
		"por":    "server",
	})
	if err != nil {
		return err
	}
	_, err = s.store.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      nombre,
		ProyectoID:  proyectoID,
		Tipo:        "pause",
		PayloadJSON: string(payloadJSON),
	})
	return err
}

func (s *Service) DeleteAgent(nombre, actor string) error {
	nombre = strings.TrimSpace(nombre)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "orquesta"
	}
	if err := s.store.DeleteAgent(nombre); err != nil {
		return err
	}
	s.store.Audit(actor, "purgar_agente", "agente", 0, nombre)
	return nil
}

func (s *Service) MergeAgents(origen, destino string) (*db.FusionAgentesResultado, error) {
	return s.store.MergeAgents(strings.TrimSpace(origen), strings.TrimSpace(destino))
}

func (s *Service) PauseTemporarily(nombre string, minutos int, motivo, accion, entidad, detalle string) error {
	nombre = strings.TrimSpace(nombre)
	motivo = strings.TrimSpace(motivo)
	if nombre == "" || minutos <= 0 || motivo == "" {
		return fmt.Errorf("debes indicar agente, minutos positivos y motivo")
	}
	if err := s.store.PauseAgent(nombre, minutos, motivo); err != nil {
		return err
	}
	if strings.TrimSpace(detalle) == "" {
		detalle = fmt.Sprintf("Bloqueado %d min por: %s", minutos, motivo)
	}
	s.store.Audit(nombre, valueOrFallback(strings.TrimSpace(accion), "pausa_externa"), valueOrFallback(strings.TrimSpace(entidad), "agente"), 0, strings.TrimSpace(detalle))
	return nil
}

func (s *Service) BuildPanelRows() ([]Row, error) {
	agentes, err := s.ListAgents()
	if err != nil {
		return nil, err
	}
	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{})
	if err != nil {
		return nil, err
	}
	activa := true
	sesiones, err := s.store.ListInspectionSessions(db.FiltroSesionesInspeccion{Activa: &activa})
	if err != nil {
		return nil, err
	}
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{})
	if err != nil {
		return nil, err
	}
	handles, err := s.store.ListRuntimeHandles(nil)
	if err != nil {
		return nil, err
	}
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{})
	if err != nil {
		return nil, err
	}
	mailbox, err := s.store.ListRuntimeMailbox(db.FiltroRuntimeMailbox{})
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{})
	if err != nil {
		return nil, err
	}
	tareas, err := s.store.ListTasks(db.FiltroTareas{})
	if err != nil {
		return nil, err
	}

	asignacionPorAgente := map[string]*db.Asignacion{}
	for _, asignacion := range asignaciones {
		if asignacion == nil || asignacion.Estado != db.AsignacionActiva {
			continue
		}
		if _, ok := asignacionPorAgente[asignacion.Agente]; !ok {
			asignacionPorAgente[asignacion.Agente] = asignacion
		}
	}

	sesionPorAgente := map[string]*db.Sesion{}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		if _, ok := sesionPorAgente[sesion.Agente]; !ok {
			sesionPorAgente[sesion.Agente] = sesion
		}
	}

	runtimePorAgente := map[string]*db.RuntimeInstance{}
	for _, runtime := range runtimes {
		if runtime == nil {
			continue
		}
		actual := runtimePorAgente[runtime.Agente]
		if actual == nil || runtimeMoment(runtime).After(runtimeMoment(actual)) {
			runtimePorAgente[runtime.Agente] = runtime
		}
	}

	handlePorAgente := map[string]*db.RuntimeHandle{}
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		actual := handlePorAgente[handle.Agente]
		if actual == nil || runtimeHandleMoment(handle).After(runtimeHandleMoment(actual)) {
			handlePorAgente[handle.Agente] = handle
		}
	}

	type orderCounters struct {
		Open   int
		Failed int
	}
	ordersPorAgente := map[string]orderCounters{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		stats := ordersPorAgente[order.Agente]
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			stats.Open++
		case "fallida":
			stats.Failed++
		}
		if strings.TrimSpace(order.ErrorText) != "" {
			stats.Failed++
		}
		ordersPorAgente[order.Agente] = stats
	}

	type mailboxCounters struct {
		Pending int
		Total   int
	}
	mailboxPorAgente := map[string]mailboxCounters{}
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		for _, agente := range []string{msg.ToAgente, msg.FromAgente} {
			if strings.TrimSpace(agente) == "" {
				continue
			}
			stats := mailboxPorAgente[agente]
			stats.Total++
			if strings.TrimSpace(msg.Estado) == "pendiente" {
				stats.Pending++
			}
			mailboxPorAgente[agente] = stats
		}
	}

	checkpointTotalPorAgente := map[string]int{}
	lastCheckpointPorAgente := map[string]*db.RuntimeCheckpoint{}
	for _, checkpoint := range checkpoints {
		if checkpoint == nil {
			continue
		}
		checkpointTotalPorAgente[checkpoint.Agente]++
		if _, ok := lastCheckpointPorAgente[checkpoint.Agente]; !ok {
			lastCheckpointPorAgente[checkpoint.Agente] = checkpoint
		}
	}

	openTasksPorAgente := map[string]int{}
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada:
			continue
		}
		openTasksPorAgente[*tarea.Agente]++
	}

	rows := make([]Row, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		orderStats := ordersPorAgente[agente.Nombre]
		mailboxStats := mailboxPorAgente[agente.Nombre]
		rows = append(rows, Row{
			Agente:         agente,
			Asignacion:     asignacionPorAgente[agente.Nombre],
			Sesion:         sesionPorAgente[agente.Nombre],
			Runtime:        runtimePorAgente[agente.Nombre],
			Handle:         handlePorAgente[agente.Nombre],
			OrdersOpen:     orderStats.Open,
			OrdersFailed:   orderStats.Failed,
			MailboxPending: mailboxStats.Pending,
			MailboxTotal:   mailboxStats.Total,
			Checkpoints:    checkpointTotalPorAgente[agente.Nombre],
			LastCheckpoint: lastCheckpointPorAgente[agente.Nombre],
			OpenTasks:      openTasksPorAgente[agente.Nombre],
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].Agente.Nombre) < strings.ToLower(rows[j].Agente.Nombre)
	})
	return rows, nil
}

func (s *Service) BuildDetail(nombre string) (*Detail, error) {
	nombre = strings.TrimSpace(nombre)
	agente, err := s.store.GetAgent(nombre)
	if err != nil {
		return nil, err
	}
	rows, err := s.BuildPanelRows()
	if err != nil {
		return nil, err
	}
	row := Row{Agente: agente}
	for _, item := range rows {
		if item.Agente != nil && item.Agente.Nombre == nombre {
			row = item
			break
		}
	}

	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	sesiones, err := s.store.ListInspectionSessions(db.FiltroSesionesInspeccion{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	handles, err := s.store.ListRuntimeHandles(&nombre)
	if err != nil {
		return nil, err
	}
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	transcript, err := s.store.ListRuntimeTranscript(db.FiltroRuntimeTranscript{Agente: &nombre, Limit: 80})
	if err != nil {
		return nil, err
	}
	mailbox, err := s.listMailboxForAgent(nombre)
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &nombre, Limit: 20})
	if err != nil {
		return nil, err
	}

	return &Detail{
		Row:          row,
		Asignaciones: asignaciones,
		Sesiones:     sesiones,
		Runtimes:     runtimes,
		Handles:      handles,
		Transcript:   transcript,
		Orders:       orders,
		Mailbox:      mailbox,
		Checkpoints:  checkpoints,
	}, nil
}

func (s *Service) Investigate(query, projectRef string, limit int) (*InvestigationReport, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta obligatoria")
	}
	if limit <= 0 {
		limit = 25
	}

	var (
		project   *db.Proyecto
		projectID *int64
		err       error
	)
	projectRef = strings.TrimSpace(projectRef)
	if projectRef != "" {
		project, err = s.store.GetProject(projectRef)
		if err != nil {
			return nil, err
		}
		if project != nil {
			projectID = &project.ID
		}
	}

	transcript, err := s.store.ListRuntimeTranscript(db.FiltroRuntimeTranscript{
		ProyectoID: projectID,
		Query:      &query,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	report := &InvestigationReport{
		Query:        query,
		Proyecto:     project,
		Results:      []*InvestigationAgentResult{},
		TotalMatches: len(transcript),
	}
	if len(transcript) == 0 {
		return report, nil
	}

	byAgent := map[string]*InvestigationAgentResult{}
	agentOrder := make([]string, 0, len(transcript))
	for _, item := range transcript {
		if item == nil {
			continue
		}
		agente := strings.TrimSpace(item.Agente)
		if agente == "" {
			continue
		}
		result := byAgent[agente]
		if result == nil {
			result, err = s.buildInvestigationAgentResult(agente, projectID)
			if err != nil {
				return nil, err
			}
			byAgent[agente] = result
			agentOrder = append(agentOrder, agente)
		}
		match := &InvestigationMatch{Transcript: item}
		match.Runtime = investigationRuntimeForMatch(result, item.RuntimeID)
		if item.HandleID != nil {
			traceDir, traceManifest, logPath, workingDir := investigationTraceInfo(result, *item.HandleID)
			match.TraceDir = traceDir
			match.TraceManifest = traceManifest
			match.LogPath = logPath
			match.WorkingDir = firstNonEmptyAgent(strings.TrimSpace(workingDir), runtimeWorkingDir(match.Runtime))
		} else {
			match.WorkingDir = runtimeWorkingDir(match.Runtime)
		}
		result.Matches = append(result.Matches, match)
	}

	for _, agente := range agentOrder {
		if item := byAgent[agente]; item != nil {
			report.Results = append(report.Results, item)
		}
	}
	sort.Slice(report.Results, func(i, j int) bool {
		left := latestInvestigationMoment(report.Results[i])
		right := latestInvestigationMoment(report.Results[j])
		if !left.Equal(right) {
			return left.After(right)
		}
		leftName := ""
		rightName := ""
		if report.Results[i].Agente != nil {
			leftName = strings.ToLower(strings.TrimSpace(report.Results[i].Agente.Nombre))
		}
		if report.Results[j].Agente != nil {
			rightName = strings.ToLower(strings.TrimSpace(report.Results[j].Agente.Nombre))
		}
		return leftName < rightName
	})
	return report, nil
}

func (s *Service) buildInvestigationAgentResult(nombre string, projectID *int64) (*InvestigationAgentResult, error) {
	agente, err := s.store.GetAgent(nombre)
	if err != nil {
		return nil, err
	}
	if agente == nil {
		agente = &db.Agente{Nombre: nombre}
	}
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &nombre, ProyectoID: projectID})
	if err != nil {
		return nil, err
	}
	handles, err := s.store.ListRuntimeHandles(&nombre)
	if err != nil {
		return nil, err
	}
	tasks, err := s.store.ListTasks(db.FiltroTareas{Agente: &nombre, ProyectoID: projectID})
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &nombre, ProyectoID: projectID, Limit: 5})
	if err != nil {
		return nil, err
	}
	return &InvestigationAgentResult{
		Agente:         agente,
		Runtime:        latestRuntimeForInvestigation(runtimes),
		LastCheckpoint: latestCheckpointForInvestigation(checkpoints),
		OpenTasks:      openTasksForInvestigation(tasks),
		Matches:        []*InvestigationMatch{},
		handles:        handles,
		runtimes:       runtimes,
	}, nil
}

func latestInvestigationMoment(result *InvestigationAgentResult) time.Time {
	if result == nil {
		return time.Time{}
	}
	var latest time.Time
	for _, item := range result.Matches {
		if item == nil || item.Transcript == nil {
			continue
		}
		if item.Transcript.CreatedAt.After(latest) {
			latest = item.Transcript.CreatedAt
		}
	}
	return latest
}

func latestRuntimeForInvestigation(items []*db.RuntimeInstance) *db.RuntimeInstance {
	var picked *db.RuntimeInstance
	for _, item := range items {
		if item == nil {
			continue
		}
		if picked == nil || runtimeMoment(item).After(runtimeMoment(picked)) {
			picked = item
		}
	}
	return picked
}

func latestCheckpointForInvestigation(items []*db.RuntimeCheckpoint) *db.RuntimeCheckpoint {
	var picked *db.RuntimeCheckpoint
	for _, item := range items {
		if item == nil {
			continue
		}
		if picked == nil || item.CreatedAt.After(picked.CreatedAt) {
			picked = item
		}
	}
	return picked
}

func openTasksForInvestigation(items []*db.Tarea) int {
	total := 0
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Estado {
		case db.TareaCompletada, db.TareaCancelada, db.TareaBacklog:
			continue
		default:
			total++
		}
	}
	return total
}

func investigationTraceInfo(result *InvestigationAgentResult, handleID int64) (string, string, string, string) {
	if result == nil || handleID <= 0 {
		return "", "", "", ""
	}
	for _, handle := range result.handles {
		if handle == nil || handle.ID != handleID {
			continue
		}
		meta := map[string]any{}
		if strings.TrimSpace(handle.MetadataJSON) != "" {
			_ = json.Unmarshal([]byte(handle.MetadataJSON), &meta)
		}
		return stringMapValueAgent(meta, "trace_dir"),
			stringMapValueAgent(meta, "trace_manifest"),
			stringMapValueAgent(meta, "log_path"),
			stringMapValueAgent(meta, "working_dir")
	}
	return "", "", "", ""
}

func investigationRuntimeForMatch(result *InvestigationAgentResult, runtimeID int64) *db.RuntimeInstance {
	if result == nil || runtimeID <= 0 {
		return result.Runtime
	}
	for _, item := range result.runtimes {
		if item != nil && item.ID == runtimeID {
			return item
		}
	}
	return result.Runtime
}

func runtimeWorkingDir(runtime *db.RuntimeInstance) string {
	if runtime == nil {
		return ""
	}
	return strings.TrimSpace(runtime.CWD)
}

func firstNonEmptyAgent(items ...string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func stringMapValueAgent(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	value, ok := raw[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func (s *Service) listMailboxForAgent(nombre string) ([]*db.RuntimeMailboxMessage, error) {
	toAgente := nombre
	inbox, err := s.store.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &toAgente})
	if err != nil {
		return nil, err
	}
	fromAgente := nombre
	outbox, err := s.store.ListRuntimeMailbox(db.FiltroRuntimeMailbox{FromAgente: &fromAgente})
	if err != nil {
		return nil, err
	}
	merged := make([]*db.RuntimeMailboxMessage, 0, len(inbox)+len(outbox))
	seen := map[int64]struct{}{}
	for _, set := range [][]*db.RuntimeMailboxMessage{inbox, outbox} {
		for _, msg := range set {
			if msg == nil {
				continue
			}
			if _, ok := seen[msg.ID]; ok {
				continue
			}
			seen[msg.ID] = struct{}{}
			merged = append(merged, msg)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].ID > merged[j].ID
	})
	return merged, nil
}

func runtimeMoment(runtime *db.RuntimeInstance) time.Time {
	if runtime == nil {
		return time.Time{}
	}
	for _, value := range []*time.Time{runtime.UltimaActividadAt, runtime.LastHeartbeatAt, runtime.LastEventAt} {
		if value != nil && !value.IsZero() {
			return *value
		}
	}
	if !runtime.UpdatedAt.IsZero() {
		return runtime.UpdatedAt
	}
	return runtime.CreatedAt
}

func runtimeHandleMoment(handle *db.RuntimeHandle) time.Time {
	if handle == nil {
		return time.Time{}
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		return *handle.LastSeenAt
	}
	if !handle.UpdatedAt.IsZero() {
		return handle.UpdatedAt
	}
	return handle.CreatedAt
}

func valueOrFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

type Repository struct{}

func (Repository) RegisterAgent(nombre, rol string) error {
	return db.RegistrarAgente(nombre, rol)
}

func (Repository) RegisterAgentAuto(proveedor, rol string) (string, error) {
	return db.RegistrarAgenteAuto(proveedor, rol)
}

func (Repository) ObserveAgentIdentity(nombre, email, usuario, fuente string, observedAt *time.Time) error {
	return db.UpsertAgenteIdentidadObservada(nombre, email, usuario, fuente, observedAt)
}

func (Repository) RetireAgent(nombre string) error {
	return db.RetirarAgente(nombre)
}

func (Repository) RehabilitateAgent(nombre string) error {
	return db.RehabilitarAgente(nombre)
}

func (Repository) ResetReanimation(nombre string) error {
	return db.ResetReanimacion(nombre)
}

func (Repository) DeleteAgent(nombre string) error {
	return db.EliminarAgente(nombre)
}

func (Repository) MergeAgents(origen, destino string) (*db.FusionAgentesResultado, error) {
	return db.FusionarAgentes(origen, destino)
}

func (Repository) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(agente, accion, entidad, entidadID, detalle)
}

func (Repository) GetAgent(nombre string) (*db.Agente, error) {
	return db.GetAgente(nombre)
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) GetConnector(ref string) (*db.Conector, error) {
	return db.GetConector(ref)
}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}

func (Repository) ListAssignments(filtro db.FiltroAsignaciones) ([]*db.Asignacion, error) {
	return db.ListarAsignaciones(filtro)
}

func (Repository) ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error) {
	return db.ListarSesionesInspeccion(filtro)
}

func (Repository) GetLastSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	return db.ObtenerUltimaSesion(agente, proyectoID)
}

func (Repository) GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	return db.GetSesionActiva(agente, proyectoID)
}

func (Repository) SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error {
	return db.GuardarSesionActiva(agente, proyectoID, upd)
}

func (Repository) ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	return db.ListarRuntimes(filtro)
}

func (Repository) ListRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandles(agente)
}

func (Repository) ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
	return db.ListarRuntimeTranscript(filtro)
}

func (Repository) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return db.ListarRuntimeOrders(filtro)
}

func (Repository) EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return db.EncolarRuntimeOrder(order)
}

func (Repository) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return db.ListarRuntimeMailbox(filtro)
}

func (Repository) ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	return db.ListarRuntimeCheckpoints(filtro)
}

func (Repository) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	return db.ListarTareas(filtro)
}

func (Repository) ListProjectPendingVotes(agente string, proyectoID int64) ([]*db.Propuesta, error) {
	return db.PropuestasPendientesVotoProyecto(agente, &proyectoID)
}

func (Repository) ListProjectOpenProposals(proyectoID int64) ([]*db.Propuesta, error) {
	estado := db.PropuestaAbierta
	return db.ListarPropuestas(&estado, &proyectoID)
}

func (Repository) GetLatestAgentBudget(agente string) (*db.PresupuestoSesion, *db.Sesion, error) {
	return db.UltimoPresupuestoAgente(agente)
}

func (Repository) ResolveGovernanceCatalog(rol string, proyectoID *int64) (*db.GovernanceCatalog, error) {
	return db.ResolveGovernanceCatalog(rol, proyectoID)
}

func (Repository) ResolveGovernanceCatalogForContext(rol string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error) {
	return db.ResolveGovernanceCatalogForContext(rol, proyectoID, agente)
}

func (Repository) ListRules(rol string) ([]*db.Regla, error) {
	return db.GetReglasAgente(rol)
}

func (Repository) ListSkills(rol string) ([]*db.Skill, error) {
	return db.GetSkillsAgente(rol)
}

func (Repository) ListWorkflows(rol string) ([]*db.Workflow, error) {
	return db.GetWorkflowsAgente(rol)
}

func (Repository) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return db.ListarEntidadesMemoria(filtro)
}

func (Repository) ConfigGet(clave string) (string, error) {
	return db.ConfigGet(clave)
}

func (Repository) PauseAgent(nombre string, minutos int, motivo string) error {
	return db.PausarAgente(nombre, minutos, motivo)
}
