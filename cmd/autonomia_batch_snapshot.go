package cmd

import (
	"database/sql"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/autonomiapolicy"
	"orquesta/db"
)

type autonomiaBudgetPauseDecision struct {
	shouldPause bool
	reason      string
}

type autonomiaSupervisorCandidate struct {
	agente    *db.Agente
	preferred bool
	activo    bool
	score     int
}

type autonomiaBatchSnapshot struct {
	sesiones []*db.Sesion

	agentesByName              map[string]*db.Agente
	operationalStateByAgent    map[string]string
	operationalDetailByAgent   map[string]string
	operationalStateResolved   map[string]struct{}
	hotHandlesByAgentProject   map[string]*db.RuntimeHandle
	asignacionesByAgent        map[string][]*db.Asignacion
	asignacionesByProject      map[int64][]*db.Asignacion
	tareasByAgentProject       map[string][]*db.Tarea
	pendingVotesByAgentProject map[string][]*db.Propuesta
	openProposalsByProject     map[int64][]*db.Propuesta
	politicasByProject         map[int64]*db.ProyectoAutonomia
	proyectosByID             map[int64]*db.Proyecto
	proyectosLoaded           map[int64]struct{}
	activeProjectByAgent       map[string]int64
	activeHandleByAgentProject map[string]bool
	supervisorByProject        map[int64]*db.Agente
	supervisorLoaded           map[int64]struct{}
	pauseByAgent               map[string]autonomiaBudgetPauseDecision
	bloqueosPorTarea           map[int64]db.ResumenBloqueo
	bloqueosLoaded             bool
}

var autonomiaOperationalStateResolver = func(agente string) (string, string, error) {
	return agentesService.OperationalStateForAgent(agente)
}

func newAutonomiaBatchSnapshot(sesiones []*db.Sesion) (*autonomiaBatchSnapshot, error) {
	agentNames := preloadAutonomiaBatchAgentNames(sesiones)
	preloadAgentsStart := time.Now()
	agentesByName, err := db.ListarAgentesConSesionOperativa(agentNames, sesiones)
	if err != nil {
		return nil, err
	}
	autonomiaTickDebugf("snapshot preload_agentes agentes=%d duration=%s", len(agentesByName), time.Since(preloadAgentsStart).Round(time.Millisecond))
	hotHandlesStart := time.Now()
	hotHandles, err := db.ListarRuntimeHandlesActivosOperativosRecientes()
	if err != nil {
		return nil, err
	}
	autonomiaTickDebugf("snapshot preload_handles handles=%d duration=%s", len(hotHandles), time.Since(hotHandlesStart).Round(time.Millisecond))
	snapshot := &autonomiaBatchSnapshot{
		sesiones:                   sesiones,
		agentesByName:              make(map[string]*db.Agente, len(agentesByName)),
		operationalStateByAgent:    map[string]string{},
		operationalDetailByAgent:   map[string]string{},
		operationalStateResolved:   map[string]struct{}{},
		hotHandlesByAgentProject:   hotHandles,
		asignacionesByAgent:        map[string][]*db.Asignacion{},
		asignacionesByProject:      map[int64][]*db.Asignacion{},
		tareasByAgentProject:       map[string][]*db.Tarea{},
		pendingVotesByAgentProject: map[string][]*db.Propuesta{},
		openProposalsByProject:     map[int64][]*db.Propuesta{},
		politicasByProject:         map[int64]*db.ProyectoAutonomia{},
		proyectosByID:              map[int64]*db.Proyecto{},
		proyectosLoaded:            map[int64]struct{}{},
		activeProjectByAgent:       map[string]int64{},
		activeHandleByAgentProject: map[string]bool{},
		supervisorByProject:        map[int64]*db.Agente{},
		supervisorLoaded:           map[int64]struct{}{},
		pauseByAgent:               map[string]autonomiaBudgetPauseDecision{},
		bloqueosPorTarea:           map[int64]db.ResumenBloqueo{},
	}
	for key, agente := range agentesByName {
		if agente == nil {
			continue
		}
		snapshot.agentesByName[key] = agente
	}
	return snapshot, nil
}

func preloadAutonomiaBatchAgentNames(sesiones []*db.Sesion) []string {
	seen := make(map[string]struct{}, len(sesiones))
	agentes := make([]string, 0, len(sesiones))
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		nombre := strings.TrimSpace(sesion.Agente)
		if nombre == "" {
			continue
		}
		key := strings.ToLower(nombre)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		agentes = append(agentes, nombre)
	}
	return agentes
}

func (s *autonomiaBatchSnapshot) invalidateAgent(agente string) {
	key := strings.ToLower(strings.TrimSpace(agente))
	if key == "" {
		return
	}
	delete(s.agentesByName, key)
	delete(s.operationalStateByAgent, key)
	delete(s.operationalDetailByAgent, key)
	delete(s.operationalStateResolved, key)
	delete(s.asignacionesByAgent, key)
	delete(s.activeProjectByAgent, key)
	delete(s.pauseByAgent, key)
	for cacheKey := range s.tareasByAgentProject {
		if strings.HasPrefix(cacheKey, key+"|") {
			delete(s.tareasByAgentProject, cacheKey)
		}
	}
	for cacheKey := range s.pendingVotesByAgentProject {
		if strings.HasPrefix(cacheKey, key+"|") {
			delete(s.pendingVotesByAgentProject, cacheKey)
		}
	}
}

func (s *autonomiaBatchSnapshot) invalidateProject(proyectoID int64) {
	if s == nil || proyectoID <= 0 {
		return
	}
	delete(s.asignacionesByProject, proyectoID)
	delete(s.openProposalsByProject, proyectoID)
	delete(s.politicasByProject, proyectoID)
	delete(s.proyectosByID, proyectoID)
	delete(s.proyectosLoaded, proyectoID)
	delete(s.supervisorByProject, proyectoID)
	delete(s.supervisorLoaded, proyectoID)
	for cacheKey := range s.tareasByAgentProject {
		if strings.HasSuffix(cacheKey, "|"+int64CacheKey(proyectoID)) {
			delete(s.tareasByAgentProject, cacheKey)
		}
	}
	for cacheKey := range s.pendingVotesByAgentProject {
		if strings.HasSuffix(cacheKey, "|"+int64CacheKey(proyectoID)) {
			delete(s.pendingVotesByAgentProject, cacheKey)
		}
	}
	for cacheKey := range s.activeHandleByAgentProject {
		if strings.HasSuffix(cacheKey, "|"+int64CacheKey(proyectoID)) {
			delete(s.activeHandleByAgentProject, cacheKey)
		}
	}
	s.bloqueosLoaded = false
	s.bloqueosPorTarea = map[int64]db.ResumenBloqueo{}
}

func (s *autonomiaBatchSnapshot) assignment(agente string, proyectoID int64) (bool, string, error) {
	asignaciones, err := s.activeAssignmentsByAgent(agente)
	if err != nil {
		return false, "", err
	}
	if len(asignaciones) == 0 {
		return false, "", nil
	}
	for _, asignacion := range asignaciones {
		if asignacion != nil && asignacion.ProyectoID == proyectoID {
			return true, asignacion.ProyectoSlug, nil
		}
	}
	return false, asignaciones[0].ProyectoSlug, nil
}

func (s *autonomiaBatchSnapshot) activeAssignmentsByAgent(agente string) ([]*db.Asignacion, error) {
	key := strings.ToLower(strings.TrimSpace(agente))
	if key == "" {
		return nil, nil
	}
	asignaciones, ok := s.asignacionesByAgent[key]
	if !ok {
		estado := db.AsignacionActiva
		list, err := db.ListarAsignaciones(db.FiltroAsignaciones{
			Agente: &agente,
			Estado: &estado,
		})
		if err != nil {
			return nil, err
		}
		asignaciones = list
		s.asignacionesByAgent[key] = list
	}
	return asignaciones, nil
}

func (s *autonomiaBatchSnapshot) tasks(agente string, proyectoID int64) ([]*db.Tarea, error) {
	key := agentProjectCacheKey(agente, proyectoID)
	if list, ok := s.tareasByAgentProject[key]; ok {
		return list, nil
	}
	list, err := db.ListarTareas(db.FiltroTareas{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		return nil, err
	}
	s.tareasByAgentProject[key] = list
	return list, nil
}

func (s *autonomiaBatchSnapshot) pendingProjectVotes(agente string, proyectoID int64) ([]*db.Propuesta, error) {
	key := agentProjectCacheKey(agente, proyectoID)
	if list, ok := s.pendingVotesByAgentProject[key]; ok {
		return list, nil
	}
	list, err := db.PropuestasPendientesVotoProyecto(strings.TrimSpace(agente), &proyectoID)
	if err != nil {
		return nil, err
	}
	s.pendingVotesByAgentProject[key] = list
	return list, nil
}

func (s *autonomiaBatchSnapshot) openProjectProposals(proyectoID int64) ([]*db.Propuesta, error) {
	if list, ok := s.openProposalsByProject[proyectoID]; ok {
		return list, nil
	}
	estadoAbierta := db.PropuestaAbierta
	list, err := db.ListarPropuestas(&estadoAbierta, &proyectoID)
	if err != nil {
		return nil, err
	}
	s.openProposalsByProject[proyectoID] = list
	return list, nil
}

func (s *autonomiaBatchSnapshot) agent(agente string) (*db.Agente, error) {
	key := strings.ToLower(strings.TrimSpace(agente))
	if key == "" {
		return nil, sql.ErrNoRows
	}
	if item, ok := s.agentesByName[key]; ok && item != nil {
		return item, nil
	}
	item, err := db.GetAgente(strings.TrimSpace(agente))
	if err != nil {
		return nil, err
	}
	s.agentesByName[key] = item
	return item, nil
}

func (s *autonomiaBatchSnapshot) budgetPause(agente string) (bool, string, error) {
	key := strings.ToLower(strings.TrimSpace(agente))
	if key == "" {
		return false, "", nil
	}
	if decision, ok := s.pauseByAgent[key]; ok {
		return decision.shouldPause, decision.reason, nil
	}
	if cached, ok := s.agentesByName[key]; ok && cached != nil {
		shouldPause, reason := agenteDebePausarPorPresupuestoVisible(cached)
		s.pauseByAgent[key] = autonomiaBudgetPauseDecision{
			shouldPause: shouldPause,
			reason:      reason,
		}
		return shouldPause, reason, nil
	}
	shouldPause, reason, err := agenteDebePausarPorPresupuesto(strings.TrimSpace(agente))
	if err != nil {
		return false, "", err
	}
	s.pauseByAgent[key] = autonomiaBudgetPauseDecision{
		shouldPause: shouldPause,
		reason:      reason,
	}
	return shouldPause, reason, nil
}

func (s *autonomiaBatchSnapshot) operationalState(agente string) (string, string, error) {
	key := strings.ToLower(strings.TrimSpace(agente))
	if key == "" {
		return "", "", nil
	}
	if s.operationalStateByAgent == nil {
		s.operationalStateByAgent = map[string]string{}
	}
	if s.operationalDetailByAgent == nil {
		s.operationalDetailByAgent = map[string]string{}
	}
	if s.operationalStateResolved == nil {
		s.operationalStateResolved = map[string]struct{}{}
	}
	if _, ok := s.operationalStateResolved[key]; ok {
		return s.operationalStateByAgent[key], s.operationalDetailByAgent[key], nil
	}
	estado, detalle, err := autonomiaOperationalStateResolver(strings.TrimSpace(agente))
	if err != nil {
		return "", "", err
	}
	s.operationalStateByAgent[key] = strings.TrimSpace(estado)
	s.operationalDetailByAgent[key] = strings.TrimSpace(detalle)
	s.operationalStateResolved[key] = struct{}{}
	return s.operationalStateByAgent[key], s.operationalDetailByAgent[key], nil
}

func (s *autonomiaBatchSnapshot) supervisorOperativo(proyectoID int64, agente string) (bool, *db.Agente, error) {
	supervisor, err := s.selectSupervisor(proyectoID)
	if err != nil || supervisor == nil {
		return false, supervisor, err
	}
	return strings.EqualFold(strings.TrimSpace(supervisor.Nombre), strings.TrimSpace(agente)), supervisor, nil
}

func (s *autonomiaBatchSnapshot) selectSupervisor(proyectoID int64) (*db.Agente, error) {
	if proyectoID <= 0 {
		return nil, nil
	}
	if _, ok := s.supervisorLoaded[proyectoID]; ok {
		return s.supervisorByProject[proyectoID], nil
	}
	policy, err := s.projectPolicy(proyectoID)
	if err != nil || policy == nil || !policy.Enabled {
		s.supervisorLoaded[proyectoID] = struct{}{}
		return nil, err
	}
	preferred := strings.TrimSpace(policy.SupervisorAgente)
	if preferred != "" {
		agente, err := s.agent(preferred)
		if err == nil && s.autonomiaDisponible(agente, proyectoID) {
			s.supervisorByProject[proyectoID] = agente
			s.supervisorLoaded[proyectoID] = struct{}{}
			return agente, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}
	seen := map[string]struct{}{}
	candidatos := make([]autonomiaSupervisorCandidate, 0, 8)
	add := func(nombre string) error {
		nombre = strings.TrimSpace(nombre)
		if nombre == "" {
			return nil
		}
		agente, err := s.agent(nombre)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		if !s.autonomiaDisponible(agente, proyectoID) {
			return nil
		}
		key := strings.ToLower(strings.TrimSpace(agente.Nombre))
		if _, ok := seen[key]; ok {
			return nil
		}
		seen[key] = struct{}{}
		activo, err := s.autonomiaActivoEnProyecto(strings.TrimSpace(agente.Nombre), proyectoID, agente)
		if err != nil {
			return err
		}
		candidatos = append(candidatos, autonomiaSupervisorCandidate{
			agente:    agente,
			preferred: preferred != "" && strings.EqualFold(strings.TrimSpace(agente.Nombre), preferred),
			activo:    activo,
			score:     autonomiapolicy.SupervisorRoleScore(agente.Rol),
		})
		return nil
	}
	if err := add(preferred); err != nil {
		return nil, err
	}
	for _, sesion := range s.sesiones {
		if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID != proyectoID {
			continue
		}
		if err := add(sesion.Agente); err != nil {
			return nil, err
		}
	}
	asignaciones, err := s.projectAssignments(proyectoID)
	if err != nil {
		return nil, err
	}
	for _, asignacion := range asignaciones {
		if asignacion == nil {
			continue
		}
		if err := add(asignacion.Agente); err != nil {
			return nil, err
		}
	}
	sort.Slice(candidatos, func(i, j int) bool {
		if candidatos[i].preferred != candidatos[j].preferred {
			return candidatos[i].preferred
		}
		if candidatos[i].activo != candidatos[j].activo {
			return candidatos[i].activo
		}
		if candidatos[i].score != candidatos[j].score {
			return candidatos[i].score < candidatos[j].score
		}
		return strings.TrimSpace(candidatos[i].agente.Nombre) < strings.TrimSpace(candidatos[j].agente.Nombre)
	})
	var supervisor *db.Agente
	if len(candidatos) > 0 {
		supervisor = candidatos[0].agente
	}
	s.supervisorByProject[proyectoID] = supervisor
	s.supervisorLoaded[proyectoID] = struct{}{}
	return supervisor, nil
}

func (s *autonomiaBatchSnapshot) projectPolicy(proyectoID int64) (*db.ProyectoAutonomia, error) {
	if policy, ok := s.politicasByProject[proyectoID]; ok {
		return policy, nil
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		return nil, err
	}
	s.politicasByProject[proyectoID] = policy
	return policy, nil
}

func (s *autonomiaBatchSnapshot) project(proyectoID int64) (*db.Proyecto, error) {
	if s == nil || proyectoID <= 0 {
		return nil, nil
	}
	if _, ok := s.proyectosLoaded[proyectoID]; ok {
		return s.proyectosByID[proyectoID], nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		return nil, err
	}
	s.proyectosByID[proyectoID] = proyecto
	s.proyectosLoaded[proyectoID] = struct{}{}
	return proyecto, nil
}

func (s *autonomiaBatchSnapshot) projectAssignments(proyectoID int64) ([]*db.Asignacion, error) {
	if list, ok := s.asignacionesByProject[proyectoID]; ok {
		return list, nil
	}
	estado := db.AsignacionActiva
	list, err := db.ListarAsignaciones(db.FiltroAsignaciones{
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return nil, err
	}
	s.asignacionesByProject[proyectoID] = list
	return list, nil
}

func (s *autonomiaBatchSnapshot) autonomiaDisponible(agente *db.Agente, proyectoID int64) bool {
	if agente == nil || !agente.Habilitado || proyectoID <= 0 {
		return false
	}
	estadoCuota := strings.ToLower(strings.TrimSpace(agente.EstadoCuota))
	if estadoCuota != "" && estadoCuota != "activo" {
		return false
	}
	proyectoActivoID, err := s.activeProject(strings.TrimSpace(agente.Nombre))
	if err != nil {
		return false
	}
	return proyectoActivoID == 0 || proyectoActivoID == proyectoID
}

func (s *autonomiaBatchSnapshot) activeProject(agente string) (int64, error) {
	key := strings.ToLower(strings.TrimSpace(agente))
	if key == "" {
		return 0, nil
	}
	if proyectoID, ok := s.activeProjectByAgent[key]; ok {
		return proyectoID, nil
	}
	proyectoID, err := db.ObtenerProyectoActivoAgente(strings.TrimSpace(agente))
	if err != nil {
		return 0, err
	}
	s.activeProjectByAgent[key] = proyectoID
	return proyectoID, nil
}

func (s *autonomiaBatchSnapshot) bloqueoSummaryMap() (map[int64]db.ResumenBloqueo, error) {
	if s == nil {
		return nil, nil
	}
	if s.bloqueosPorTarea == nil {
		s.bloqueosPorTarea = map[int64]db.ResumenBloqueo{}
	}
	if s.bloqueosLoaded {
		return s.bloqueosPorTarea, nil
	}
	resumenBloqueos, err := db.ListarResumenBloqueos()
	if err != nil {
		return nil, err
	}
	for _, bloqueo := range resumenBloqueos {
		if bloqueo.ID <= 0 {
			continue
		}
		if _, ok := s.bloqueosPorTarea[bloqueo.ID]; !ok {
			s.bloqueosPorTarea[bloqueo.ID] = bloqueo
		}
	}
	s.bloqueosLoaded = true
	return s.bloqueosPorTarea, nil
}

func (s *autonomiaBatchSnapshot) autonomiaActivoEnProyecto(agente string, proyectoID int64, info *db.Agente) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, nil
	}
	for _, sesion := range s.sesiones {
		if sesion == nil || sesion.ProyectoID == nil {
			continue
		}
		if *sesion.ProyectoID == proyectoID && strings.EqualFold(strings.TrimSpace(sesion.Agente), agente) {
			return true, nil
		}
	}
	if info == nil || !info.Activo {
		return false, nil
	}
	key := agentProjectCacheKey(agente, proyectoID)
	if activo, ok := s.activeHandleByAgentProject[key]; ok {
		return activo, nil
	}
	if handle := s.hotHandlesByAgentProject[key]; handle != nil {
		s.activeHandleByAgentProject[key] = true
		return true, nil
	}
	handle, err := db.GetRuntimeHandleOperativoRecienteAgenteProyecto(agente, &proyectoID)
	if err != nil {
		return false, err
	}
	activo := handle != nil
	s.activeHandleByAgentProject[key] = activo
	return activo, nil
}

func agentProjectCacheKey(agente string, proyectoID int64) string {
	return strings.ToLower(strings.TrimSpace(agente)) + "|" + int64CacheKey(proyectoID)
}

func int64CacheKey(v int64) string {
	return strconv.FormatInt(v, 10)
}
