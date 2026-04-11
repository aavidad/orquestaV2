/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/runtimesapp"
)

const (
	agenteControlAccionStart  = "start"
	agenteControlAccionPause  = "pause"
	agenteControlAccionResume = "resume"
	agenteControlAccionStop   = "stop"
)

type apiAgenteControlRequest struct {
	Agente       string `json:"agente"`
	Proyecto     string `json:"proyecto"`
	Accion       string `json:"accion"`
	Conector     string `json:"conector"`
	Modelo       string `json:"modelo"`
	Razonamiento string `json:"razonamiento"`
	Perfil       string `json:"perfil"`
	Motivo       string `json:"motivo"`
	Por          string `json:"por"`
}

type apiAgenteControlResponse struct {
	OK     bool   `json:"ok"`
	ID     int64  `json:"id"`
	Agente string `json:"agente"`
	Accion string `json:"accion"`
}

var agenteControlCmd = &cobra.Command{
	Use:   "control <arrancar|pausar|continuar|detener> <agente>",
	Short: "Encola una orden de ciclo de vida del agente en el control plane",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")
		conector, _ := cmd.Flags().GetString("conector")
		modelo, _ := cmd.Flags().GetString("modelo")
		razonamiento, _ := cmd.Flags().GetString("razonamiento")
		perfil, _ := cmd.Flags().GetString("perfil")
		motivo, _ := cmd.Flags().GetString("motivo")
		por, _ := cmd.Flags().GetString("por")

		req := apiAgenteControlRequest{
			Agente:       strings.TrimSpace(args[1]),
			Proyecto:     strings.TrimSpace(proyecto),
			Accion:       strings.TrimSpace(args[0]),
			Conector:     strings.TrimSpace(conector),
			Modelo:       strings.TrimSpace(modelo),
			Razonamiento: strings.TrimSpace(razonamiento),
			Perfil:       strings.TrimSpace(perfil),
			Motivo:       strings.TrimSpace(motivo),
			Por:          strings.TrimSpace(por),
		}

		orderID, accion, ok, err := encolarControlAgentePorAPI(req)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("agente control")
		}

		fmt.Printf("✓ Orden %s #%d encolada para %s\n", accion, orderID, req.Agente)
		return nil
	},
}

func init() {
	agenteControlCmd.Flags().String("proyecto", "", "Proyecto destino de la orden")
	agenteControlCmd.Flags().String("conector", "", "Conector a usar para arrancar")
	agenteControlCmd.Flags().String("modelo", "", "Modelo a usar para arrancar o reanudar")
	agenteControlCmd.Flags().String("razonamiento", "", "Reasoning effort deseado")
	agenteControlCmd.Flags().String("perfil", "", "Perfil de tarea del agente")
	agenteControlCmd.Flags().String("motivo", "", "Motivo operativo de la orden")
	agenteControlCmd.Flags().String("por", "orquesta", "Actor que solicita la orden")
	agenteCmd.AddCommand(agenteControlCmd)
}

func apiHandlerAgenteControl(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiAgenteControlRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	orderID, accion, err := encolarControlAgenteLocal(req)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, apiAgenteControlResponse{
		OK:     true,
		ID:     orderID,
		Agente: strings.TrimSpace(req.Agente),
		Accion: accion,
	})
}

func encolarControlAgentePorAPI(req apiAgenteControlRequest) (int64, string, bool, error) {
	var resp apiAgenteControlResponse
	ok, err := apiPost("/api/agente/control", req, &resp)
	if !ok || err != nil {
		return 0, "", ok, err
	}
	return resp.ID, strings.TrimSpace(resp.Accion), true, nil
}

func encolarControlAgenteLocal(req apiAgenteControlRequest) (int64, string, error) {
	orderID, accion, err := runtimesService.EnqueueAgentControl(runtimesapp.AgentControlRequest{
		Agente:       strings.TrimSpace(req.Agente),
		Proyecto:     strings.TrimSpace(req.Proyecto),
		Accion:       strings.TrimSpace(req.Accion),
		Conector:     strings.TrimSpace(req.Conector),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
		Motivo:       strings.TrimSpace(req.Motivo),
		Por:          strings.TrimSpace(req.Por),
	})
	if err != nil {
		return 0, "", err
	}
	resetStatusSnapshotCache()
	return orderID, accion, nil
}

// apiAgenteLanzarRequest agrupa los parámetros del endpoint POST /api/agente/lanzar.
type apiAgenteLanzarRequest struct {
	Agente       string `json:"agente"`
	Proyecto     string `json:"proyecto"`
	Conector     string `json:"conector"`
	Modelo       string `json:"modelo"`
	Razonamiento string `json:"razonamiento"`
	Perfil       string `json:"perfil"`
	Motivo       string `json:"motivo"`
	Por          string `json:"por"`
}

type apiAgenteLanzarResponse struct {
	OK       bool                     `json:"ok"`
	OrderID  int64                    `json:"order_id"`
	Agente   string                   `json:"agente"`
	Proyecto agentesapp.ProjectBundle `json:"proyecto"`
	Conector agentesapp.ConnectorBundle `json:"conector"`
}

// apiHandlerAgenteLanzar combina BuildPrepare + encolar orden "start" en un solo POST.
// Es el punto de entrada unificado para arrancar un agente desde el control plane.
func apiHandlerAgenteLanzar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiAgenteLanzarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	req.Agente = strings.TrimSpace(req.Agente)
	req.Proyecto = strings.TrimSpace(req.Proyecto)
	if req.Agente == "" || req.Proyecto == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("agente y proyecto son obligatorios"))
		return
	}

	// Validar y resolver el plan completo antes de encolar.
	prep, err := agentesService.BuildPrepare(agentesapp.PrepareInput{
		Agente:       req.Agente,
		Proyecto:     req.Proyecto,
		Conector:     strings.TrimSpace(req.Conector),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	por := strings.TrimSpace(req.Por)
	if por == "" {
		por = "orquesta"
	}
	orderID, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:       req.Agente,
		Proyecto:     req.Proyecto,
		Accion:       agenteControlAccionStart,
		Conector:     prep.Conector.Slug,
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
		Motivo:       strings.TrimSpace(req.Motivo),
		Por:          por,
	})
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	apiWriteJSON(w, http.StatusCreated, apiAgenteLanzarResponse{
		OK:       true,
		OrderID:  orderID,
		Agente:   prep.Agente,
		Proyecto: prep.Proyecto,
		Conector: prep.Conector,
	})
}

func resolverHandleControlAgente(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return runtimesService.ResolveControlHandle(agente, proyectoID)
}

func resolverHandleEntregaAgente(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return runtimesService.ResolveDeliveryHandle(agente, proyectoID)
}

func resolverHandleEntregaTranscript(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error) {
	return runtimesService.ResolveTranscriptDeliveryHandle(item)
}
