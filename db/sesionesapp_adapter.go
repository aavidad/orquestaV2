package db

import (
	"orquesta/sesionesapp"
)

type SqliteSesionesRepo struct{}

func (SqliteSesionesRepo) RegisterCodex() (string, error) {
	return RegistrarCodex()
}

func (SqliteSesionesRepo) StartSession(agente string) (int64, error) {
	return IniciarSesion(agente)
}

func (SqliteSesionesRepo) FinishSession(agente string) error {
	return FinSesion(agente)
}

func (SqliteSesionesRepo) GetProject(ref string) (*sesionesapp.Proyecto, error) {
	p, err := GetProyecto(ref)
	if err != nil || p == nil {
		return nil, err
	}
	return &sesionesapp.Proyecto{ID: p.ID, Slug: p.Slug, Nombre: p.Nombre, RutaAbs: p.RutaAbs}, nil
}

func (SqliteSesionesRepo) GetConnector(ref string) (*sesionesapp.Conector, error) {
	c, err := GetConector(ref)
	if err != nil || c == nil {
		return nil, err
	}
	return &sesionesapp.Conector{ID: c.ID, Slug: c.Slug}, nil
}

func (SqliteSesionesRepo) ActivateAssignment(agente string, proyectoID int64, nota string) error {
	return ActivarAsignacion(agente, proyectoID, nota)
}

func mapSesionToApp(s *Sesion) *sesionesapp.Sesion {
	if s == nil {
		return nil
	}
	return &sesionesapp.Sesion{
		ID:                 s.ID,
		Agente:             s.Agente,
		ConectorID:         s.ConectorID,
		ConectorSlug:       s.ConectorSlug,
		ConectorNombre:     s.ConectorNombre,
		ProyectoID:         s.ProyectoID,
		ProyectoSlug:       s.ProyectoSlug,
		ProyectoNombre:     s.ProyectoNombre,
		Inicio:             s.Inicio,
		Fin:                s.Fin,
		Activa:             s.Activa,
		Estado:             s.Estado,
		CWD:                s.CWD,
		Herramienta:        s.Herramienta,
		ExternalSessionID:  s.ExternalSessionID,
		ResumePayloadJSON:  s.ResumePayloadJSON,
		ResumenContinuidad: s.ResumenContinuidad,
		Branch:             s.Branch,
		HeartbeatAt:        s.HeartbeatAt,
		Host:               s.Host,
		PID:                s.PID,
	}
}

func (SqliteSesionesRepo) StartSessionContext(in sesionesapp.SesionInicio) (*sesionesapp.Sesion, error) {
	s, err := IniciarSesionContexto(SesionInicio{
		Agente:             in.Agente,
		ConectorID:         in.ConectorID,
		ProyectoID:         in.ProyectoID,
		CWD:                in.CWD,
		Herramienta:        in.Herramienta,
		ExternalSessionID:  in.ExternalSessionID,
		ResumePayloadJSON:  in.ResumePayloadJSON,
		ResumenContinuidad: in.ResumenContinuidad,
		Branch:             in.Branch,
		Host:               in.Host,
		PID:                in.PID,
	})
	return mapSesionToApp(s), err
}

func (SqliteSesionesRepo) GetLastSession(agente string, proyectoID *int64) (*sesionesapp.Sesion, error) {
	s, err := ObtenerUltimaSesion(agente, proyectoID)
	return mapSesionToApp(s), err
}

func (SqliteSesionesRepo) GetSessionByID(id int64) (*sesionesapp.Sesion, error) {
	s, err := GetSesionByID(id)
	return mapSesionToApp(s), err
}

func (SqliteSesionesRepo) ListInspectionSessions(filtro sesionesapp.FiltroInspeccion) ([]*sesionesapp.Sesion, error) {
	res, err := ListarSesionesInspeccion(FiltroSesionesInspeccion{})
	var list []*sesionesapp.Sesion
	for _, s := range res {
		list = append(list, mapSesionToApp(s))
	}
	return list, err
}

func (SqliteSesionesRepo) GetInspectionSessionByID(id int64) (*sesionesapp.Sesion, error) {
	s, err := GetSesionInspeccionByID(id)
	return mapSesionToApp(s), err
}

func (SqliteSesionesRepo) SaveActiveSession(agente string, proyectoID *int64, upd sesionesapp.SesionUpdate) error {
	return GuardarSesionActiva(agente, proyectoID, SesionUpdate{
		CWD:                upd.CWD,
		Herramienta:        upd.Herramienta,
		ExternalSessionID:  upd.ExternalSessionID,
		ResumePayloadJSON:  upd.ResumePayloadJSON,
		ResumenContinuidad: upd.ResumenContinuidad,
		Branch:             upd.Branch,
		Host:               upd.Host,
		PID:                upd.PID,
		Heartbeat:          upd.Heartbeat,
		Estado:             upd.Estado,
	})
}

func (SqliteSesionesRepo) GetActiveSession(agente string, proyectoID *int64) (*sesionesapp.Sesion, error) {
	s, err := GetSesionActiva(agente, proyectoID)
	return mapSesionToApp(s), err
}

func (SqliteSesionesRepo) GetLastSessionWithFilter(agente string, proyectoID *int64, cwd string) (*sesionesapp.Sesion, error) {
	s, err := ObtenerUltimaSesionConFiltro(agente, proyectoID, cwd)
	return mapSesionToApp(s), err
}

func mapBudgetToApp(p *PresupuestoSesion) *sesionesapp.PresupuestoSesion {
	if p == nil {
		return nil
	}
	return &sesionesapp.PresupuestoSesion{
		ID:                p.ID,
		SesionID:          p.SesionID,
		PoolID:            p.PoolID,
		ModelSlug:         p.ModelSlug,
		WindowKind:        p.WindowKind,
		WindowStartedAt:   p.WindowStartedAt,
		ResetAt:           p.ResetAt,
		RemainingSeconds:  p.RemainingSeconds,
		RemainingMessages: p.RemainingMessages,
		RemainingTokens:   p.RemainingTokens,
		RemainingCredits:  p.RemainingCredits,
		BudgetSource:      p.BudgetSource,
		RawSnapshotJSON:   p.RawSnapshotJSON,
		CheckedAt:         p.CheckedAt,
		CreatedAt:         p.CreatedAt,
	}
}

func mapEvaluacionToApp(ev *EvaluacionPresupuesto) *sesionesapp.EvaluacionPresupuesto {
	if ev == nil {
		return nil
	}
	return &sesionesapp.EvaluacionPresupuesto{
		Estado:         ev.Estado,
		DebeHandoff:    ev.DebeHandoff,
		Motivo:         ev.Motivo,
		RemainingRatio: ev.RemainingRatio,
	}
}

func (SqliteSesionesRepo) RegisterSessionBudget(p *sesionesapp.PresupuestoSesion) (int64, error) {
	return RegistrarPresupuestoSesion(&PresupuestoSesion{
		ID:                p.ID,
		SesionID:          p.SesionID,
		PoolID:            p.PoolID,
		ModelSlug:         p.ModelSlug,
		WindowKind:        p.WindowKind,
		WindowStartedAt:   p.WindowStartedAt,
		ResetAt:           p.ResetAt,
		RemainingSeconds:  p.RemainingSeconds,
		RemainingMessages: p.RemainingMessages,
		RemainingTokens:   p.RemainingTokens,
		RemainingCredits:  p.RemainingCredits,
		BudgetSource:      p.BudgetSource,
		RawSnapshotJSON:   p.RawSnapshotJSON,
	})
}

func (SqliteSesionesRepo) GetLatestSessionBudget(sesionID int64) (*sesionesapp.PresupuestoSesion, error) {
	p, err := UltimoPresupuestoSesion(sesionID)
	return mapBudgetToApp(p), err
}

func (SqliteSesionesRepo) EvaluateSessionBudget(p *sesionesapp.PresupuestoSesion) (*sesionesapp.EvaluacionPresupuesto, error) {
	ev, err := EvaluarPresupuestoSesion(&PresupuestoSesion{
		ID:                p.ID,
		SesionID:          p.SesionID,
		PoolID:            p.PoolID,
		ModelSlug:         p.ModelSlug,
		WindowKind:        p.WindowKind,
		WindowStartedAt:   p.WindowStartedAt,
		ResetAt:           p.ResetAt,
		RemainingSeconds:  p.RemainingSeconds,
		RemainingMessages: p.RemainingMessages,
		RemainingTokens:   p.RemainingTokens,
		RemainingCredits:  p.RemainingCredits,
		BudgetSource:      p.BudgetSource,
	})
	return mapEvaluacionToApp(ev), err
}

func (SqliteSesionesRepo) ListAgents() ([]*sesionesapp.Agente, error) {
	agentes, err := ListarAgentes()
	var list []*sesionesapp.Agente
	for _, a := range agentes {
		list = append(list, &sesionesapp.Agente{Nombre: a.Nombre, Rol: a.Rol})
	}
	return list, err
}

func (SqliteSesionesRepo) ResolveModelPolicy(input sesionesapp.ModelPolicyInput) (*sesionesapp.ModelPolicyResolution, error) {
	resolucion, err := ResolverPoliticaModelo(ResolverPoliticaInput{
		AgenteNombre: input.AgentName,
		TareaID:      input.TaskID,
		ProyectoSlug: input.ProjectSlug,
		Fase:         input.Phase,
		PerfilTarea:  input.TaskProfile,
	})
	if err != nil || resolucion == nil {
		return nil, err
	}
	return &sesionesapp.ModelPolicyResolution{
		TaskProfile:     resolucion.PerfilTarea,
		PoolSlug:        resolucion.PoolSlug,
		ModelSlug:       resolucion.ModelSlug,
		ReasoningEffort: resolucion.ReasoningEffort,
	}, nil
}

func (SqliteSesionesRepo) GetCapacityPool(slug string) (*sesionesapp.CapacityPool, error) {
	pool, err := GetPool(slug)
	if err != nil || pool == nil {
		return nil, err
	}
	return &sesionesapp.CapacityPool{
		ID:           pool.ID,
		Runtime:      pool.Runtime,
		MetadataJSON: pool.MetadataJSON,
	}, nil
}

func (SqliteSesionesRepo) AssignSessionPool(sessionID, poolID int64) error {
	_, err := DB.Exec(`UPDATE sesiones SET pool_id = ? WHERE id = ?`, poolID, sessionID)
	return err
}

func (SqliteSesionesRepo) ListPendingProposals(agente string) ([]*sesionesapp.Propuesta, error) {
	props, err := PropuestasPendientesVoto(agente)
	var list []*sesionesapp.Propuesta
	for _, p := range props {
		list = append(list, &sesionesapp.Propuesta{ID: p.ID, Codigo: p.Codigo, Titulo: p.Titulo})
	}
	return list, err
}

func mapCatalogToApp(cat *GovernanceCatalog) (*sesionesapp.GovernanceCatalog, error) {
	if cat == nil {
		return nil, nil
	}
	appCat := &sesionesapp.GovernanceCatalog{}
	for _, r := range cat.Reglas {
		appCat.Reglas = append(appCat.Reglas, &sesionesapp.Regla{Categoria: r.Categoria, Titulo: r.Titulo, Descripcion: r.Descripcion, Obligatoria: r.Activa})
	}
	for _, s := range cat.Skills {
		appCat.Skills = append(appCat.Skills, &sesionesapp.Skill{Nombre: s.Nombre})
	}
	for _, w := range cat.Workflows {
		appCat.Workflows = append(appCat.Workflows, &sesionesapp.Workflow{Nombre: w.Nombre, Pasos: w.Pasos})
	}
	return appCat, nil
}

func (SqliteSesionesRepo) ResolveGovernanceCatalog(rol string, proyectoID *int64) (*sesionesapp.GovernanceCatalog, error) {
	cat, err := ResolveGovernanceCatalog(rol, proyectoID)
	if err != nil {
		return nil, err
	}
	return mapCatalogToApp(cat)
}

func (SqliteSesionesRepo) ResolveGovernanceCatalogForContext(rol string, proyectoID *int64, agente string) (*sesionesapp.GovernanceCatalog, error) {
	cat, err := ResolveGovernanceCatalogForContext(rol, proyectoID, agente)
	if err != nil {
		return nil, err
	}
	return mapCatalogToApp(cat)
}

func (SqliteSesionesRepo) ResolveGovernanceWorkflowForContext(rol string, proyectoID *int64, agente, nombre string) (*sesionesapp.Workflow, error) {
	w, err := ResolveGovernanceWorkflowForContext(rol, proyectoID, agente, nombre)
	if err != nil || w == nil {
		return nil, err
	}
	return &sesionesapp.Workflow{Nombre: w.Nombre, Pasos: w.Pasos}, nil
}

func (SqliteSesionesRepo) ListRules(rol string) ([]*sesionesapp.Regla, error) {
	rules, err := GetReglasAgente(rol)
	var list []*sesionesapp.Regla
	for _, r := range rules {
		list = append(list, &sesionesapp.Regla{Categoria: r.Categoria, Titulo: r.Titulo, Descripcion: r.Descripcion, Obligatoria: r.Activa})
	}
	return list, err
}

func (SqliteSesionesRepo) ListSkills(rol string) ([]*sesionesapp.Skill, error) {
	skills, err := GetSkillsAgente(rol)
	var list []*sesionesapp.Skill
	for _, s := range skills {
		list = append(list, &sesionesapp.Skill{Nombre: s.Nombre})
	}
	return list, err
}

func (SqliteSesionesRepo) GetWorkflow(rol, nombre string) (*sesionesapp.Workflow, error) {
	w, err := GetWorkflow(rol, nombre)
	if err != nil || w == nil {
		return nil, err
	}
	return &sesionesapp.Workflow{Nombre: w.Nombre, Pasos: w.Pasos}, nil
}

func (SqliteSesionesRepo) ListWorkflows(rol string) ([]*sesionesapp.Workflow, error) {
	flows, err := GetWorkflowsAgente(rol)
	var list []*sesionesapp.Workflow
	for _, w := range flows {
		list = append(list, &sesionesapp.Workflow{Nombre: w.Nombre, Pasos: w.Pasos})
	}
	return list, err
}
