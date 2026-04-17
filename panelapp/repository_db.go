package panelapp

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

func (Repository) CountTasksByState() (map[string]int, error) {
	return db.ContarTareasPorEstado()
}

func (Repository) ListProposals(estado *EstadoPropuesta) ([]*Propuesta, error) {
	var dbEstado *db.EstadoPropuesta
	if estado != nil {
		value := db.EstadoPropuesta(*estado)
		dbEstado = &value
	}
	items, err := db.ListarPropuestas(dbEstado, nil)
	if err != nil {
		return nil, err
	}
	out := make([]*Propuesta, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapPropuesta(item))
	}
	return out, nil
}

func (Repository) CountVotes(propuestaID int64) (int, int, int, int, error) {
	return db.ContarVotos(propuestaID)
}

func (Repository) ListTasks(filtro FiltroTareas) ([]*Tarea, error) {
	items, err := db.ListarTareas(mapFiltroTareas(filtro))
	if err != nil {
		return nil, err
	}
	out := make([]*Tarea, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapTarea(item))
	}
	return out, nil
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

func mapPropuesta(in *db.Propuesta) *Propuesta {
	if in == nil {
		return nil
	}
	return &Propuesta{
		ID:           in.ID,
		Codigo:       in.Codigo,
		Titulo:       in.Titulo,
		Descripcion:  in.Descripcion,
		ProyectoID:   in.ProyectoID,
		Tipo:         in.Tipo,
		Estado:       EstadoPropuesta(in.Estado),
		PropuestoPor: in.PropuestoPor,
		Distribuidor: in.Distribuidor,
		CreatedAt:    in.CreatedAt,
		UpdatedAt:    in.UpdatedAt,
		CerradaAt:    in.CerradaAt,
	}
}

func mapTarea(in *db.Tarea) *Tarea {
	if in == nil {
		return nil
	}
	dependencias := append([]int64(nil), in.Dependencias...)
	return &Tarea{
		ID:               in.ID,
		Titulo:           in.Titulo,
		Descripcion:      in.Descripcion,
		ProyectoID:       in.ProyectoID,
		Modulo:           in.Modulo,
		Estado:           EstadoTarea(in.Estado),
		Agente:           in.Agente,
		PropuestaID:      in.PropuestaID,
		Prioridad:        PrioridadTarea(in.Prioridad),
		Dependencias:     dependencias,
		ContratoDefinido: in.ContratoDefinido,
		BlueprintKey:     in.BlueprintKey,
		CreadoPor:        in.CreadoPor,
		CommitCierre:     in.CommitCierre,
		Notas:            in.Notas,
		CreatedAt:        in.CreatedAt,
		UpdatedAt:        in.UpdatedAt,
		CompletadaAt:     in.CompletadaAt,
	}
}

func mapFiltroTareas(in FiltroTareas) db.FiltroTareas {
	out := db.FiltroTareas{
		Agente:      in.Agente,
		ProyectoID:  in.ProyectoID,
		Modulo:      in.Modulo,
		PropuestaID: in.PropuestaID,
		Libre:       in.Libre,
	}
	if in.Estado != nil {
		value := db.EstadoTarea(*in.Estado)
		out.Estado = &value
	}
	return out
}
