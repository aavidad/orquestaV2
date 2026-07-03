package orquestaestadovivo

import (
	"strings"
	"time"
)

const (
	CodigoConflictoProcesoVivoTrasTerminalV0 = "proceso_vivo_tras_terminal"
)

func decidirFaseCicloVidaV0(
	evidencias []EvidenciaEstadoV0,
	ahora time.Time,
	umbralHuerfano time.Duration,
	runRef string,
) (FaseCicloVidaV0, []ConflictoEstadoV0) {
	if len(evidencias) == 0 {
		return FaseDesconocidoV0, nil
	}

	tieneProcesoVivo := false
	tieneTerminal := false
	tieneTerminalAceptado := false
	fuentesProceso := make([]string, 0)
	fuentesTerminal := make([]string, 0)

	for _, evidencia := range evidencias {
		if evidencia.ProcesoVivo {
			tieneProcesoVivo = true
			fuentesProceso = append(fuentesProceso, evidencia.Fuente)
		}
		if evidencia.Terminal {
			tieneTerminal = true
			fuentesTerminal = append(fuentesTerminal, evidencia.Fuente)
			if evidencia.Aceptado {
				tieneTerminalAceptado = true
			}
		}
	}

	if tieneProcesoVivo && tieneTerminal {
		return FaseConflictoV0, []ConflictoEstadoV0{{
			RunRef:  runRef,
			Codigo:  CodigoConflictoProcesoVivoTrasTerminalV0,
			Fuentes: ordenarStringsUnicosV0(append(fuentesProceso, fuentesTerminal...)),
		}}
	}
	if tieneProcesoVivo {
		return FaseProcesoVivoV0, nil
	}
	if tieneTerminal {
		if tieneTerminalAceptado {
			return FaseTerminalAceptadoV0, nil
		}
		return FaseTerminalReworkV0, nil
	}
	if tieneEstadoBloqueadoV0(evidencias) {
		return FaseBloqueadoV0, nil
	}
	if soloMarkerSinEstadoNiProcesoV0(evidencias) {
		if markerViejoV0(evidencias, ahora, umbralHuerfano) {
			return FaseHuerfanoV0, nil
		}
		return FaseLanzadoV0, nil
	}
	if tieneEntregaParcialV0(evidencias) {
		return FaseEntregadoParcialV0, nil
	}
	if tieneOutboxOColaPendienteV0(evidencias) {
		return FaseSolicitadoV0, nil
	}
	if tieneWaitExternalOLanzadoV0(evidencias) {
		return FaseLanzadoV0, nil
	}

	return FaseDesconocidoV0, nil
}

func tieneEstadoBloqueadoV0(evidencias []EvidenciaEstadoV0) bool {
	for _, evidencia := range evidencias {
		switch tokenEstadoV0(evidencia.Estado) {
		case "blocked", "bloqueado":
			return true
		}
	}
	return false
}

func soloMarkerSinEstadoNiProcesoV0(evidencias []EvidenciaEstadoV0) bool {
	if len(evidencias) == 0 {
		return false
	}
	for _, evidencia := range evidencias {
		if tokenFuenteV0(evidencia.Fuente) != "run_marker" ||
			strings.TrimSpace(evidencia.Estado) != "" ||
			evidencia.ProcesoVivo ||
			evidencia.Terminal {
			return false
		}
	}
	return true
}

func markerViejoV0(
	evidencias []EvidenciaEstadoV0,
	ahora time.Time,
	umbralHuerfano time.Duration,
) bool {
	ultimoObservado, ok := ultimoObservadoEnV0(evidencias)
	if !ok {
		return false
	}
	return ahora.Sub(ultimoObservado) > umbralHuerfano
}

func ultimoObservadoEnV0(evidencias []EvidenciaEstadoV0) (time.Time, bool) {
	var ultimo time.Time
	ok := false
	for _, evidencia := range evidencias {
		observado, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(evidencia.ObservadoEn))
		if err != nil {
			continue
		}
		if !ok || observado.After(ultimo) {
			ultimo = observado
			ok = true
		}
	}
	return ultimo, ok
}

func tieneEntregaParcialV0(evidencias []EvidenciaEstadoV0) bool {
	for _, evidencia := range evidencias {
		if tokenFuenteV0(evidencia.Fuente) != "receipt" || evidencia.Terminal {
			continue
		}
		switch tokenEstadoV0(evidencia.Estado) {
		case "artifact", "artifacts", "artefacto", "artefactos", "delivery", "delivered",
			"delivery_registered", "entrega", "entregado_parcial", "partial", "partial_delivery",
			"submitted", "validating", "review_pending":
			return true
		}
		if len(evidencia.EvidenceRefs) > 0 && strings.TrimSpace(evidencia.Estado) != "" {
			return true
		}
	}
	return false
}

func tieneOutboxOColaPendienteV0(evidencias []EvidenciaEstadoV0) bool {
	for _, evidencia := range evidencias {
		fuente := tokenFuenteV0(evidencia.Fuente)
		estado := tokenEstadoV0(evidencia.Estado)
		if (fuente == "outbox" || fuente == "queue") &&
			(estado == "" || estado == "pending" || estado == "pendiente" || estado == "queued" || estado == "en_cola") {
			return true
		}
	}
	return false
}

func tieneWaitExternalOLanzadoV0(evidencias []EvidenciaEstadoV0) bool {
	for _, evidencia := range evidencias {
		switch tokenEstadoV0(evidencia.Estado) {
		case "wait_external", "waiting_external", "external_wait", "launched", "lanzado":
			return true
		}
	}
	return false
}

func tokenFuenteV0(valor string) string {
	return tokenEstadoV0(valor)
}

func tokenEstadoV0(valor string) string {
	token := strings.ToLower(strings.TrimSpace(valor))
	token = strings.ReplaceAll(token, "-", "_")
	token = strings.ReplaceAll(token, " ", "_")
	return token
}
