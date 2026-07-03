package orquestaestadovivo

import (
	"sort"
	"strings"
	"time"
)

func ConstruirProyeccionCicloVidaV0(
	evidencias []EvidenciaEstadoV0,
	ahora time.Time,
	umbralHuerfano time.Duration,
) ProyeccionCicloVidaV0 {
	proyeccion := ProyeccionCicloVidaV0{
		SchemaVersion: ProyeccionCicloVidaSchemaV0,
		GeneradaEn:    ahora.UTC().Format(time.RFC3339Nano),
	}
	if len(evidencias) == 0 {
		proyeccion.Nodos = []NodoCicloVidaV0{{Fase: FaseDesconocidoV0}}
		return proyeccion
	}

	grupos := agruparEvidenciasCicloVidaV0(evidencias)
	claves := make([]string, 0, len(grupos))
	for clave := range grupos {
		claves = append(claves, clave)
	}
	sort.Strings(claves)

	proyeccion.Nodos = make([]NodoCicloVidaV0, 0, len(claves))
	for _, clave := range claves {
		evidenciasNodo := copiarYOrdenarEvidenciasV0(grupos[clave])
		nodo := nodoBaseCicloVidaV0(evidenciasNodo)
		fase, conflictos := decidirFaseCicloVidaV0(evidenciasNodo, ahora, umbralHuerfano, refConflictoV0(clave, nodo))
		nodo.Fase = fase
		nodo.Evidencias = evidenciasNodo
		nodo.Conflictos = conflictos
		proyeccion.Nodos = append(proyeccion.Nodos, nodo)
	}

	return proyeccion
}

func agruparEvidenciasCicloVidaV0(evidencias []EvidenciaEstadoV0) map[string][]EvidenciaEstadoV0 {
	grupos := make(map[string][]EvidenciaEstadoV0)
	for _, evidencia := range evidencias {
		clave := strings.TrimSpace(evidencia.RunRef)
		if clave == "" {
			clave = strings.TrimSpace(evidencia.GoalRef)
		}
		grupos[clave] = append(grupos[clave], evidencia)
	}
	return grupos
}

func nodoBaseCicloVidaV0(evidencias []EvidenciaEstadoV0) NodoCicloVidaV0 {
	nodo := NodoCicloVidaV0{}
	for _, evidencia := range evidencias {
		if nodo.RunRef == "" && strings.TrimSpace(evidencia.RunRef) != "" {
			nodo.RunRef = strings.TrimSpace(evidencia.RunRef)
		}
		if nodo.GoalRef == "" && strings.TrimSpace(evidencia.GoalRef) != "" {
			nodo.GoalRef = strings.TrimSpace(evidencia.GoalRef)
		}
		if nodo.ExternalGoalRef == "" && strings.TrimSpace(evidencia.ExternalGoalRef) != "" {
			nodo.ExternalGoalRef = strings.TrimSpace(evidencia.ExternalGoalRef)
		}
	}
	return nodo
}

func refConflictoV0(clave string, nodo NodoCicloVidaV0) string {
	if nodo.RunRef != "" {
		return nodo.RunRef
	}
	return clave
}

func copiarYOrdenarEvidenciasV0(evidencias []EvidenciaEstadoV0) []EvidenciaEstadoV0 {
	copia := make([]EvidenciaEstadoV0, len(evidencias))
	copy(copia, evidencias)
	for i := range copia {
		copia[i].EvidenceRefs = append([]string(nil), copia[i].EvidenceRefs...)
		sort.Strings(copia[i].EvidenceRefs)
	}
	sort.SliceStable(copia, func(i, j int) bool {
		return claveEvidenciaV0(copia[i]) < claveEvidenciaV0(copia[j])
	})
	return copia
}

func claveEvidenciaV0(evidencia EvidenciaEstadoV0) string {
	return strings.Join([]string{
		evidencia.RunRef,
		evidencia.GoalRef,
		evidencia.ExternalGoalRef,
		evidencia.Fuente,
		evidencia.Estado,
		boolOrdenableV0(evidencia.ProcesoVivo),
		boolOrdenableV0(evidencia.Terminal),
		boolOrdenableV0(evidencia.Aceptado),
		evidencia.ObservadoEn,
		strings.Join(evidencia.EvidenceRefs, "\x00"),
	}, "\x01")
}

func boolOrdenableV0(valor bool) string {
	if valor {
		return "1"
	}
	return "0"
}

func ordenarStringsUnicosV0(valores []string) []string {
	vistos := make(map[string]struct{}, len(valores))
	ordenados := make([]string, 0, len(valores))
	for _, valor := range valores {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		if _, ok := vistos[valor]; ok {
			continue
		}
		vistos[valor] = struct{}{}
		ordenados = append(ordenados, valor)
	}
	sort.Strings(ordenados)
	return ordenados
}
