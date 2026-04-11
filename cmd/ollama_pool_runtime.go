package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/internal/ollamapool"
	"orquesta/runtimeagente"
	"orquesta/sesionesapp"
)

var ollamaPoolManager = ollamapool.NuevoGestor(endpointOllamaLocal(), nil)

func newTestOllamaPoolManager(endpoint string) *ollamapool.Gestor {
	return ollamapool.NuevoGestor(endpoint, nil)
}

type proveedorPoolLocalOllama struct{}

func (proveedorPoolLocalOllama) DescribirPoolLocalCompartido(slug string) (*capacidadapp.TelemetriaPoolLocal, error) {
	telemetria := ollamaPoolManager.DescribirPool(strings.TrimSpace(slug))
	if telemetria == nil {
		return nil, nil
	}
	actualizadoEn := telemetria.ActualizadoEn
	return &capacidadapp.TelemetriaPoolLocal{
		PoolSlug:               strings.TrimSpace(telemetria.PoolSlug),
		SlotsActivos:           telemetria.SlotsActivos,
		SesionesLogicasActivas: telemetria.SesionesLogicasActivas,
		SesionesReady:          telemetria.SesionesReady,
		SesionesWorking:        telemetria.SesionesWorking,
		SesionesFailed:         telemetria.SesionesFailed,
		ActualizadoEn:          &actualizadoEn,
	}, nil
}

func init() {
	capacidadService.SetPoolLocalProvider(proveedorPoolLocalOllama{})
}

type apiOllamaPoolLaunchRequest struct {
	Agente   string `json:"agente"`
	Proyecto string `json:"proyecto"`
	Plan     runtimeagente.LaunchPlan `json:"plan"`
}

type apiOllamaPoolLaunchResponse struct {
	ExternalSessionID string         `json:"external_session_id"`
	HandleRef         string         `json:"handle_ref"`
	HandleKind        string         `json:"handle_kind"`
	Metadata          map[string]any `json:"metadata"`
	Capabilities      map[string]any `json:"capabilities"`
}

type apiOllamaPoolHandleRequest struct {
	HandleRef         string `json:"handle_ref"`
	ExternalSessionID string `json:"external_session_id"`
	Texto             string `json:"texto"`
}

func endpointOllamaLocal() string {
	if value := strings.TrimSpace(os.Getenv("ORQUESTA_OLLAMA_ENDPOINT")); value != "" {
		return strings.TrimRight(value, "/")
	}
	if value := strings.TrimSpace(os.Getenv("OLLAMA_HOST")); value != "" {
		value = strings.TrimRight(value, "/")
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			return value
		}
		return "http://" + value
	}
	return "http://127.0.0.1:11434"
}

func apiHandlerOllamaPoolLaunch(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiOllamaPoolLaunchRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	pool, err := resolverPoolLocalCompartidoParaLaunch(strings.TrimSpace(req.Agente), strings.TrimSpace(req.Proyecto), strings.TrimSpace(req.Plan.PerfilTarea))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resumenContinuidad := resolverResumenContinuidadPoolLocalCompartido(strings.TrimSpace(req.Agente), strings.TrimSpace(req.Proyecto))
	sesion, err := ollamaPoolManager.Lanzar(context.Background(), ollamapool.EntradaLanzamiento{
		Agente:             strings.TrimSpace(req.Agente),
		Proyecto:           strings.TrimSpace(req.Proyecto),
		PoolSlug:           strings.TrimSpace(pool.PoolSlug),
		SlotsMaximos:       pool.SlotsMaximos,
		Modelo:             valorConFallback(strings.TrimSpace(req.Plan.Modelo), strings.TrimSpace(pool.ModeloPreferente)),
		PerfilTarea:        strings.TrimSpace(req.Plan.PerfilTarea),
		Razonamiento:       strings.TrimSpace(req.Plan.Razonamiento),
		Sistema:            "MICROTAREA CERRADA. Sigue solo la especificacion activa y no abras frentes fuera del write_set.",
		ResumenContinuidad: resumenContinuidad,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := asegurarSesionActivaPoolLocalCompartido(strings.TrimSpace(req.Agente), strings.TrimSpace(req.Proyecto), sesion); err != nil {
		_ = ollamaPoolManager.Detener(sesion.HandleRef)
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiOllamaPoolLaunchResponse{
		ExternalSessionID: sesion.ID,
		HandleRef:         sesion.HandleRef,
		HandleKind:        "session",
		Metadata: map[string]any{
			"driver":              "ollama_pool_local",
			"transport":           "api",
			"timeout_ms":          180000,
			"pool_local":          true,
			"pool_slug":           sesion.PoolSlug,
			"modelo":              sesion.Modelo,
			"perfil_tarea":        sesion.PerfilTarea,
			"ollama_endpoint":     endpointOllamaLocal(),
			"external_session_id": sesion.ID,
			"handle_ref":          sesion.HandleRef,
		},
		Capabilities: map[string]any{
			"can_send_input":          true,
			"can_pause":               false,
			"can_stop":                true,
			"can_checkpoint":          false,
			"can_resume":              false,
			"can_track_continuity":    true,
			"can_stop_without_reauth": true,
		},
	})
}

func apiHandlerOllamaPoolInput(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiOllamaPoolHandleRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	handleRef := strings.TrimSpace(req.HandleRef)
	if handleRef == "" {
		handleRef = strings.TrimSpace(req.ExternalSessionID)
	}
	result, err := ollamaPoolManager.Enviar(context.Background(), handleRef, strings.TrimSpace(req.Texto))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if result != nil && result.Sesion != nil {
		if err := actualizarSesionActivaPoolLocalCompartido(result.Sesion); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"handle_ref": handleRef,
		"respuesta":  result.Respuesta,
	})
}

func apiHandlerOllamaPoolStatus(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiOllamaPoolHandleRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	handleRef := strings.TrimSpace(req.HandleRef)
	if handleRef == "" {
		handleRef = strings.TrimSpace(req.ExternalSessionID)
	}
	sesion, err := ollamaPoolManager.Estado(handleRef)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	estadoHandle := "ready"
	if sesion.Estado == "stopped" {
		estadoHandle = "stopped"
	} else if sesion.Estado == "failed" {
		estadoHandle = "failed"
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{
		"handle_state":  estadoHandle,
		"logical_state": sesion.Estado,
		"metadata": map[string]any{
			"driver":              "ollama_pool_local",
			"transport":           "api",
			"timeout_ms":          180000,
			"pool_local":          true,
			"pool_slug":           sesion.PoolSlug,
			"modelo":              sesion.Modelo,
			"perfil_tarea":        sesion.PerfilTarea,
			"resumen_continuidad": sesion.ResumenContinuidad,
			"error_ultimo":        sesion.ErrorUltimo,
			"mensajes_totales":    len(sesion.Mensajes),
		},
		"capabilities": map[string]any{
			"can_send_input": true,
			"can_stop":       true,
			"can_pause":      false,
		},
	})
}

func apiHandlerOllamaPoolStop(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiOllamaPoolHandleRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	handleRef := strings.TrimSpace(req.HandleRef)
	if handleRef == "" {
		handleRef = strings.TrimSpace(req.ExternalSessionID)
	}
	if handleRef == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("handle_ref obligatorio"))
		return
	}
	sesion, err := ollamaPoolManager.Estado(handleRef)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := ollamaPoolManager.Detener(handleRef); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := marcarSesionActivaPoolLocalCompartidoDetenida(sesion); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "handle_ref": handleRef})
}

func asegurarSesionActivaPoolLocalCompartido(agente, proyecto string, sesion *ollamapool.Sesion) error {
	if sesion == nil {
		return fmt.Errorf("sesion de pool obligatoria")
	}
	existente, err := sesionesAPIService.GetActiveSession(strings.TrimSpace(agente), strings.TrimSpace(proyecto))
	if err == nil && existente != nil {
		return actualizarSesionActivaPoolLocalCompartido(sesion)
	}
	started, err := sesionesAPIService.StartContext(sesionesapp.StartContextInput{
		Agente:            strings.TrimSpace(agente),
		Proyecto:          strings.TrimSpace(proyecto),
		Conector:          "ollama_pool_local",
		Herramienta:       "ollama_pool_local",
		ExternalSessionID: strings.TrimSpace(sesion.ID),
		ResumePayload:     resumePayloadSesionPoolLocalCompartido(sesion),
		Resumen:           strings.TrimSpace(sesion.ResumenContinuidad),
	})
	if err != nil {
		return err
	}
	if started == nil || started.Sesion == nil {
		return fmt.Errorf("no se pudo crear sesion activa para %s", strings.TrimSpace(agente))
	}
	return nil
}

func actualizarSesionActivaPoolLocalCompartido(sesion *ollamapool.Sesion) error {
	if sesion == nil {
		return fmt.Errorf("sesion de pool obligatoria")
	}
	externalSessionID := strings.TrimSpace(sesion.ID)
	herramienta := "ollama_pool_local"
	resumePayload := resumePayloadSesionPoolLocalCompartido(sesion)
	resumen := resumenSesionPoolLocalCompartido(sesion)
	estado := estadoSesionDesdePoolLocalCompartido(sesion.Estado)
	_, err := sesionesAPIService.SaveActiveSession(strings.TrimSpace(sesion.Agente), strings.TrimSpace(sesion.Proyecto), sesionesapp.SesionUpdate{
		ExternalSessionID:  &externalSessionID,
		Herramienta:        &herramienta,
		ResumePayloadJSON:  &resumePayload,
		ResumenContinuidad: &resumen,
		Estado:             &estado,
		Heartbeat:          true,
	})
	return err
}

func marcarSesionActivaPoolLocalCompartidoDetenida(sesion *ollamapool.Sesion) error {
	if sesion == nil {
		return nil
	}
	externalSessionID := strings.TrimSpace(sesion.ID)
	herramienta := "ollama_pool_local"
	resumePayload := resumePayloadSesionPoolLocalCompartido(sesion)
	resumen := resumenSesionPoolLocalCompartido(sesion)
	estado := estadoSesionDesdePoolLocalCompartido("stopped")
	_, err := sesionesAPIService.SaveActiveSession(strings.TrimSpace(sesion.Agente), strings.TrimSpace(sesion.Proyecto), sesionesapp.SesionUpdate{
		ExternalSessionID:  &externalSessionID,
		Herramienta:        &herramienta,
		ResumePayloadJSON:  &resumePayload,
		ResumenContinuidad: &resumen,
		Estado:             &estado,
		Heartbeat:          true,
	})
	return err
}

func estadoSesionDesdePoolLocalCompartido(estado string) string {
	switch strings.ToLower(strings.TrimSpace(estado)) {
	case "working", "ready", "running", "":
		return "activa"
	case "paused", "pausada":
		return "pausada"
	case "failed", "fallida":
		return "fallida"
	case "stopped", "stale", "closed", "cerrada":
		return "cerrada"
	default:
		return "activa"
	}
}

func resumenSesionPoolLocalCompartido(sesion *ollamapool.Sesion) string {
	if sesion == nil {
		return ""
	}
	if resumen := strings.TrimSpace(sesion.ResumenContinuidad); resumen != "" {
		return resumen
	}
	partes := make([]string, 0, 2)
	for i := len(sesion.Mensajes) - 1; i >= 0 && len(partes) < 2; i-- {
		msg := sesion.Mensajes[i]
		rol := strings.ToLower(strings.TrimSpace(msg.Rol))
		if rol == "system" {
			continue
		}
		texto := strings.Join(strings.Fields(strings.TrimSpace(msg.Contenido)), " ")
		if texto == "" {
			continue
		}
		if len(texto) > 120 {
			texto = strings.TrimSpace(texto[:117]) + "..."
		}
		switch rol {
		case "assistant":
			partes = append([]string{"RESPUESTA: " + texto}, partes...)
		case "user":
			partes = append([]string{"SOLICITUD: " + texto}, partes...)
		default:
			partes = append([]string{strings.ToUpper(rol) + ": " + texto}, partes...)
		}
	}
	return strings.TrimSpace(strings.Join(partes, " | "))
}

func resolverPoolLocalCompartidoParaLaunch(agente, proyecto, perfil string) (*capacidadapp.PoolLocalCompartido, error) {
	input := db.ResolverPoliticaInput{
		ProyectoSlug: strings.TrimSpace(proyecto),
		PerfilTarea:  strings.TrimSpace(perfil),
	}
	if nombre := strings.TrimSpace(agente); nombre != "" {
		input.AgenteNombre = &nombre
	}
	resolucion, err := capacidadService.ResolveModelPolicy(input)
	if err != nil {
		return nil, err
	}
	if resolucion == nil || strings.TrimSpace(resolucion.PoolSlug) == "" {
		return nil, fmt.Errorf("no existe pool local compartido resuelto para el perfil %q", strings.TrimSpace(perfil))
	}
	pool, err := capacidadService.DescribirPoolLocalCompartido(strings.TrimSpace(resolucion.PoolSlug))
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, fmt.Errorf("pool local %q no disponible", strings.TrimSpace(resolucion.PoolSlug))
	}
	return pool, nil
}

func resolverResumenContinuidadPoolLocalCompartido(agente, proyecto string) string {
	agente = strings.TrimSpace(agente)
	proyecto = strings.TrimSpace(proyecto)
	if agente == "" {
		return ""
	}
	if activa, err := sesionesAPIService.GetActiveSession(agente, proyecto); err == nil && activa != nil {
		if resumen := strings.TrimSpace(activa.ResumenContinuidad); resumen != "" {
			return resumen
		}
	}
	if ultima, err := sesionesAPIService.GetLastSession(agente, proyecto); err == nil && ultima != nil {
		return strings.TrimSpace(ultima.ResumenContinuidad)
	}
	return ""
}

func resumePayloadSesionPoolLocalCompartido(sesion *ollamapool.Sesion) string {
	if sesion == nil {
		return ""
	}
	envelope := map[string]any{
		"perfil_ejecucion": map[string]any{
			"perfil_tarea": strings.TrimSpace(sesion.PerfilTarea),
			"modelo":       strings.TrimSpace(sesion.Modelo),
			"razonamiento": strings.TrimSpace(sesion.Razonamiento),
			"driver":       "ollama_pool_local",
			"pool_slug":    strings.TrimSpace(sesion.PoolSlug),
		},
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return ""
	}
	return string(data)
}
