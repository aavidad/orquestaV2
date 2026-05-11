package orquestaweb

import orquestafactory "orquesta/modulos/orquesta-factory"

const NuevaAppBacklogPreviewSchemaV0 = "nueva_app_backlog_preview.v0"

type WebNuevaAppBacklogPreviewV0 struct {
	SchemaVersion    string                                  `json:"schema_version"`
	Fases            []WebNuevaAppBacklogPreviewFaseV0       `json:"fases"`
	Microtareas      []WebNuevaAppBacklogPreviewMicrotareaV0 `json:"microtareas"`
	TotalFases       int                                     `json:"total_fases"`
	TotalMicrotareas int                                     `json:"total_microtareas"`
	TieneBloqueos    bool                                    `json:"tiene_bloqueos"`
}

type WebNuevaAppBacklogPreviewFaseV0 struct {
	Key         string `json:"key"`
	Titulo      string `json:"titulo"`
	Objetivo    string `json:"objetivo"`
	Orden       int    `json:"orden"`
	Microtareas int    `json:"microtareas"`
}

type WebNuevaAppBacklogPreviewMicrotareaV0 struct {
	Key              string   `json:"key"`
	Fase             string   `json:"fase"`
	Titulo           string   `json:"titulo"`
	ModuloFrontera   string   `json:"modulo_frontera"`
	WriteSetPrevisto []string `json:"write_set_previsto"`
	Bloqueos         []string `json:"bloqueos"`
	CriterioCierre   string   `json:"criterio_cierre"`
}

func emptyBacklogPreviewV0() WebNuevaAppBacklogPreviewV0 {
	return WebNuevaAppBacklogPreviewV0{
		SchemaVersion: NuevaAppBacklogPreviewSchemaV0,
		Fases:         []WebNuevaAppBacklogPreviewFaseV0{},
		Microtareas:   []WebNuevaAppBacklogPreviewMicrotareaV0{},
	}
}

func backlogPreviewFromBacklogV0(backlog orquestafactory.BacklogInicialPropuestoV0) WebNuevaAppBacklogPreviewV0 {
	fases := backlogPreviewFasesFromBacklogV0(backlog.Fases, backlog.Microtareas)
	microtareas := backlogPreviewMicrotareasFromBacklogV0(backlog.Microtareas)
	return WebNuevaAppBacklogPreviewV0{
		SchemaVersion:    NuevaAppBacklogPreviewSchemaV0,
		Fases:            fases,
		Microtareas:      microtareas,
		TotalFases:       len(fases),
		TotalMicrotareas: len(microtareas),
		TieneBloqueos:    backlogPreviewHasBlockersV0(microtareas),
	}
}

func backlogPreviewFasesFromBacklogV0(fases []orquestafactory.FaseInicialV0, microtareas []orquestafactory.MicrotareaPropuestaV0) []WebNuevaAppBacklogPreviewFaseV0 {
	countByPhase := map[string]int{}
	for _, task := range microtareas {
		if fase := trimV0(task.Fase); fase != "" {
			countByPhase[fase]++
		}
	}
	out := make([]WebNuevaAppBacklogPreviewFaseV0, 0, len(fases))
	for _, fase := range fases {
		key := trimV0(fase.ID)
		out = append(out, WebNuevaAppBacklogPreviewFaseV0{
			Key:         key,
			Titulo:      trimV0(fase.Nombre),
			Objetivo:    trimV0(fase.Objetivo),
			Orden:       fase.Orden,
			Microtareas: countByPhase[key],
		})
	}
	if out == nil {
		return []WebNuevaAppBacklogPreviewFaseV0{}
	}
	return out
}

func backlogPreviewMicrotareasFromBacklogV0(values []orquestafactory.MicrotareaPropuestaV0) []WebNuevaAppBacklogPreviewMicrotareaV0 {
	out := make([]WebNuevaAppBacklogPreviewMicrotareaV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebNuevaAppBacklogPreviewMicrotareaV0{
			Key:              trimV0(value.ID),
			Fase:             trimV0(value.Fase),
			Titulo:           trimV0(value.Objetivo),
			ModuloFrontera:   moduloFronteraPreviewV0(value.ModuloSugerido, value.Contrato),
			WriteSetPrevisto: compactStringsV0(value.WriteSetPrevisto),
			Bloqueos:         compactStringsV0(value.Bloqueos),
			CriterioCierre:   trimV0(value.Validacion),
		})
	}
	if out == nil {
		return []WebNuevaAppBacklogPreviewMicrotareaV0{}
	}
	return out
}

func moduloFronteraPreviewV0(modulo, contrato string) string {
	modulo = trimV0(modulo)
	contrato = trimV0(contrato)
	switch {
	case modulo != "" && contrato != "":
		return modulo + " / " + contrato
	case modulo != "":
		return modulo
	default:
		return contrato
	}
}

func backlogPreviewHasBlockersV0(microtareas []WebNuevaAppBacklogPreviewMicrotareaV0) bool {
	for _, microtarea := range microtareas {
		if len(microtarea.Bloqueos) > 0 {
			return true
		}
	}
	return false
}
