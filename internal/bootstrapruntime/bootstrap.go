package bootstrapruntime

import (
	"encoding/json"
	"fmt"
	"strings"

	"orquesta/db"
	"orquesta/runtimeagente"
)

type State struct {
	Order      *db.RuntimeOrder            `json:"order,omitempty"`
	Mailbox    []*db.RuntimeMailboxMessage `json:"mailbox,omitempty"`
	Checkpoint *db.RuntimeCheckpoint       `json:"checkpoint,omitempty"`
}

func Preparar(agente string, proyecto *db.Proyecto, ultima *db.Sesion) (runtimeagente.ResumeContext, *State, error) {
	var resume runtimeagente.ResumeContext
	if ultima != nil {
		resume = runtimeagente.ResumeContext{
			ExternalSessionID:  ultima.ExternalSessionID,
			ResumePayloadJSON:  ultima.ResumePayloadJSON,
			ResumenContinuidad: ultima.ResumenContinuidad,
			Branch:             ultima.Branch,
			CWD:                ultima.CWD,
		}
	}
	if proyecto == nil {
		return resume, nil, nil
	}

	state := &State{}
	order, err := db.PeekNextBootstrapRuntimeOrder(strings.TrimSpace(agente), &proyecto.ID)
	if err != nil {
		return resume, nil, err
	}
	state.Order = order

	estadoPendiente := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   strPtr(strings.TrimSpace(agente)),
		ProyectoID: &proyecto.ID,
		Estado:     &estadoPendiente,
	})
	if err != nil {
		return resume, nil, err
	}
	state.Mailbox = mailbox

	checkpoint, err := db.UltimoRuntimeCheckpoint(strings.TrimSpace(agente), &proyecto.ID)
	if err != nil {
		return resume, nil, err
	}
	state.Checkpoint = checkpoint

	if state.Order == nil && len(state.Mailbox) == 0 && state.Checkpoint == nil {
		return resume, nil, nil
	}

	ajustarResumeDesdeBootstrap(&resume, proyecto, state)
	if payload := construirResumePayloadBootstrap(resume.ResumePayloadJSON, state); payload != "" {
		resume.ResumePayloadJSON = payload
	}
	if resumen := construirResumenBootstrap(resume.ResumenContinuidad, state); resumen != "" {
		resume.ResumenContinuidad = resumen
	}
	enriquecerResumeConContextoProyecto(&resume, strings.TrimSpace(agente), proyecto)
	enriquecerResumeConGobernanza(&resume, strings.TrimSpace(agente), &proyecto.ID)
	return resume, state, nil
}

func ajustarResumeDesdeBootstrap(resume *runtimeagente.ResumeContext, proyecto *db.Proyecto, state *State) {
	if resume == nil || proyecto == nil || state == nil {
		return
	}
	if state.Checkpoint != nil {
		if strings.TrimSpace(resume.Branch) == "" {
			resume.Branch = strings.TrimSpace(state.Checkpoint.Branch)
		}
		if strings.TrimSpace(resume.CWD) == "" {
			resume.CWD = strings.TrimSpace(state.Checkpoint.CWD)
		}
	}
	if strings.TrimSpace(resume.CWD) == "" {
		resume.CWD = strings.TrimSpace(proyecto.RutaAbs)
	}
	if state.Order != nil && state.Order.Tipo == "handoff" {
		var payload db.HandoffPayload
		if err := json.Unmarshal([]byte(state.Order.PayloadJSON), &payload); err == nil {
			if strings.TrimSpace(resume.ExternalSessionID) == "" {
				resume.ExternalSessionID = strings.TrimSpace(payload.ExternalSessionID)
			}
			if strings.TrimSpace(resume.ResumenContinuidad) == "" {
				resume.ResumenContinuidad = strings.TrimSpace(payload.ResumenContinuidad)
			}
		}
	}
}

func construirResumePayloadBootstrap(prev string, state *State) string {
	prev = strings.TrimSpace(prev)
	if state == nil {
		return prev
	}
	envelope := map[string]any{}
	if state.Order != nil {
		envelope["runtime_order"] = map[string]any{
			"id":      state.Order.ID,
			"tipo":    state.Order.Tipo,
			"payload": rawJSONOrString(state.Order.PayloadJSON),
		}
	}
	if len(state.Mailbox) > 0 {
		items := make([]map[string]any, 0, len(state.Mailbox))
		for _, msg := range state.Mailbox {
			if msg == nil {
				continue
			}
			items = append(items, map[string]any{
				"id":          msg.ID,
				"from_agente": msg.FromAgente,
				"kind":        msg.Kind,
				"payload":     rawJSONOrString(msg.PayloadJSON),
			})
		}
		if len(items) > 0 {
			envelope["mailbox"] = items
		}
	}
	if state.Checkpoint != nil {
		envelope["checkpoint"] = map[string]any{
			"id":              state.Checkpoint.ID,
			"kind":            state.Checkpoint.CheckpointKind,
			"resumen":         state.Checkpoint.Resumen,
			"branch":          state.Checkpoint.Branch,
			"cwd":             state.Checkpoint.CWD,
			"payload":         rawJSONOrString(state.Checkpoint.PayloadJSON),
			"resume_strategy": state.Checkpoint.ResumeStrategy,
			"source":          state.Checkpoint.Source,
		}
	}
	if len(envelope) == 0 {
		return prev
	}
	return db.MergeResumePayloadEnvelope(prev, envelope)
}

func construirResumenBootstrap(prev string, state *State) string {
	partes := make([]string, 0, 4)
	prev = strings.TrimSpace(prev)
	if prev != "" {
		partes = append(partes, prev)
	}
	if state != nil && state.Order != nil {
		switch strings.TrimSpace(state.Order.Tipo) {
		case "handoff":
			var payload db.HandoffPayload
			if err := json.Unmarshal([]byte(state.Order.PayloadJSON), &payload); err == nil {
				resumen := strings.TrimSpace(payload.ResumenContinuidad)
				if resumen != "" {
					partes = append(partes, "Handoff: "+resumen)
				} else if strings.TrimSpace(payload.Motivo) != "" {
					partes = append(partes, "Handoff: "+strings.TrimSpace(payload.Motivo))
				}
			}
		default:
			partes = append(partes, "Orden pendiente aplicada: "+strings.TrimSpace(state.Order.Tipo))
		}
	}
	if state != nil && state.Checkpoint != nil && strings.TrimSpace(state.Checkpoint.Resumen) != "" {
		partes = append(partes, "Checkpoint: "+strings.TrimSpace(state.Checkpoint.Resumen))
	}
	if state != nil && len(state.Mailbox) > 0 {
		partes = append(partes, fmt.Sprintf("Mailbox: %d mensaje(s) inyectados", len(state.Mailbox)))
	}
	return strings.Join(partes, ". ")
}

func enriquecerResumeConContextoProyecto(resume *runtimeagente.ResumeContext, agente string, proyecto *db.Proyecto) {
	if resume == nil || proyecto == nil || !resumeTieneContexto(*resume) {
		return
	}
	contexto, resumen := db.BuildProjectContextSummary(agente, proyecto)
	if len(contexto) == 0 {
		return
	}
	if payload := db.AppendProjectContextPayload(resume.ResumePayloadJSON, contexto); payload != "" {
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

func enriquecerResumeConGobernanza(resume *runtimeagente.ResumeContext, agente string, proyectoID *int64) {
	if resume == nil || !resumeTieneContexto(*resume) {
		return
	}
	info, err := db.GetAgente(strings.TrimSpace(agente))
	if err != nil || info == nil {
		return
	}
	contexto, resumen := db.BuildGovernanceContextSummaryForContext(strings.TrimSpace(info.Rol), proyectoID, strings.TrimSpace(agente))
	if len(contexto) == 0 {
		return
	}
	if payload := db.AppendGovernanceCatalogPayload(resume.ResumePayloadJSON, contexto); payload != "" {
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

func resumeTieneContexto(resume runtimeagente.ResumeContext) bool {
	return strings.TrimSpace(resume.ExternalSessionID) != "" ||
		strings.TrimSpace(resume.ResumePayloadJSON) != "" ||
		strings.TrimSpace(resume.ResumenContinuidad) != ""
}

func checkpointIDOrZero(cp *db.RuntimeCheckpoint) int64 {
	if cp == nil {
		return 0
	}
	return cp.ID
}

func rawJSONOrString(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed
	}
	return raw
}

func strPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
