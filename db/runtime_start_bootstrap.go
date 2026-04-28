package db

import (
	"encoding/json"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
	"strings"
	"time"
)

func promptArranquePendiente(plan *runtimeagente.LaunchPlan) string {
	if plan == nil || plan.LaunchPromptEmbedded {
		return ""
	}
	return strings.TrimSpace(runtimeagente.LaunchPromptText(plan))
}

func nombreAgenteRuntime(preferido, almacenado string) string {
	preferido = strings.TrimSpace(preferido)
	almacenado = strings.TrimSpace(almacenado)
	if preferido != "" && strings.EqualFold(preferido, almacenado) {
		return preferido
	}
	return almacenado
}

func resolverConectorRuntimeOrder(agenteNombre, proyectoSlug, conectorRef string, ultima *Sesion) (*Conector, error) {
	ref := strings.TrimSpace(conectorRef)
	if ref == "" && ultima != nil {
		if strings.TrimSpace(ultima.ConectorSlug) != "" &&
			runtimeagente.ConectorCompatibleConAgente(agenteNombre, strings.TrimSpace(ultima.ConectorSlug), "") {
			ref = strings.TrimSpace(ultima.ConectorSlug)
		} else if ultima.ConectorID != nil && *ultima.ConectorID > 0 &&
			runtimeagente.ConectorCompatibleConAgente(agenteNombre, "", strings.TrimSpace(ultima.Herramienta)) {
			ref = jsonNumber(*ultima.ConectorID)
		}
	}
	if ref == "" {
		if runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(agenteNombre), "") {
			if preferido, err := resolverConectorPoolLocalCompartido(agenteNombre, proyectoSlug, ""); err == nil && strings.TrimSpace(preferido) != "" {
				ref = preferido
			}
		}
	}
	if ref == "" {
		ref = runtimeagente.ConectorPorDefectoAgente(agenteNombre)
	}
	return GetConector(ref)
}

func resolverConectorPoolLocalCompartido(agenteNombre, proyectoSlug, perfilTarea string) (string, error) {
	resolucion, err := ResolverPoliticaModelo(ResolverPoliticaInput{
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		PerfilTarea:  strings.TrimSpace(perfilTarea),
		AgenteNombre: strPtrRuntime(strings.TrimSpace(agenteNombre)),
	})
	if err != nil || resolucion == nil || strings.TrimSpace(resolucion.PoolSlug) == "" {
		return "", err
	}
	pool, err := GetPool(strings.TrimSpace(resolucion.PoolSlug))
	if err != nil || pool == nil {
		return "", err
	}
	meta := mapFromJSON(strings.TrimSpace(pool.MetadataJSON))
	if !strings.EqualFold(strings.TrimSpace(pool.Runtime), "ollama") {
		return "", nil
	}
	if strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "conector_canonico", "")), "ollama_pool_local") {
		return "ollama_pool_local", nil
	}
	return "", nil
}

func enriquecerResumeConGobernanzaDB(resume *runtimeagente.ResumeContext, rol, agente string, proyecto *Proyecto) {
	if resume == nil || strings.TrimSpace(rol) == "" {
		return
	}
	if strings.TrimSpace(resume.ExternalSessionID) == "" &&
		strings.TrimSpace(resume.ResumePayloadJSON) == "" &&
		strings.TrimSpace(resume.ResumenContinuidad) == "" {
		return
	}
	var proyectoID *int64
	if proyecto != nil {
		proyectoID = &proyecto.ID
	}
	contexto, resumen := BuildGovernanceContextSummaryForContext(strings.TrimSpace(rol), proyectoID, agente)
	if len(contexto) == 0 {
		return
	}
	if payload := AppendGovernanceCatalogPayload(resume.ResumePayloadJSON, contexto); payload != "" {
		resume.ResumePayloadJSON = payload
	}
	if resumen != "" && !strings.Contains(resume.ResumenContinuidad, resumen) {
		if strings.TrimSpace(resume.ResumenContinuidad) == "" {
			resume.ResumenContinuidad = resumen
		} else {
			resume.ResumenContinuidad += ". " + resumen
		}
	}
}

func prepararResumeBootstrapRuntime(agente string, proyecto *Proyecto, resume runtimeagente.ResumeContext, excludeOrderID int64) (runtimeagente.ResumeContext, *bootstrapRuntimeData, error) {
	if proyecto == nil {
		return resume, nil, nil
	}
	if canonical, err := CanonicalizeAgentName(agente); err == nil {
		agente = canonical
	}
	agente = strings.TrimSpace(agente)
	resume = SanitizeResumeContextForProject(resume, proyecto)
	order, err := claimBootstrapRuntimeOrderParaStart(agente, &proyecto.ID, excludeOrderID)
	if err != nil {
		return resume, nil, err
	}
	estadoPendiente := "pendiente"
	mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrRuntime(agente),
		ProyectoID: &proyecto.ID,
		Estado:     &estadoPendiente,
	})
	if err != nil {
		return resume, nil, err
	}
	checkpoint, err := UltimoRuntimeCheckpoint(agente, &proyecto.ID)
	if err != nil {
		return resume, nil, err
	}
	bootstrap := &bootstrapRuntimeData{
		Order:      order,
		Mailbox:    mailbox,
		Checkpoint: checkpoint,
	}

	if checkpoint != nil {
		if strings.TrimSpace(resume.Branch) == "" {
			resume.Branch = strings.TrimSpace(checkpoint.Branch)
		}
		if strings.TrimSpace(resume.CWD) == "" {
			resume.CWD = strings.TrimSpace(checkpoint.CWD)
		}
	}
	resume.CWD = RutaTrabajoPreferidaAgenteProyecto(agente, proyecto, strings.TrimSpace(resume.CWD))
	if strings.TrimSpace(resume.CWD) == "" {
		resume.CWD = strings.TrimSpace(proyecto.RutaAbs)
	}
	if order != nil && order.Tipo == "handoff" {
		var payload HandoffPayload
		if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err == nil {
			if strings.TrimSpace(resume.ExternalSessionID) == "" {
				resume.ExternalSessionID = strings.TrimSpace(payload.ExternalSessionID)
			}
			if strings.TrimSpace(resume.ResumenContinuidad) == "" {
				resume.ResumenContinuidad = strings.TrimSpace(payload.ResumenContinuidad)
			}
		}
	}
	if payload := construirResumePayloadBootstrapDB(resume.ResumePayloadJSON, order, mailbox, checkpoint); payload != "" {
		resume.ResumePayloadJSON = payload
	}
	if resumen := construirResumenBootstrapDB(resume.ResumenContinuidad, order, mailbox, checkpoint); resumen != "" {
		resume.ResumenContinuidad = resumen
	}
	enriquecerResumeConContextoProyectoDB(&resume, agente, proyecto)
	if infoAgente, err := GetAgente(agente); err == nil && infoAgente != nil {
		enriquecerResumeConGobernanzaDB(&resume, strings.TrimSpace(infoAgente.Rol), agente, proyecto)
	}
	return resume, bootstrap, nil
}

func marcarBootstrapRuntimePreparationEjecutando(startOrder *RuntimeOrder, bootstrap *bootstrapRuntimeData, sesion *Sesion, handle *RuntimeHandle, runtime *RuntimeInstance) error {
	if bootstrap == nil || bootstrap.Order == nil || bootstrap.Order.ID <= 0 {
		return nil
	}
	resultado := map[string]any{
		"ok":            true,
		"bootstrap":     true,
		"lease_state":   "waiting_for_evidence",
		"checkpoint_id": checkpointIDOrZeroDB(bootstrap.Checkpoint),
		"mailbox_count": len(bootstrap.Mailbox),
		"mailbox_ids":   runtimeMailboxIDs(bootstrap.Mailbox),
		"order_tipo":    strings.TrimSpace(bootstrap.Order.Tipo),
		"ack_mode":      "runtime_transcript_or_tick",
		"continuidad":   bootstrap.Checkpoint != nil || len(bootstrap.Mailbox) > 0,
	}
	if startOrder != nil && startOrder.ID > 0 {
		resultado["start_order_id"] = startOrder.ID
	}
	if sesion != nil && sesion.ID > 0 {
		resultado["sesion_id"] = sesion.ID
	}
	if handle != nil && handle.ID > 0 {
		resultado["handle_id"] = handle.ID
	}
	if runtime != nil && runtime.ID > 0 {
		resultado["runtime_id"] = runtime.ID
	}
	return MarcarRuntimeOrderEstado(bootstrap.Order.ID, "ejecutando", mergeRuntimeOrderResultJSON(bootstrap.Order.ResultadoJSON, resultado), "")
}

func revertBootstrapRuntimePreparation(bootstrap *bootstrapRuntimeData) error {
	if bootstrap == nil || bootstrap.Order == nil || bootstrap.Order.ID <= 0 {
		return nil
	}
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado='pendiente',
		    started_at=NULL,
		    claimed_by='',
		    lease_token='',
		    lease_expires_at=NULL,
		updated_at=CURRENT_TIMESTAMP
		WHERE id = ?
		  AND estado IN ('tomada','ejecutando')`, bootstrap.Order.ID)
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(bootstrap.Order.ID)
}

func timeoutParaArranqueDurable(order *RuntimeOrder) time.Duration {
	if order == nil {
		return 0
	}
	payload := mapFromJSON(order.PayloadJSON)
	por := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "por", "")))
	if por == "orquesta" || por == "sistema" || por == "autonomia" {
		return 1500 * time.Millisecond
	}
	return 0
}

func marcarRuntimeMailboxConsumidoPorIDs(ids []int64) error {
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if err := MarcarRuntimeMailboxEntregado(id); err != nil {
			return err
		}
		if err := MarcarRuntimeMailboxConsumido(id); err != nil {
			return err
		}
	}
	return nil
}

func compactarMetadataRuntimeHandle(meta map[string]any) {
	runtimepolicy.CompactRuntimeHandleMetadata(meta)
}

func runtimeMailboxIDs(mailbox []*RuntimeMailboxMessage) []int64 {
	if len(mailbox) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(mailbox))
	for _, msg := range mailbox {
		if msg == nil || msg.ID <= 0 {
			continue
		}
		ids = append(ids, msg.ID)
	}
	return ids
}

func mergeRuntimeOrderResultJSON(prev string, extras map[string]any) string {
	base := map[string]any{}
	prev = strings.TrimSpace(prev)
	if prev != "" {
		_ = json.Unmarshal([]byte(prev), &base)
	}
	for k, v := range extras {
		base[k] = v
	}
	data, err := json.Marshal(base)
	if err != nil {
		return prev
	}
	return string(data)
}
