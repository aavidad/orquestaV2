package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/internal/ollamapool"
	"orquesta/microprogramacionapp"
	"orquesta/runtimeagente"
	"orquesta/sesionesapp"
)

var ollamaPoolManager = ollamapool.NuevoGestor(endpointOllamaLocal(), clienteHTTPOllamaLocal())

func clienteHTTPOllamaLocal() *http.Client {
	return &http.Client{Timeout: timeoutClienteOllamaLocal()}
}

func timeoutClienteOllamaLocal() time.Duration {
	timeout := ollamapool.TimeoutClienteDefecto
	if raw := strings.TrimSpace(os.Getenv("ORQUESTA_OLLAMA_TIMEOUT_MS")); raw != "" {
		if ms, err := strconv.ParseInt(raw, 10, 64); err == nil && ms > 0 {
			timeout = time.Duration(ms) * time.Millisecond
		}
	}
	return timeout
}

func timeoutClienteOllamaLocalMS() int64 {
	return timeoutClienteOllamaLocal().Milliseconds()
}

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
	Agente   string                   `json:"agente"`
	Proyecto string                   `json:"proyecto"`
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
	RuntimeOrderID    int64  `json:"runtime_order_id"`
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
	modeloObjetivo := valorConFallback(strings.TrimSpace(req.Plan.Modelo), strings.TrimSpace(pool.ModeloPreferente))
	if modeloActivo, err := detectarModeloOllamaActivoIncompatible(modeloObjetivo); err != nil {
		apiError(w, http.StatusBadGateway, err)
		return
	} else if modeloActivo != "" {
		apiError(w, http.StatusConflict, fmt.Errorf("modelo ollama activo incompatible: %s; detenerlo antes de lanzar %s", modeloActivo, modeloObjetivo))
		return
	}
	resumenContinuidad := resolverResumenContinuidadPoolLocalCompartido(strings.TrimSpace(req.Agente), strings.TrimSpace(req.Proyecto))
	sesion, err := ollamaPoolManager.Lanzar(context.Background(), ollamapool.EntradaLanzamiento{
		Agente:             strings.TrimSpace(req.Agente),
		Proyecto:           strings.TrimSpace(req.Proyecto),
		PoolSlug:           strings.TrimSpace(pool.PoolSlug),
		SlotsMaximos:       pool.SlotsMaximos,
		Modelo:             modeloObjetivo,
		PerfilTarea:        strings.TrimSpace(req.Plan.PerfilTarea),
		Razonamiento:       strings.TrimSpace(req.Plan.Razonamiento),
		Sistema:            "MICROTAREA CERRADA. Sigue solo la especificacion activa y no abras frentes fuera del write_set.",
		ResumenContinuidad: resumenContinuidad,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	worktree := resolverWorktreeSesionPoolLocalCompartido(strings.TrimSpace(req.Agente), strings.TrimSpace(req.Proyecto))
	aplicarWorktreeSesionPoolLocalCompartido(sesion, worktree)
	if err := asegurarSesionActivaPoolLocalCompartido(strings.TrimSpace(req.Agente), strings.TrimSpace(req.Proyecto), sesion); err != nil {
		_ = ollamaPoolManager.Detener(sesion.HandleRef)
		apiError(w, http.StatusBadRequest, err)
		return
	}
	serverURL := endpointServidorDesdeRequest(r)
	apiWriteJSON(w, http.StatusOK, apiOllamaPoolLaunchResponse{
		ExternalSessionID: sesion.ID,
		HandleRef:         sesion.HandleRef,
		HandleKind:        "session",
		Metadata: map[string]any{
			"driver":                "ollama_pool_local",
			"transport":             "api",
			"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
			"endpoint":              serverURL,
			"input_path":            "/api/runtime/ollama-pool/input",
			"status_path":           "/api/runtime/ollama-pool/status",
			"stop_path":             "/api/runtime/ollama-pool/stop",
			"timeout_ms":            timeoutClienteOllamaLocalMS(),
			"pool_local":            true,
			"pool_slug":             sesion.PoolSlug,
			"modelo":                sesion.Modelo,
			"perfil_tarea":          sesion.PerfilTarea,
			"ollama_endpoint":       endpointOllamaLocal(),
			"external_session_id":   sesion.ID,
			"handle_ref":            sesion.HandleRef,
			"worktree_id":           valorIDWorktreePoolLocal(worktree),
			"ruta_worktree":         valorRutaWorktreePoolLocal(worktree),
			"branch_worktree":       valorBranchWorktreePoolLocal(worktree),
			"base_ref_worktree":     valorBaseRefWorktreePoolLocal(worktree),
		},
		Capabilities: map[string]any{
			"can_send_input":          true,
			"mailbox_delivery_mode":   runtimeagente.MailboxDeliveryInteractive,
			"can_pause":               false,
			"can_stop":                true,
			"can_checkpoint":          false,
			"can_resume":              false,
			"can_track_continuity":    true,
			"can_stop_without_reauth": true,
		},
	})
}

func detectarModeloOllamaActivoIncompatible(modeloObjetivo string) (string, error) {
	modeloObjetivo = strings.TrimSpace(modeloObjetivo)
	if modeloObjetivo == "" {
		return "", nil
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	items, err := nuevoGestorRuntimeModelosOllama(endpointOllamaLocal(), client).ListarModelosActivos()
	if err != nil {
		return "", nil
	}
	for _, item := range items {
		modeloActivo := strings.TrimSpace(item.Modelo)
		if modeloActivo == "" {
			modeloActivo = strings.TrimSpace(item.ID)
		}
		if modeloActivo == "" || strings.EqualFold(modeloActivo, modeloObjetivo) {
			continue
		}
		return modeloActivo, nil
	}
	return "", nil
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
	runtimeOrderID := req.RuntimeOrderID
	sesionBase, err := asegurarSesionPoolLocalCompartidoEnMemoria(handleRef)
	if err != nil {
		if runtimeOrderID > 0 {
			_ = runtimesService.MarcarRuntimeOrderDispatchFallido(runtimeOrderID, err.Error())
		}
		apiError(w, http.StatusBadRequest, err)
		return
	}
	sesionWorking, err := ollamaPoolManager.EnviarAsincrono(context.Background(), handleRef, strings.TrimSpace(req.Texto), func(result *ollamapool.ResultadoEntrada, err error) {
		defer func() {
			if recover() != nil {
				wakeControlPlaneRuntimeOrders()
			}
		}()
		if result != nil && result.Sesion != nil {
			_ = actualizarSesionActivaPoolLocalCompartido(result.Sesion)
		}
		if err != nil {
			if runtimeOrderID > 0 {
				_ = runtimesService.MarcarRuntimeOrderDispatchFallido(runtimeOrderID, err.Error())
			}
			wakeControlPlaneRuntimeOrders()
			return
		}
		if result == nil || result.Sesion == nil || strings.TrimSpace(result.Respuesta) == "" {
			wakeControlPlaneRuntimeOrders()
			return
		}
		proyectoID, resolveErr := sesionesAPIService.ResolveProjectID(strings.TrimSpace(result.Sesion.Proyecto))
		if resolveErr != nil {
			wakeControlPlaneRuntimeOrders()
			return
		}
		if _, transcriptErr := runtimesService.RegistrarSalidaObservada(strings.TrimSpace(result.Sesion.Agente), proyectoID, strings.TrimSpace(result.Respuesta)); transcriptErr != nil {
			wakeControlPlaneRuntimeOrders()
			return
		}
		_, _, entregaCompletada := procesarEntregaMicroprogramacionPoolLocal(result.Sesion.Agente, result.Sesion.Proyecto, result.Respuesta)
		if !entregaCompletada {
			wakeControlPlaneRuntimeOrders()
		}
	})
	if err != nil {
		if runtimeOrderID > 0 {
			_ = runtimesService.MarcarRuntimeOrderDispatchFallido(runtimeOrderID, err.Error())
		}
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if sesionWorking != nil {
		if err := actualizarSesionActivaPoolLocalCompartido(sesionWorking); err != nil {
			if runtimeOrderID > 0 {
				_ = runtimesService.MarcarRuntimeOrderDispatchFallido(runtimeOrderID, err.Error())
			}
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	if runtimeOrderID > 0 {
		if err := runtimesService.MarcarRuntimeOrderDispatchNotificado(runtimeOrderID, "worker notificado"); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{
		"ok":                      true,
		"handle_ref":              handleRef,
		"external_session":        strings.TrimSpace(sesionBase.ID),
		"runtime_order_id":        runtimeOrderID,
		"delivery_state":          "notified",
		"delivery_async":          true,
		"respuesta":               "",
		"ficheros_materializados": nil,
		"entrega_git":             nil,
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
	sesion, err := asegurarSesionPoolLocalCompartidoEnMemoria(handleRef)
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
	serverURL := endpointServidorDesdeRequest(r)
	worktree := resolverWorktreeSesionPoolLocalCompartido(strings.TrimSpace(sesion.Agente), strings.TrimSpace(sesion.Proyecto))
	aplicarWorktreeSesionPoolLocalCompartido(sesion, worktree)
	apiWriteJSON(w, http.StatusOK, map[string]any{
		"handle_state":  estadoHandle,
		"logical_state": sesion.Estado,
		"metadata": map[string]any{
			"driver":                "ollama_pool_local",
			"transport":             "api",
			"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
			"endpoint":              serverURL,
			"input_path":            "/api/runtime/ollama-pool/input",
			"status_path":           "/api/runtime/ollama-pool/status",
			"stop_path":             "/api/runtime/ollama-pool/stop",
			"timeout_ms":            timeoutClienteOllamaLocalMS(),
			"pool_local":            true,
			"pool_slug":             sesion.PoolSlug,
			"modelo":                sesion.Modelo,
			"perfil_tarea":          sesion.PerfilTarea,
			"resumen_continuidad":   sesion.ResumenContinuidad,
			"error_ultimo":          sesion.ErrorUltimo,
			"mensajes_totales":      len(sesion.Mensajes),
			"worktree_id":           valorIDWorktreePoolLocal(worktree),
			"ruta_worktree":         valorRutaWorktreePoolLocal(worktree),
			"branch_worktree":       valorBranchWorktreePoolLocal(worktree),
			"base_ref_worktree":     valorBaseRefWorktreePoolLocal(worktree),
		},
		"capabilities": map[string]any{
			"can_send_input":        true,
			"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
			"can_stop":              true,
			"can_pause":             false,
		},
	})
}

func endpointServidorDesdeRequest(r *http.Request) string {
	if r == nil || strings.TrimSpace(r.Host) == "" {
		return strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_URL"))
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.ToLower(forwarded)
	}
	return scheme + "://" + strings.TrimSpace(r.Host)
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
	sesion, err := asegurarSesionPoolLocalCompartidoEnMemoria(handleRef)
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

func asegurarSesionPoolLocalCompartidoEnMemoria(handleRef string) (*ollamapool.Sesion, error) {
	handleRef = strings.TrimSpace(handleRef)
	if handleRef == "" {
		return nil, fmt.Errorf("handle_ref obligatorio")
	}
	sesion, err := ollamaPoolManager.Estado(handleRef)
	if err == nil && sesion != nil {
		return sesion, nil
	}
	activa, err := sesionesAPIService.FindActiveSessionByExternalSessionID(handleRef)
	if err != nil {
		return nil, err
	}
	return ollamaPoolManager.RegistrarSesion(sesionPoolLocalCompartidoPersistida(activa))
}

func sesionPoolLocalCompartidoPersistida(activa *sesionesapp.Sesion) *ollamapool.Sesion {
	if activa == nil {
		return nil
	}
	modelo, perfilTarea, razonamiento, poolSlug, worktreeID, rutaWorktree, branchWorktree, baseRefWorktree := perfilSesionPoolLocalCompartidoPersistido(activa.ResumePayloadJSON)
	handleRef := strings.TrimSpace(activa.ExternalSessionID)
	if handleRef == "" {
		handleRef = fmt.Sprintf("ollama-pool-%d", activa.ID)
	}
	estado := "ready"
	if !activa.Activa || strings.EqualFold(strings.TrimSpace(activa.Estado), "cerrada") {
		estado = "stopped"
	}
	actualizadoEn := activa.Inicio.UTC()
	if activa.HeartbeatAt != nil && !activa.HeartbeatAt.IsZero() {
		actualizadoEn = activa.HeartbeatAt.UTC()
	}
	return &ollamapool.Sesion{
		ID:                  handleRef,
		HandleRef:           handleRef,
		Agente:              strings.TrimSpace(activa.Agente),
		Proyecto:            strings.TrimSpace(activa.ProyectoSlug),
		PoolSlug:            poolSlug,
		Modelo:              modelo,
		PerfilTarea:         perfilTarea,
		Razonamiento:        razonamiento,
		WorktreeID:          worktreeID,
		RutaWorktree:        rutaWorktree,
		BranchWorktree:      branchWorktree,
		BaseRefWorktree:     baseRefWorktree,
		MaxMensajesContexto: 6,
		Estado:              estado,
		ResumenContinuidad:  strings.TrimSpace(activa.ResumenContinuidad),
		CreadoEn:            activa.Inicio.UTC(),
		ActualizadoEn:       actualizadoEn,
	}
}

func perfilSesionPoolLocalCompartidoPersistido(resumePayload string) (string, string, string, string, int64, string, string, string) {
	envelope := db.ParseResumePayloadEnvelope(strings.TrimSpace(resumePayload))
	perfilMap, _ := envelope["perfil_ejecucion"].(map[string]any)
	modelo := strings.TrimSpace(stringDesdeAny(perfilMap["modelo"]))
	perfilTarea := strings.TrimSpace(stringDesdeAny(perfilMap["perfil_tarea"]))
	razonamiento := strings.TrimSpace(stringDesdeAny(perfilMap["razonamiento"]))
	poolSlug := strings.TrimSpace(stringDesdeAny(perfilMap["pool_slug"]))
	worktreeMap, _ := envelope["worktree"].(map[string]any)
	worktreeID := int64DesdeAny(worktreeMap["id"])
	rutaWorktree := strings.TrimSpace(stringDesdeAny(worktreeMap["ruta"]))
	branchWorktree := strings.TrimSpace(stringDesdeAny(worktreeMap["branch"]))
	baseRefWorktree := strings.TrimSpace(stringDesdeAny(worktreeMap["base_ref"]))
	if poolSlug == "" {
		poolSlug = strings.TrimSpace(stringDesdeAny(envelope["pool_slug"]))
	}
	return modelo, perfilTarea, razonamiento, poolSlug, worktreeID, rutaWorktree, branchWorktree, baseRefWorktree
}

func stringDesdeAny(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func int64DesdeAny(v any) int64 {
	switch typed := v.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

func resumePayloadSesionPoolLocalCompartido(sesion *ollamapool.Sesion) string {
	if sesion == nil {
		return ""
	}
	envelope := map[string]any{
		"driver":                "ollama_pool_local",
		"transport":             "api",
		"endpoint":              endpointServidorPoolLocalCompartido(),
		"input_path":            "/api/runtime/ollama-pool/input",
		"status_path":           "/api/runtime/ollama-pool/status",
		"stop_path":             "/api/runtime/ollama-pool/stop",
		"can_send_input":        true,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"pool_compartido":       true,
		"perfil_ejecucion": map[string]any{
			"perfil_tarea":          strings.TrimSpace(sesion.PerfilTarea),
			"modelo":                strings.TrimSpace(sesion.Modelo),
			"razonamiento":          strings.TrimSpace(sesion.Razonamiento),
			"driver":                "ollama_pool_local",
			"transport":             "api",
			"endpoint":              endpointServidorPoolLocalCompartido(),
			"input_path":            "/api/runtime/ollama-pool/input",
			"status_path":           "/api/runtime/ollama-pool/status",
			"stop_path":             "/api/runtime/ollama-pool/stop",
			"can_send_input":        true,
			"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
			"pool_slug":             strings.TrimSpace(sesion.PoolSlug),
		},
	}
	if sesion.WorktreeID > 0 || strings.TrimSpace(sesion.RutaWorktree) != "" || strings.TrimSpace(sesion.BranchWorktree) != "" || strings.TrimSpace(sesion.BaseRefWorktree) != "" {
		envelope["worktree"] = map[string]any{
			"id":       sesion.WorktreeID,
			"ruta":     strings.TrimSpace(sesion.RutaWorktree),
			"branch":   strings.TrimSpace(sesion.BranchWorktree),
			"base_ref": strings.TrimSpace(sesion.BaseRefWorktree),
		}
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return ""
	}
	return string(data)
}

func endpointServidorPoolLocalCompartido() string {
	if endpoint := strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_URL")); endpoint != "" {
		return endpoint
	}
	return "http://127.0.0.1:16543"
}

func resolverWorktreeSesionPoolLocalCompartido(agente, proyecto string) *db.Worktree {
	agente = strings.TrimSpace(agente)
	proyecto = strings.TrimSpace(proyecto)
	if agente == "" || proyecto == "" {
		return nil
	}
	item, err := gitService.ResolveActiveWorktree(proyecto, agente)
	if err != nil || item == nil {
		return nil
	}
	return item
}

func aplicarWorktreeSesionPoolLocalCompartido(sesion *ollamapool.Sesion, worktree *db.Worktree) {
	if sesion == nil || worktree == nil {
		return
	}
	sesion.WorktreeID = worktree.ID
	sesion.RutaWorktree = strings.TrimSpace(worktree.RutaAbs)
	sesion.BranchWorktree = strings.TrimSpace(worktree.Branch)
	sesion.BaseRefWorktree = strings.TrimSpace(worktree.BaseRef)
}

func valorIDWorktreePoolLocal(worktree *db.Worktree) any {
	if worktree == nil || worktree.ID <= 0 {
		return nil
	}
	return worktree.ID
}

func valorRutaWorktreePoolLocal(worktree *db.Worktree) any {
	if worktree == nil || strings.TrimSpace(worktree.RutaAbs) == "" {
		return nil
	}
	return strings.TrimSpace(worktree.RutaAbs)
}

func valorBranchWorktreePoolLocal(worktree *db.Worktree) any {
	if worktree == nil || strings.TrimSpace(worktree.Branch) == "" {
		return nil
	}
	return strings.TrimSpace(worktree.Branch)
}

func valorBaseRefWorktreePoolLocal(worktree *db.Worktree) any {
	if worktree == nil || strings.TrimSpace(worktree.BaseRef) == "" {
		return nil
	}
	return strings.TrimSpace(worktree.BaseRef)
}

func procesarEntregaMicroprogramacionPoolLocal(agente, proyectoRef, respuesta string) ([]string, map[string]any, bool) {
	agente = strings.TrimSpace(agente)
	proyectoRef = strings.TrimSpace(proyectoRef)
	respuesta = strings.TrimSpace(respuesta)
	if agente == "" || proyectoRef == "" || respuesta == "" {
		return nil, nil, false
	}
	if runtimeTranscriptTextoEsBootstrapPoolLocal(respuesta, "") {
		return nil, nil, false
	}
	proyectoID, err := sesionesAPIService.ResolveProjectID(proyectoRef)
	if err != nil {
		return nil, nil, false
	}
	ctx, err := runtimesService.ResolverContextoEntregaMicroprogramacionParaRespuesta(agente, proyectoID, respuesta)
	if err != nil || ctx == nil || ctx.EspecificacionID <= 0 {
		return nil, nil, false
	}
	if microprogramacionapp.FormatoSalidaUsaGitWorktree(ctx.FormatoSalida) {
		registro, err := runtimesService.RegistrarEntregaGitMicroprogramacionActiva(
			agente,
			proyectoID,
			proyectoRef,
			fmt.Sprintf("runtime_order=%d transcript=ollama_pool_local", ctx.RuntimeOrderID),
			"orquesta",
		)
		if err != nil || registro == nil || registro.Entrega == nil {
			return nil, nil, false
		}
		resultado := registro.Entrega
		return nil, map[string]any{
			"git_merge_id":        resultado.GitMergeID,
			"worktree_id":         resultado.WorktreeID,
			"ruta_worktree":       resultado.RutaWorktree,
			"source_branch":       resultado.SourceBranch,
			"target_branch":       resultado.TargetBranch,
			"head_commit":         resultado.HeadCommit,
			"archivos_entregados": resultado.ArchivosEntregados,
		}, true
	}
	resultado, err := runtimesService.MaterializarEntregaMicroprogramacionActivaDesdeRespuesta(
		agente,
		proyectoID,
		proyectoRef,
		respuesta,
		fmt.Sprintf("runtime_order=%d transcript=ollama_pool_local", ctx.RuntimeOrderID),
		"ollama_pool_local_materializada",
	)
	if err != nil {
		if respuestaPareceIntentoEntregaMicroprogramacion(respuesta) {
			_, _ = runtimesService.ReencolarCorreccionEntregaMicroprogramacion(ctx.RuntimeOrderID, err.Error())
		}
		return nil, nil, false
	}
	if resultado == nil {
		return nil, nil, false
	}
	return resultado.ArchivosMaterializados, nil, true
}

func respuestaPareceIntentoEntregaMicroprogramacion(respuesta string) bool {
	respuesta = strings.TrimSpace(respuesta)
	if respuesta == "" {
		return false
	}
	if len(microprogramacionapp.ExtraerArchivosEntrega(respuesta)) > 0 {
		return true
	}
	texto := strings.ToLower(respuesta)
	return strings.Contains(texto, "patch_unificado") ||
		strings.Contains(texto, "diff --git") ||
		strings.HasPrefix(texto, "--- ") ||
		strings.Contains(texto, "\n--- ")
}
