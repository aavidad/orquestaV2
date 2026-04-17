package operacionesapp

import "orquesta/db"

type Repository struct{}

func (Repository) ListAgents() ([]*Agente, error) {
	items, err := db.ListarAgentes()
	if err != nil {
		return nil, err
	}
	out := make([]*Agente, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapAgente(item))
	}
	return out, nil
}

func (Repository) ListConnectors() ([]*Conector, error) {
	items, err := db.ListarConectores()
	if err != nil {
		return nil, err
	}
	out := make([]*Conector, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapConector(item))
	}
	return out, nil
}

func (Repository) ListAssignments(estado, agente string) ([]*Asignacion, error) {
	items, err := db.ListarAsignacionesOpsView(estado, agente)
	if err != nil {
		return nil, err
	}
	out := make([]*Asignacion, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapAsignacion(item))
	}
	return out, nil
}

func (Repository) ListActiveSessions() ([]*SesionActiva, error) {
	items, err := db.ListarSesionesActivasOpsView()
	if err != nil {
		return nil, err
	}
	out := make([]*SesionActiva, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapSesionActiva(item))
	}
	return out, nil
}

func (Repository) GetProject(ref string) (*Proyecto, error) {
	item, err := db.GetProyecto(ref)
	if err != nil || item == nil {
		return nil, err
	}
	return mapProyecto(item), nil
}

func (Repository) ListInspectionSessions(filtro FiltroSesionesInspeccion) ([]*Sesion, error) {
	items, err := db.ListarSesionesInspeccion(mapFiltroSesionesInspeccion(filtro))
	if err != nil {
		return nil, err
	}
	out := make([]*Sesion, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapSesion(item))
	}
	return out, nil
}

func (Repository) GetInspectionSession(id int64) (*Sesion, error) {
	item, err := db.GetSesionInspeccionByID(id)
	if err != nil || item == nil {
		return nil, err
	}
	return mapSesion(item), nil
}

func (Repository) AuditLog(limit int) ([]AuditEntry, error) {
	items, err := db.AuditLog(limit)
	if err != nil {
		return nil, err
	}
	out := make([]AuditEntry, 0, len(items))
	for _, item := range items {
		out = append(out, mapAuditEntry(item))
	}
	return out, nil
}

func (Repository) RegisterAgent(nombre, rol string) error {
	return db.RegistrarAgente(nombre, rol)
}

func (Repository) RetireAgent(nombre string) error {
	return db.RetirarAgente(nombre)
}

func (Repository) RehabilitateAgent(nombre string) error {
	return db.RehabilitarAgente(nombre)
}

func mapAgente(in *db.Agente) *Agente {
	if in == nil {
		return nil
	}
	return &Agente{
		Nombre:                    in.Nombre,
		Rol:                       in.Rol,
		Activo:                    in.Activo,
		Habilitado:                in.Habilitado,
		SinCuotaProveedor:         in.SinCuotaProveedor,
		EstadoSesion:              in.EstadoSesion,
		UltimaSesion:              in.UltimaSesion,
		ConsumoDiaSegundos:        in.ConsumoDiaSegundos,
		ConsumoSemanalSegundos:    in.ConsumoSemanalSegundos,
		LimiteDiaSegundos:         in.LimiteDiaSegundos,
		LimiteSemanalSegundos:     in.LimiteSemanalSegundos,
		LastUsageResetAt:          in.LastUsageResetAt,
		EstadoCuota:               in.EstadoCuota,
		ReanimarAt:                in.ReanimarAt,
		MotivoPausa:               in.MotivoPausa,
		CuotaRestantePct:          in.CuotaRestantePct,
		PresupuestoEstado:         in.PresupuestoEstado,
		PresupuestoFuente:         in.PresupuestoFuente,
		PresupuestoCheckedAt:      in.PresupuestoCheckedAt,
		PresupuestoStale:          in.PresupuestoStale,
		PresupuestoVentana:        in.PresupuestoVentana,
		PresupuestoResetAt:        in.PresupuestoResetAt,
		PresupuestoSesionPct:      in.PresupuestoSesionPct,
		PresupuestoSesionResetAt:  in.PresupuestoSesionResetAt,
		PresupuestoDiarioPct:      in.PresupuestoDiarioPct,
		PresupuestoDiarioResetAt:  in.PresupuestoDiarioResetAt,
		PresupuestoSemanalPct:     in.PresupuestoSemanalPct,
		PresupuestoSemanalResetAt: in.PresupuestoSemanalResetAt,
		RemainingSeconds:          in.RemainingSeconds,
		RemainingMessages:         in.RemainingMessages,
		RemainingTokens:           in.RemainingTokens,
		RemainingCredits:          in.RemainingCredits,
		ObservedUsageTokens:       in.ObservedUsageTokens,
		ObservedUsageCostUSD:      in.ObservedUsageCostUSD,
		ObservedUsageMessages:     in.ObservedUsageMessages,
		ObservedUsageTurns:        in.ObservedUsageTurns,
		ObservedUsageUpdatedAt:    in.ObservedUsageUpdatedAt,
		ObservedSessionPath:       in.ObservedSessionPath,
		CuentaID:                  in.CuentaID,
		CuentaUsuario:             in.CuentaUsuario,
		CuentaEmail:               in.CuentaEmail,
		CuentaFuente:              in.CuentaFuente,
		CuentaObservadaAt:         in.CuentaObservadaAt,
	}
}

func mapConector(in *db.Conector) *Conector {
	if in == nil {
		return nil
	}
	return &Conector{
		ID:           in.ID,
		Slug:         in.Slug,
		Nombre:       in.Nombre,
		Transporte:   in.Transporte,
		Comando:      in.Comando,
		ArgsJSON:     in.ArgsJSON,
		EnvJSON:      in.EnvJSON,
		MetadataJSON: in.MetadataJSON,
		Activo:       in.Activo,
		CreatedAt:    in.CreatedAt,
		UpdatedAt:    in.UpdatedAt,
	}
}

func mapAsignacion(in *db.Asignacion) *Asignacion {
	if in == nil {
		return nil
	}
	return &Asignacion{
		ID:             in.ID,
		Agente:         in.Agente,
		ProyectoID:     in.ProyectoID,
		ProyectoSlug:   in.ProyectoSlug,
		ProyectoNombre: in.ProyectoNombre,
		Estado:         EstadoAsignacion(in.Estado),
		Nota:           in.Nota,
		CreatedAt:      in.CreatedAt,
		UpdatedAt:      in.UpdatedAt,
		CerradaAt:      in.CerradaAt,
	}
}

func mapSesionActiva(in *db.SesionActiva) *SesionActiva {
	if in == nil {
		return nil
	}
	return &SesionActiva{
		ID:                 in.ID,
		Agente:             in.Agente,
		Inicio:             in.Inicio,
		Fin:                in.Fin,
		Activa:             in.Activa,
		ConectorID:         in.ConectorID,
		ConectorSlug:       in.ConectorSlug,
		ProyectoID:         in.ProyectoID,
		ProyectoSlug:       in.ProyectoSlug,
		Estado:             in.Estado,
		Cwd:                in.Cwd,
		Herramienta:        in.Herramienta,
		ExternalSessionID:  in.ExternalSessionID,
		ResumePayloadJSON:  in.ResumePayloadJSON,
		ResumenContinuidad: in.ResumenContinuidad,
		Branch:             in.Branch,
		HeartbeatAt:        in.HeartbeatAt,
		Host:               in.Host,
		PID:                in.PID,
	}
}

func mapProyecto(in *db.Proyecto) *Proyecto {
	if in == nil {
		return nil
	}
	return &Proyecto{
		ID:        in.ID,
		Slug:      in.Slug,
		Nombre:    in.Nombre,
		RutaAbs:   in.RutaAbs,
		Tipo:      string(in.Tipo),
		ParentID:  in.ParentID,
		Activo:    in.Activo,
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}

func mapSesion(in *db.Sesion) *Sesion {
	if in == nil {
		return nil
	}
	return &Sesion{
		ID:                 in.ID,
		Agente:             in.Agente,
		ConectorID:         in.ConectorID,
		ConectorSlug:       in.ConectorSlug,
		ConectorNombre:     in.ConectorNombre,
		ProyectoID:         in.ProyectoID,
		ProyectoSlug:       in.ProyectoSlug,
		ProyectoNombre:     in.ProyectoNombre,
		Inicio:             in.Inicio,
		Fin:                in.Fin,
		Activa:             in.Activa,
		Estado:             in.Estado,
		CWD:                in.CWD,
		Herramienta:        in.Herramienta,
		ExternalSessionID:  in.ExternalSessionID,
		ResumePayloadJSON:  in.ResumePayloadJSON,
		ResumenContinuidad: in.ResumenContinuidad,
		Branch:             in.Branch,
		HeartbeatAt:        in.HeartbeatAt,
		Host:               in.Host,
		PID:                in.PID,
	}
}

func mapAuditEntry(in db.AuditEntry) AuditEntry {
	return AuditEntry{
		Agente:    in.Agente,
		Accion:    in.Accion,
		Entidad:   in.Entidad,
		EntidadID: in.EntidadID,
		Detalle:   in.Detalle,
		CreatedAt: in.CreatedAt,
	}
}

func mapFiltroSesionesInspeccion(in FiltroSesionesInspeccion) db.FiltroSesionesInspeccion {
	return db.FiltroSesionesInspeccion{
		Agente:     in.Agente,
		ProyectoID: in.ProyectoID,
		Activa:     in.Activa,
		Estado:     in.Estado,
		Limit:      in.Limit,
	}
}
