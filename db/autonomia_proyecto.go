package db

import (
	"database/sql"
	"fmt"
	"orquesta/planificadorpolicy"
	"strings"
	"time"
)

type EstadoAutonomiaProyecto string

const (
	AutonomiaProyectoActiva          EstadoAutonomiaProyecto = "activo"
	AutonomiaProyectoEsperandoReview EstadoAutonomiaProyecto = "esperando_review"
	AutonomiaProyectoEsperandoHumano EstadoAutonomiaProyecto = "esperando_humano"
	AutonomiaProyectoCerrando        EstadoAutonomiaProyecto = "cerrando"
	AutonomiaProyectoCerrado         EstadoAutonomiaProyecto = "cerrado"
)

type ProyectoAutonomia struct {
	ProyectoID           int64                   `json:"proyecto_id"`
	ProyectoSlug         string                  `json:"proyecto_slug,omitempty"`
	Enabled              bool                    `json:"enabled"`
	ObjetivoGeneral      string                  `json:"objetivo_general"`
	DefinitionOfDoneJSON string                  `json:"definition_of_done_json"`
	MaxWorkers           int                     `json:"max_workers"`
	SupervisorAgente     string                  `json:"supervisor_agente"`
	ReviewerAgente       string                  `json:"reviewer_agente"`
	ReserveReviewer      bool                    `json:"reserve_reviewer"`
	ReserveSupervisor    bool                    `json:"reserve_supervisor"`
	ReviewRequired       bool                    `json:"review_required"`
	AutoCreateTasks      bool                    `json:"auto_create_tasks"`
	AutoCloseProject     bool                    `json:"auto_close_project"`
	EstadoAutonomia      EstadoAutonomiaProyecto `json:"estado_autonomia"`
	LastSupervisionAt    *time.Time              `json:"last_supervision_at,omitempty"`
	LastReviewAt         *time.Time              `json:"last_review_at,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

type FiltroAutonomiaCiclos struct {
	ProyectoID *int64
	Kind       *string
	Agente     *string
	Limit      int
}

type AutonomiaCiclo struct {
	ID           int64     `json:"id"`
	ProyectoID   int64     `json:"proyecto_id"`
	ProyectoSlug string    `json:"proyecto_slug,omitempty"`
	Kind         string    `json:"kind"`
	Agente       string    `json:"agente"`
	SesionID     *int64    `json:"sesion_id,omitempty"`
	RuntimeID    *int64    `json:"runtime_id,omitempty"`
	InputJSON    string    `json:"input_json"`
	DecisionJSON string    `json:"decision_json"`
	Resultado    string    `json:"resultado"`
	CreatedAt    time.Time `json:"created_at"`
}

func GetProyectoAutonomia(proyectoID int64) (*ProyectoAutonomia, error) {
	if proyectoID <= 0 {
		return nil, fmt.Errorf("proyecto_id inválido")
	}
	row := DB.QueryRow(`
		SELECT pa.proyecto_id, COALESCE(p.slug, ''), pa.enabled, pa.objetivo_general,
		       pa.definition_of_done_json, pa.max_workers, pa.supervisor_agente,
		       pa.reviewer_agente, pa.reserve_reviewer, pa.reserve_supervisor,
		       pa.review_required, pa.auto_create_tasks, pa.auto_close_project,
		       pa.estado_autonomia, pa.last_supervision_at, pa.last_review_at,
		       pa.created_at, pa.updated_at
		  FROM proyectos_autonomia pa
		  LEFT JOIN proyectos p ON p.id = pa.proyecto_id
		 WHERE pa.proyecto_id = ?`, proyectoID)
	item, err := scanProyectoAutonomia(row)
	if err == sql.ErrNoRows {
		return proyectoAutonomiaDefault(proyectoID), nil
	}
	return item, err
}

func ListarProyectosAutonomia(soloEnabled *bool) ([]*ProyectoAutonomia, error) {
	q := `
		SELECT pa.proyecto_id, COALESCE(p.slug, ''), pa.enabled, pa.objetivo_general,
		       pa.definition_of_done_json, pa.max_workers, pa.supervisor_agente,
		       pa.reviewer_agente, pa.reserve_reviewer, pa.reserve_supervisor,
		       pa.review_required, pa.auto_create_tasks, pa.auto_close_project,
		       pa.estado_autonomia, pa.last_supervision_at, pa.last_review_at,
		       pa.created_at, pa.updated_at
		  FROM proyectos_autonomia pa
		  LEFT JOIN proyectos p ON p.id = pa.proyecto_id
		 WHERE 1=1`
	args := []any{}
	if soloEnabled != nil {
		q += ` AND pa.enabled = ?`
		args = append(args, boolToIntAutonomia(*soloEnabled))
	}
	q += ` ORDER BY pa.proyecto_id`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ProyectoAutonomia
	for rows.Next() {
		item, err := scanProyectoAutonomia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func UpsertProyectoAutonomia(item *ProyectoAutonomia) (int64, error) {
	if item == nil || item.ProyectoID <= 0 {
		return 0, fmt.Errorf("proyecto_autonomia inválido")
	}
	item.ObjetivoGeneral = strings.TrimSpace(item.ObjetivoGeneral)
	item.SupervisorAgente = strings.TrimSpace(item.SupervisorAgente)
	item.ReviewerAgente = strings.TrimSpace(item.ReviewerAgente)
	if strings.TrimSpace(item.DefinitionOfDoneJSON) == "" {
		item.DefinitionOfDoneJSON = "{}"
	}
	if strings.TrimSpace(string(item.EstadoAutonomia)) == "" {
		item.EstadoAutonomia = AutonomiaProyectoActiva
	}
	if _, err := DB.Exec(`
		INSERT INTO proyectos_autonomia (
			proyecto_id, enabled, objetivo_general, definition_of_done_json,
			max_workers, supervisor_agente, reviewer_agente, reserve_reviewer,
			reserve_supervisor, review_required, auto_create_tasks,
			auto_close_project, estado_autonomia, last_supervision_at,
			last_review_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(proyecto_id) DO UPDATE SET
			enabled = excluded.enabled,
			objetivo_general = excluded.objetivo_general,
			definition_of_done_json = excluded.definition_of_done_json,
			max_workers = excluded.max_workers,
			supervisor_agente = excluded.supervisor_agente,
			reviewer_agente = excluded.reviewer_agente,
			reserve_reviewer = excluded.reserve_reviewer,
			reserve_supervisor = excluded.reserve_supervisor,
			review_required = excluded.review_required,
			auto_create_tasks = excluded.auto_create_tasks,
			auto_close_project = excluded.auto_close_project,
			estado_autonomia = excluded.estado_autonomia,
			last_supervision_at = COALESCE(excluded.last_supervision_at, proyectos_autonomia.last_supervision_at),
			last_review_at = COALESCE(excluded.last_review_at, proyectos_autonomia.last_review_at),
			updated_at = CURRENT_TIMESTAMP`,
		item.ProyectoID, boolToIntAutonomia(item.Enabled), item.ObjetivoGeneral, item.DefinitionOfDoneJSON,
		item.MaxWorkers, item.SupervisorAgente, item.ReviewerAgente, boolToIntAutonomia(item.ReserveReviewer),
		boolToIntAutonomia(item.ReserveSupervisor), boolToIntAutonomia(item.ReviewRequired),
		boolToIntAutonomia(item.AutoCreateTasks), boolToIntAutonomia(item.AutoCloseProject),
		string(item.EstadoAutonomia), nullableTimeAutonomia(item.LastSupervisionAt), nullableTimeAutonomia(item.LastReviewAt),
	); err != nil {
		return 0, err
	}
	Audit("orquesta", "upsert_proyecto_autonomia", "proyecto_autonomia", item.ProyectoID, item.ObjetivoGeneral)
	return item.ProyectoID, nil
}

func RegistrarAutonomiaCiclo(item *AutonomiaCiclo) (int64, error) {
	if item == nil || item.ProyectoID <= 0 || strings.TrimSpace(item.Kind) == "" {
		return 0, fmt.Errorf("autonomia_ciclo inválido")
	}
	if strings.TrimSpace(item.InputJSON) == "" {
		item.InputJSON = "{}"
	}
	if strings.TrimSpace(item.DecisionJSON) == "" {
		item.DecisionJSON = "{}"
	}
	res, err := DB.Exec(`
		INSERT INTO autonomia_ciclos (
			proyecto_id, kind, agente, sesion_id, runtime_id, input_json, decision_json, resultado
		) VALUES (?,?,?,?,?,?,?,?)`,
		item.ProyectoID, strings.TrimSpace(item.Kind), strings.TrimSpace(item.Agente),
		item.SesionID, item.RuntimeID, item.InputJSON, item.DecisionJSON, strings.TrimSpace(item.Resultado),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	item.ID = id
	Audit("orquesta", "registrar_autonomia_ciclo", "autonomia_ciclo", id, strings.TrimSpace(item.Kind))
	return id, nil
}

func ListarAutonomiaCiclos(filtro FiltroAutonomiaCiclos) ([]*AutonomiaCiclo, error) {
	q := `
		SELECT ac.id, ac.proyecto_id, COALESCE(p.slug, ''), ac.kind, ac.agente,
		       ac.sesion_id, ac.runtime_id, ac.input_json, ac.decision_json,
		       ac.resultado, ac.created_at
		  FROM autonomia_ciclos ac
		  LEFT JOIN proyectos p ON p.id = ac.proyecto_id
		 WHERE 1=1`
	args := []any{}
	if filtro.ProyectoID != nil {
		q += ` AND ac.proyecto_id = ?`
		args = append(args, *filtro.ProyectoID)
	}
	if filtro.Kind != nil {
		q += ` AND ac.kind = ?`
		args = append(args, strings.TrimSpace(*filtro.Kind))
	}
	if filtro.Agente != nil {
		q += ` AND ac.agente = ?`
		args = append(args, strings.TrimSpace(*filtro.Agente))
	}
	q += ` ORDER BY ac.id DESC`
	if filtro.Limit <= 0 {
		filtro.Limit = 50
	}
	q += ` LIMIT ?`
	args = append(args, filtro.Limit)
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*AutonomiaCiclo
	for rows.Next() {
		item, err := scanAutonomiaCiclo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func MarcarProyectoAutonomiaSupervisado(proyectoID int64, when time.Time) error {
	if proyectoID <= 0 {
		return fmt.Errorf("proyecto_id inválido")
	}
	_, err := DB.Exec(`
		UPDATE proyectos_autonomia
		   SET last_supervision_at = ?,
		       updated_at = CURRENT_TIMESTAMP
		 WHERE proyecto_id = ?`, when.UTC(), proyectoID)
	return err
}

func MarcarProyectoAutonomiaRevisado(proyectoID int64, when time.Time) error {
	if proyectoID <= 0 {
		return fmt.Errorf("proyecto_id inválido")
	}
	_, err := DB.Exec(`
		UPDATE proyectos_autonomia
		   SET last_review_at = ?,
		       updated_at = CURRENT_TIMESTAMP
		 WHERE proyecto_id = ?`, when.UTC(), proyectoID)
	return err
}

func proyectoAutonomiaDefault(proyectoID int64) *ProyectoAutonomia {
	return &ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              false,
		DefinitionOfDoneJSON: "{}",
		ReserveReviewer:      true,
		ReserveSupervisor:    true,
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      AutonomiaProyectoActiva,
	}
}

func scanProyectoAutonomia(scanner interface{ Scan(dest ...any) error }) (*ProyectoAutonomia, error) {
	var (
		item              ProyectoAutonomia
		enabled           int
		reserveReviewer   int
		reserveSupervisor int
		reviewRequired    int
		autoCreateTasks   int
		autoCloseProject  int
		lastSupervision   sql.NullTime
		lastReview        sql.NullTime
	)
	if err := scanner.Scan(
		&item.ProyectoID, &item.ProyectoSlug, &enabled, &item.ObjetivoGeneral,
		&item.DefinitionOfDoneJSON, &item.MaxWorkers, &item.SupervisorAgente,
		&item.ReviewerAgente, &reserveReviewer, &reserveSupervisor,
		&reviewRequired, &autoCreateTasks, &autoCloseProject,
		&item.EstadoAutonomia, &lastSupervision, &lastReview,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.SupervisorAgente = strings.TrimSpace(item.SupervisorAgente)
	item.ReviewerAgente = strings.TrimSpace(item.ReviewerAgente)
	item.Enabled = enabled == 1
	item.ReserveReviewer = reserveReviewer == 1
	item.ReserveSupervisor = reserveSupervisor == 1
	item.ReviewRequired = reviewRequired == 1
	item.AutoCreateTasks = autoCreateTasks == 1
	item.AutoCloseProject = autoCloseProject == 1
	if lastSupervision.Valid {
		item.LastSupervisionAt = &lastSupervision.Time
	}
	if lastReview.Valid {
		item.LastReviewAt = &lastReview.Time
	}
	return &item, nil
}

func scanAutonomiaCiclo(scanner interface{ Scan(dest ...any) error }) (*AutonomiaCiclo, error) {
	var (
		item      AutonomiaCiclo
		sesionID  sql.NullInt64
		runtimeID sql.NullInt64
	)
	if err := scanner.Scan(
		&item.ID, &item.ProyectoID, &item.ProyectoSlug, &item.Kind, &item.Agente,
		&sesionID, &runtimeID, &item.InputJSON, &item.DecisionJSON,
		&item.Resultado, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	if sesionID.Valid {
		item.SesionID = &sesionID.Int64
	}
	if runtimeID.Valid {
		item.RuntimeID = &runtimeID.Int64
	}
	return &item, nil
}

type candidatoProyectoAutomatico struct {
	Proyecto        *Proyecto
	Operacion       *ProyectoOperacion
	Tarea           *Tarea
	Activos         int
	Workers         int
	Deseados        int
	Deficit         int
	TieneMin        bool
	CargaRatio      int
	MicroCerrada    bool
	ContratoCerrado bool
}

func seleccionarProyectoAutomaticoDisponible(agente string) (int64, error) {
	agente = strings.TrimSpace(agente)
	activo := true
	proyectos, err := ListarProyectos(FiltroProyectos{Activo: &activo})
	if err != nil {
		return 0, err
	}
	totalPlanificables, err := contarAgentesPlanificables()
	if err != nil {
		return 0, err
	}
	activosPorProyecto, err := ContarAsignacionesActivasPorProyecto()
	if err != nil {
		return 0, err
	}
	var mejor *candidatoProyectoAutomatico
	preferirMicroprogramacion := false
	for _, proyecto := range proyectos {
		if proyecto == nil || proyecto.ID == 0 {
			continue
		}
		reservado, _, err := agenteReservadoAutonomiaProyecto(agente, proyecto.ID)
		if err != nil {
			return 0, err
		}
		if reservado {
			continue
		}
		disponible, err := ProyectoDisponibleParaAutonomia(proyecto.ID)
		if err != nil {
			return 0, err
		}
		if !disponible {
			continue
		}
		tarea, err := buscarSiguienteTareaLibreParaAgente(agente, proyecto.ID)
		if err != nil {
			return 0, err
		}
		if tarea == nil {
			continue
		}
		if !preferirMicroprogramacion {
			preferirMicroprogramacion, err = agentePrefiereTrabajoMicroprogramacion(agente, proyecto.ID)
			if err != nil {
				return 0, err
			}
		}
		microCerrada := false
		contratoCerrado := false
		if preferirMicroprogramacion {
			if microCerrada, err = tareaTieneEspecificacionActiva(tarea.ID); err != nil {
				return 0, err
			}
			contratoCerrado = tarea.ContratoDefinido
		}
		op, err := GetProyectoOperacion(proyecto.ID)
		if err != nil {
			return 0, err
		}
		activos := activosPorProyecto[proyecto.ID]
		workers, err := contarWorkersActivosProyecto(proyecto.ID)
		if err != nil {
			return 0, err
		}
		if op.MaxAgentes > 0 && activos >= op.MaxAgentes {
			continue
		}
		admiteWorker, err := proyectoAdmiteWorkerAutonomia(proyecto.ID)
		if err != nil {
			return 0, err
		}
		if !admiteWorker {
			continue
		}
		deseados := cupoDeseadoProyecto(op, totalPlanificables)
		candidato := &candidatoProyectoAutomatico{
			Proyecto:        proyecto,
			Operacion:       op,
			Tarea:           tarea,
			Activos:         activos,
			Workers:         workers,
			Deseados:        deseados,
			Deficit:         deseados - workers,
			TieneMin:        workers < op.MinAgentes,
			CargaRatio:      cargaProyecto(workers, deseados),
			MicroCerrada:    microCerrada,
			ContratoCerrado: contratoCerrado,
		}
		if mejorProyectoAutomatico(candidato, mejor) {
			mejor = candidato
		}
	}
	if mejor == nil || mejor.Proyecto == nil {
		return 0, nil
	}
	return mejor.Proyecto.ID, nil
}

func mejorProyectoAutomatico(candidato *candidatoProyectoAutomatico, actual *candidatoProyectoAutomatico) bool {
	return planificadorpolicy.PreferAutomaticProjectCandidate(
		automaticProjectCandidateSnapshot(candidato),
		automaticProjectCandidateSnapshot(actual),
	)
}

func contarAgentesPlanificables() (int, error) {
	agentes, err := ListarAgentesPlanificables()
	if err != nil {
		return 0, err
	}
	if len(agentes) == 0 {
		return 1, nil
	}
	return len(agentes), nil
}

func cupoDeseadoProyecto(op *ProyectoOperacion, totalAgentes int) int {
	return planificadorpolicy.DesiredProjectQuota(projectOperationSnapshot(op), totalAgentes)
}

func cargaProyecto(activos, deseados int) int {
	return planificadorpolicy.ProjectLoad(activos, deseados)
}

func projectOperationSnapshot(op *ProyectoOperacion) *planificadorpolicy.ProjectOperationSnapshot {
	if op == nil {
		return nil
	}
	return &planificadorpolicy.ProjectOperationSnapshot{
		ObjectivePct: op.ObjetivoPct,
		MinAgents:    op.MinAgentes,
		MaxAgents:    op.MaxAgentes,
		Priority:     op.Prioridad,
	}
}

func automaticProjectCandidateSnapshot(candidate *candidatoProyectoAutomatico) *planificadorpolicy.AutomaticProjectCandidateSnapshot {
	if candidate == nil || candidate.Proyecto == nil {
		return nil
	}
	return &planificadorpolicy.AutomaticProjectCandidateSnapshot{
		ProjectID:      candidate.Proyecto.ID,
		Operation:      projectOperationSnapshot(candidate.Operacion),
		Deficit:        candidate.Deficit,
		HasMin:         candidate.TieneMin,
		LoadRatio:      candidate.CargaRatio,
		MicroClosed:    candidate.MicroCerrada,
		ContractClosed: candidate.ContratoCerrado,
	}
}

func agenteReservadoAutonomiaProyecto(agente string, proyectoID int64) (bool, string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, "", nil
	}
	policy, err := GetProyectoAutonomia(proyectoID)
	if err != nil || policy == nil || !policy.Enabled {
		return false, "", err
	}
	if policy.ReserveSupervisor && strings.EqualFold(agente, strings.TrimSpace(policy.SupervisorAgente)) {
		return true, "supervisor", nil
	}
	if policy.ReserveReviewer && strings.EqualFold(agente, strings.TrimSpace(policy.ReviewerAgente)) {
		return true, "reviewer", nil
	}
	return false, "", nil
}

func contarWorkersActivosProyecto(proyectoID int64) (int, error) {
	proyecto, err := GetProyecto(jsonNumber(proyectoID))
	if err != nil {
		return 0, err
	}
	estado := AsignacionActiva
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return 0, err
	}
	total := 0
	for _, asignacion := range asignaciones {
		if asignacion == nil {
			continue
		}
		reservado, _, err := agenteReservadoAutonomiaProyecto(asignacion.Agente, proyectoID)
		if err != nil {
			return 0, err
		}
		if reservado {
			continue
		}
		infoAgente, err := GetAgente(asignacion.Agente)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return 0, err
		}
		if infoAgente == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(infoAgente.EstadoCuota), "activo") {
			continue
		}
		if !agentePertenecePoolAutobootstrapProyecto(asignacion.Agente, proyecto) {
			continue
		}
		total++
	}
	return total, nil
}

func proyectoAdmiteWorkerAutonomiaParaAgente(proyectoID int64, agente string) (bool, error) {
	if proyectoID <= 0 {
		return true, nil
	}
	policy, err := GetProyectoAutonomia(proyectoID)
	if err != nil || policy == nil || !policy.Enabled || policy.MaxWorkers <= 0 {
		return true, err
	}
	workers, err := contarWorkersActivosProyecto(proyectoID)
	if err != nil {
		return false, err
	}
	agente = strings.TrimSpace(agente)
	if agente != "" {
		estado := AsignacionActiva
		asignaciones, err := ListarAsignaciones(FiltroAsignaciones{
			Agente:     &agente,
			ProyectoID: &proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return false, err
		}
		if len(asignaciones) > 0 {
			reservado, _, err := agenteReservadoAutonomiaProyecto(agente, proyectoID)
			if err != nil {
				return false, err
			}
			if !reservado && workers > 0 {
				workers--
			}
		}
	}
	return workers < policy.MaxWorkers, nil
}

func proyectoAdmiteWorkerAutonomia(proyectoID int64) (bool, error) {
	return proyectoAdmiteWorkerAutonomiaParaAgente(proyectoID, "")
}

func boolToIntAutonomia(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableTimeAutonomia(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.UTC()
}
