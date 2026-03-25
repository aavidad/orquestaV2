/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
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

type agenteControlPayload struct {
	Accion       string `json:"accion"`
	Proyecto     string `json:"proyecto,omitempty"`
	Conector     string `json:"conector,omitempty"`
	Modelo       string `json:"modelo,omitempty"`
	Razonamiento string `json:"razonamiento,omitempty"`
	Perfil       string `json:"perfil,omitempty"`
	Motivo       string `json:"motivo,omitempty"`
	Por          string `json:"por,omitempty"`
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
		if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			orderID, accion, err = encolarControlAgenteLocal(req)
		}
		if err != nil {
			return err
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
	agente := strings.TrimSpace(req.Agente)
	if agente == "" {
		return 0, "", fmt.Errorf("agente obligatorio")
	}
	accion, err := normalizarAccionControlAgente(req.Accion)
	if err != nil {
		return 0, "", err
	}
	if err := validarAgenteControlExiste(agente); err != nil {
		return 0, "", err
	}

	var (
		proyectoID *int64
		handleID   *int64
		runtimeID  *int64
	)
	proyectoRef := strings.TrimSpace(req.Proyecto)
	if accion == agenteControlAccionStart && proyectoRef == "" {
		return 0, "", fmt.Errorf("debes indicar proyecto para arrancar el agente")
	}
	if proyectoRef != "" {
		proyecto, err := runtimesService.GetProject(proyectoRef)
		if err != nil {
			return 0, "", err
		}
		proyectoID = &proyecto.ID
	}

	handle, err := resolverHandleControlAgente(agente, proyectoID)
	if err != nil {
		return 0, "", err
	}
	if accion == agenteControlAccionStart && handle != nil {
		return 0, "", fmt.Errorf("el agente %s ya tiene un runtime handle activo", agente)
	}
	if handle != nil {
		handleID = &handle.ID
		if handle.RuntimeID != nil {
			runtimeID = handle.RuntimeID
		}
	}

	payloadJSON, err := json.Marshal(agenteControlPayload{
		Accion:       accion,
		Proyecto:     proyectoRef,
		Conector:     strings.TrimSpace(req.Conector),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
		Motivo:       strings.TrimSpace(req.Motivo),
		Por:          valorConFallback(strings.TrimSpace(req.Por), "orquesta"),
	})
	if err != nil {
		return 0, "", err
	}

	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      agente,
		ProyectoID:  proyectoID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        accion,
		PayloadJSON: string(payloadJSON),
	})
	if err != nil {
		return 0, "", err
	}

	detalle := fmt.Sprintf("%s agente=%s proyecto=%s", accion, agente, valorConFallback(proyectoRef, ""))
	if motivo := strings.TrimSpace(req.Motivo); motivo != "" {
		detalle += " motivo=" + motivo
	}
	db.Audit(valorConFallback(strings.TrimSpace(req.Por), "orquesta"), "control_agente_"+accion, "runtime_order", orderID, detalle)
	return orderID, accion, nil
}

func validarAgenteControlExiste(agente string) error {
	if _, err := db.GetAgente(strings.TrimSpace(agente)); err != nil {
		return err
	}
	return nil
}

func resolverHandleControlAgente(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if proyectoID != nil {
		handle, err := runtimesService.GetActiveRuntimeHandleForProject(agente, proyectoID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
	}
	return runtimesService.GetActiveRuntimeHandle(agente)
}

func normalizarAccionControlAgente(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "start", "arrancar", "arranque", "iniciar":
		return agenteControlAccionStart, nil
	case "pause", "pausar", "pausa":
		return agenteControlAccionPause, nil
	case "resume", "continuar", "reanudar":
		return agenteControlAccionResume, nil
	case "stop", "detener", "parar":
		return agenteControlAccionStop, nil
	default:
		return "", fmt.Errorf("acción de control no soportada: %s", strings.TrimSpace(v))
	}
}
