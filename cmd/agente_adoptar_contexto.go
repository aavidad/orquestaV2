package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/runtimeagente"
	"orquesta/sesionesapp"
)

type apiAgenteAdoptarContextoRequest struct {
	Agente            string `json:"agente"`
	Proyecto          string `json:"proyecto"`
	Conector          string `json:"conector"`
	CWD               string `json:"cwd"`
	Herramienta       string `json:"herramienta"`
	Branch            string `json:"branch"`
	ExternalSessionID string `json:"external_session_id"`
	ResumePayload     string `json:"resume_payload_json"`
	Resumen           string `json:"resumen_continuidad"`
	Nota              string `json:"nota"`
	Host              string `json:"host"`
	PID               int64  `json:"pid"`
}

type apiAgenteAdoptarContextoResponse struct {
	Sesion         *db.Sesion            `json:"sesion"`
	Checkpoint     *db.RuntimeCheckpoint `json:"checkpoint,omitempty"`
	RuntimeOrderID *int64                `json:"runtime_order_id,omitempty"`
	Rol            string                `json:"rol,omitempty"`
	Gobernanza     *db.GovernanceCatalog `json:"governance_catalog,omitempty"`
}

func apiHandlerAgenteAdoptarContexto(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiAgenteAdoptarContextoRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resp, err := adoptarContextoAgente(req)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, resp)
}

func adoptarContextoAgente(req apiAgenteAdoptarContextoRequest) (*apiAgenteAdoptarContextoResponse, error) {
	req.Agente = strings.TrimSpace(req.Agente)
	req.Proyecto = strings.TrimSpace(req.Proyecto)
	req.Conector = strings.TrimSpace(req.Conector)
	req.CWD = strings.TrimSpace(req.CWD)
	req.Herramienta = strings.TrimSpace(req.Herramienta)
	req.Branch = strings.TrimSpace(req.Branch)
	req.ExternalSessionID = strings.TrimSpace(req.ExternalSessionID)
	req.ResumePayload = strings.TrimSpace(req.ResumePayload)
	req.Resumen = strings.TrimSpace(req.Resumen)
	req.Nota = strings.TrimSpace(req.Nota)
	req.Host = strings.TrimSpace(req.Host)
	if req.Agente == "" || req.Proyecto == "" {
		return nil, fmt.Errorf("debes indicar agente y proyecto")
	}

	agente, err := agentesService.GetAgent(req.Agente)
	if err != nil {
		return nil, err
	}
	proyecto, err := sesionesAPIService.GetProject(req.Proyecto)
	if err != nil {
		return nil, err
	}
	dbProyecto, err := db.GetProyecto(strconv.FormatInt(proyecto.ID, 10))
	if err != nil {
		return nil, err
	}

	rol := strings.TrimSpace(agente.Rol)
	creadaPorAdopcion := false
	existente, err := sesionesAPIService.GetActiveSession(agente.Nombre, proyecto.Slug)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if existente == nil {
		started, err := sesionesAPIService.StartContext(sesionesapp.StartContextInput{
			Agente:            agente.Nombre,
			Proyecto:          proyecto.Slug,
			Conector:          req.Conector,
			CWD:               req.CWD,
			Herramienta:       req.Herramienta,
			ExternalSessionID: req.ExternalSessionID,
			ResumePayload:     req.ResumePayload,
			Resumen:           req.Resumen,
			Branch:            req.Branch,
			Host:              req.Host,
			PID:               req.PID,
		})
		if err != nil {
			return nil, err
		}
		existente = started.Sesion
		creadaPorAdopcion = true
		if strings.TrimSpace(started.Rol) != "" {
			rol = strings.TrimSpace(started.Rol)
		}
	}

	if req.Herramienta == "" {
		req.Herramienta = firstNonEmpty(existente.Herramienta, req.Conector, runtimeagente.ConectorPorDefectoAgente(req.Agente))
	}
	prevResume := db.SanitizeResumeContextForProject(runtimeagente.ResumeContext{
		ExternalSessionID:  strings.TrimSpace(existente.ExternalSessionID),
		ResumePayloadJSON:  strings.TrimSpace(existente.ResumePayloadJSON),
		ResumenContinuidad: strings.TrimSpace(existente.ResumenContinuidad),
		Branch:             strings.TrimSpace(existente.Branch),
		CWD:                strings.TrimSpace(existente.CWD),
	}, dbProyecto)
	if req.CWD == "" {
		req.CWD = firstNonEmpty(prevResume.CWD, proyecto.RutaAbs)
	}
	if req.Branch == "" {
		req.Branch = strings.TrimSpace(prevResume.Branch)
	}
	if req.ExternalSessionID == "" {
		req.ExternalSessionID = strings.TrimSpace(prevResume.ExternalSessionID)
	}

	contextoGob, resumenGob := db.BuildGovernanceContextSummaryForContext(rol, &proyecto.ID, agente.Nombre)
	catalogo, err := gobernanzaService.ResolveCatalogForContext("", proyecto.Slug, agente.Nombre)
	if err != nil {
		return nil, err
	}
	resumePayload, err := construirResumePayloadAdoptado(prevResume.ResumePayloadJSON, req, dbProyecto, contextoGob)
	if err != nil {
		return nil, err
	}
	resumen := construirResumenContextoAdoptado(prevResume.ResumenContinuidad, req.Resumen, resumenGob, dbProyecto, req)
	estado := "activa"
	sesion, err := sesionesAPIService.SaveActiveSession(agente.Nombre, proyecto.Slug, db.SesionUpdate{
		CWD:                stringPtr(req.CWD),
		Herramienta:        stringPtr(req.Herramienta),
		ExternalSessionID:  stringPtr(req.ExternalSessionID),
		ResumePayloadJSON:  &resumePayload,
		ResumenContinuidad: &resumen,
		Branch:             stringPtr(req.Branch),
		Host:               stringPtr(req.Host),
		PID:                pidPtr(req.PID),
		Heartbeat:          true,
		Estado:             &estado,
	})
	if err != nil {
		return nil, err
	}
	if creadaPorAdopcion {
		if err := db.AparcarSesionActiva(agente.Nombre, &proyecto.ID); err != nil {
			return nil, err
		}
		sesion, err = db.GetSesionByID(sesion.ID)
		if err != nil {
			return nil, err
		}
	}

	checkpointID, err := runtimesService.CreateRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         agente.Nombre,
		ProyectoID:     &proyecto.ID,
		SesionID:       &sesion.ID,
		CheckpointKind: "contexto_adoptado",
		Resumen:        resumen,
		Branch:         req.Branch,
		CWD:            req.CWD,
		PayloadJSON:    resumePayload,
		ResumeStrategy: "resume",
		Source:         fmt.Sprintf("agente_adoptar:%s:%d", agente.Nombre, sesion.ID),
	})
	if err != nil {
		return nil, err
	}
	checkpoint, err := runtimesService.GetRuntimeCheckpoint(checkpointID)
	if err != nil {
		return nil, err
	}

	var orderID *int64
	handle, err := runtimesService.GetOperationalRuntimeHandleAgentProject(agente.Nombre, &proyecto.ID)
	if err != nil {
		return nil, err
	}
	if handle == nil {
		handle, err = runtimesService.GetActiveRuntimeHandleAgentProject(agente.Nombre, &proyecto.ID)
	}
	if err != nil {
		return nil, err
	}
	if creadaPorAdopcion {
		handle = nil
	}
	if creadaPorAdopcion || requiereTakeoverAdoptado(sesion, handle) {
		if existente := continuidadPendienteID(agente.Nombre, proyecto.ID); existente != nil {
			orderID = existente
		} else {
			payload := map[string]any{
				"accion":    "start",
				"proyecto":  proyecto.Slug,
				"motivo":    "adopt_context",
				"por":       "orquesta",
				"sesion_id": sesion.ID,
			}
			if req.Conector != "" {
				payload["conector"] = req.Conector
			}
			if req.Nota != "" {
				payload["nota"] = req.Nota
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			id, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
				Agente:      agente.Nombre,
				ProyectoID:  &proyecto.ID,
				Tipo:        "start",
				PayloadJSON: string(payloadJSON),
			})
			if err != nil {
				return nil, err
			}
			orderID = &id
		}
	}

	return &apiAgenteAdoptarContextoResponse{
		Sesion:         sesion,
		Checkpoint:     checkpoint,
		RuntimeOrderID: orderID,
		Rol:            rol,
		Gobernanza:     catalogo,
	}, nil
}

func construirResumePayloadAdoptado(prev string, req apiAgenteAdoptarContextoRequest, proyecto *db.Proyecto, contextoGob map[string]any) (string, error) {
	prev = db.SanitizeResumePayloadForProject(prev, proyecto)
	envelope := db.ParseResumePayloadEnvelope(prev)
	if strings.TrimSpace(req.ResumePayload) != "" {
		var incoming map[string]any
		if err := json.Unmarshal([]byte(req.ResumePayload), &incoming); err == nil {
			for k, v := range incoming {
				envelope[k] = v
			}
		} else {
			envelope["resume_payload_input"] = req.ResumePayload
		}
	}
	rutaAbs := db.RutaProyectoEfectiva(proyecto.ID, proyecto.RutaAbs, req.CWD)
	envelope["project_context"] = map[string]any{
		"id":       proyecto.ID,
		"slug":     proyecto.Slug,
		"nombre":   proyecto.Nombre,
		"ruta_abs": rutaAbs,
	}
	if contextoGob != nil {
		envelope["governance_catalog"] = contextoGob
	}
	envelope["adopted_context"] = map[string]any{
		"source":              "api_agente_adoptar_contexto",
		"adopted_at":          time.Now().UTC().Format(time.RFC3339),
		"cwd":                 req.CWD,
		"branch":              req.Branch,
		"herramienta":         req.Herramienta,
		"external_session_id": req.ExternalSessionID,
		"nota":                req.Nota,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func construirResumenContextoAdoptado(previo, input, resumenGob string, proyecto *db.Proyecto, req apiAgenteAdoptarContextoRequest) string {
	previo = db.SanitizeContinuitySummaryForProject(previo, proyecto)
	partes := []string{}
	if item := strings.TrimSpace(previo); item != "" {
		partes = append(partes, item)
	}
	if item := strings.TrimSpace(input); item != "" {
		partes = append(partes, item)
	}
	adopcion := fmt.Sprintf("Contexto adoptado por Orquesta sobre %s", strings.TrimSpace(proyecto.Slug))
	if strings.TrimSpace(req.Branch) != "" {
		adopcion += " en rama " + strings.TrimSpace(req.Branch)
	}
	if strings.TrimSpace(req.CWD) != "" {
		adopcion += " (" + strings.TrimSpace(req.CWD) + ")"
	}
	partes = append(partes, adopcion)
	if item := strings.TrimSpace(req.Nota); item != "" {
		partes = append(partes, "Nota: "+item)
	}
	if item := strings.TrimSpace(resumenGob); item != "" && !strings.Contains(strings.Join(partes, " "), item) {
		partes = append(partes, item)
	}
	return unirPartesUnicas(partes)
}

func unirPartesUnicas(items []string) string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return strings.Join(out, ". ")
}

func hayContinuidadPendiente(agente string, proyectoID int64) bool {
	return continuidadPendienteID(agente, proyectoID) != nil
}

func continuidadPendienteID(agente string, proyectoID int64) *int64 {
	estado := "pendiente"
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return nil
	}
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch strings.TrimSpace(order.Tipo) {
		case "start", "resume", "handoff":
			id := order.ID
			return &id
		}
	}
	return nil
}

func requiereTakeoverAdoptado(sesion *db.Sesion, handle *db.RuntimeHandle) bool {
	if sesion == nil {
		return false
	}
	if handle == nil {
		return true
	}
	if strings.TrimSpace(handle.Estado) != "activo" {
		return true
	}
	switch strings.TrimSpace(handle.HandleKind) {
	case "process":
		return false
	case "session":
		if strings.TrimSpace(sesion.ExternalSessionID) == "" && (sesion.PID == nil || *sesion.PID == 0) {
			return true
		}
		if ext := strings.TrimSpace(sesion.ExternalSessionID); ext != "" && strings.TrimSpace(handle.HandleRef) == ext {
			return false
		}
		if strings.TrimSpace(handle.HandleRef) == strconv.FormatInt(sesion.ID, 10) {
			return true
		}
	}
	return false
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := strings.TrimSpace(v)
	return &s
}

func pidPtr(pid int64) *int64 {
	if pid <= 0 {
		return nil
	}
	v := pid
	return &v
}

var agenteAdoptarContextoCmd = &cobra.Command{
	Use:   "adoptar-contexto <agente>",
	Short: "Entrega el contexto actual al orquestador para que continúe ese frente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")
		if strings.TrimSpace(proyecto) == "" {
			return fmt.Errorf("debes indicar --proyecto")
		}
		cwd, _ := cmd.Flags().GetString("cwd")
		if strings.TrimSpace(cwd) == "" {
			if actual, err := os.Getwd(); err == nil {
				cwd = actual
			}
		}
		branch, _ := cmd.Flags().GetString("branch")
		if strings.TrimSpace(branch) == "" {
			branch = detectarBranchGit(strings.TrimSpace(cwd))
		}
		herramienta, _ := cmd.Flags().GetString("herramienta")
		if strings.TrimSpace(herramienta) == "" {
			herramienta = runtimeagente.ConectorPorDefectoAgente(strings.TrimSpace(args[0]))
		}
		conector, _ := cmd.Flags().GetString("conector")
		externalSessionID, _ := cmd.Flags().GetString("external-session-id")
		resumePayload, _ := cmd.Flags().GetString("resume-payload")
		resumen, _ := cmd.Flags().GetString("resumen")
		nota, _ := cmd.Flags().GetString("nota")
		host, _ := os.Hostname()

		resp, ok, err := adoptarContextoAgentePorAPI(apiAgenteAdoptarContextoRequest{
			Agente:            strings.TrimSpace(args[0]),
			Proyecto:          strings.TrimSpace(proyecto),
			Conector:          strings.TrimSpace(conector),
			CWD:               strings.TrimSpace(cwd),
			Herramienta:       strings.TrimSpace(herramienta),
			Branch:            strings.TrimSpace(branch),
			ExternalSessionID: strings.TrimSpace(externalSessionID),
			ResumePayload:     strings.TrimSpace(resumePayload),
			Resumen:           strings.TrimSpace(resumen),
			Nota:              strings.TrimSpace(nota),
			Host:              strings.TrimSpace(host),
		})
		if !ok {
			return serverFirstCommandError("agente adoptar-contexto")
		}
		if err != nil {
			return err
		}
		fmt.Printf("✓ Contexto adoptado para %s en %s\n", args[0], proyecto)
		if resp.Sesion != nil {
			fmt.Printf("  Sesión: %d\n", resp.Sesion.ID)
		}
		if resp.Checkpoint != nil {
			fmt.Printf("  Checkpoint: %d (%s)\n", resp.Checkpoint.ID, resp.Checkpoint.CheckpointKind)
		}
		if resp.RuntimeOrderID != nil {
			fmt.Printf("  Runtime order: %d\n", *resp.RuntimeOrderID)
		}
		if resp.Gobernanza != nil && strings.TrimSpace(resp.Gobernanza.ResolucionActual) != "" {
			fmt.Printf("  Gobernanza: %s\n", strings.TrimSpace(resp.Gobernanza.ResolucionActual))
		}
		return nil
	},
}

func detectarBranchGit(cwd string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return ""
	}
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return ""
	}
	return branch
}
