/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/runtimesapp"
)

type apiRuntimeTraceResponse struct {
	Trace *runtimesapp.RuntimeRawTrace `json:"trace"`
}

var runtimeTrazaCmd = &cobra.Command{
	Use:   "traza",
	Short: "Muestra la traza raw PTY de un handle o del runtime activo de un agente",
	RunE: func(cmd *cobra.Command, args []string) error {
		handleID, _ := cmd.Flags().GetInt64("handle-id")
		agente, _ := cmd.Flags().GetString("agente")
		proyecto, _ := cmd.Flags().GetString("proyecto")
		maxBytes, _ := cmd.Flags().GetInt("bytes")
		if handleID <= 0 && strings.TrimSpace(agente) == "" {
			return fmt.Errorf("debes indicar --handle-id o --agente")
		}

		query := url.Values{}
		if handleID > 0 {
			query.Set("handle_id", strconv.FormatInt(handleID, 10))
		}
		if strings.TrimSpace(agente) != "" {
			query.Set("agente", strings.TrimSpace(agente))
		}
		if strings.TrimSpace(proyecto) != "" {
			query.Set("proyecto", strings.TrimSpace(proyecto))
		}
		if maxBytes > 0 {
			query.Set("bytes", strconv.Itoa(maxBytes))
		}

		if trace, ok, err := cargarRuntimeTraceDesdeAPI(query); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeTraza(trace)
		} else if runtimeModoRecuperacionLocalExplicito() {
			trace, err := cargarRuntimeTraceRecuperacionLocal(query)
			if err != nil {
				return err
			}
			return imprimirRuntimeTraza(trace)
		}
		return serverFirstCommandError("runtime traza")
	},
}

func cargarRuntimeTraceDesdeAPI(query url.Values) (*runtimesapp.RuntimeRawTrace, bool, error) {
	var resp apiRuntimeTraceResponse
	ok, err := apiGetQuery("/api/runtime-trace", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Trace, true, nil
}

func cargarRuntimeTraceRecuperacionLocal(query url.Values) (*runtimesapp.RuntimeRawTrace, error) {
	req := runtimesapp.RuntimeTraceRequest{
		Agente:   strings.TrimSpace(query.Get("agente")),
		Proyecto: strings.TrimSpace(query.Get("proyecto")),
	}
	if raw := strings.TrimSpace(query.Get("handle_id")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("handle_id inválido")
		}
		req.HandleID = value
	}
	if raw := strings.TrimSpace(query.Get("bytes")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("bytes inválido")
		}
		req.MaxBytes = value
	}
	return runtimesService.ReadRuntimeTrace(req)
}

func imprimirRuntimeTraza(trace *runtimesapp.RuntimeRawTrace) error {
	if trace == nil {
		fmt.Println("No hay traza raw disponible.")
		return nil
	}
	fmt.Printf("Handle:      %d\n", trace.HandleID)
	fmt.Printf("Agente:      %s\n", valorVacio(trace.Agente))
	fmt.Printf("Proyecto:    %s\n", valorVacio(trace.Proyecto))
	fmt.Printf("Driver:      %s\n", valorVacio(trace.Driver))
	fmt.Printf("Working dir: %s\n", valorVacio(trace.WorkingDir))
	fmt.Printf("Trace dir:   %s\n", valorVacio(trace.TraceDir))
	fmt.Printf("Manifest:    %s\n", valorVacio(trace.TraceManifest))
	fmt.Printf("Log:         %s\n", valorVacio(trace.LogPath))
	fmt.Printf("Disponible:  %s\n", boolTextTrace(trace.Available))
	if strings.TrimSpace(trace.Reason) != "" {
		fmt.Printf("Motivo:      %s\n", strings.TrimSpace(trace.Reason))
	}
	fmt.Printf("Bytes:       total=%d leidos=%d truncada=%s\n", trace.TotalBytes, trace.BytesRead, boolTextTrace(trace.Truncated))
	if !trace.Available || trace.RawTail == "" {
		return nil
	}
	fmt.Println("\n----- PTY RAW -----")
	fmt.Print(trace.RawTail)
	if !strings.HasSuffix(trace.RawTail, "\n") {
		fmt.Println()
	}
	return nil
}

func boolTextTrace(v bool) string {
	if v {
		return "si"
	}
	return "no"
}

func apiHandlerRuntimeTrace(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	req := runtimesapp.RuntimeTraceRequest{
		Agente:   strings.TrimSpace(r.URL.Query().Get("agente")),
		Proyecto: strings.TrimSpace(r.URL.Query().Get("proyecto")),
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("handle_id")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("handle_id inválido"))
			return
		}
		req.HandleID = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("bytes")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("bytes inválido"))
			return
		}
		req.MaxBytes = value
	}
	if req.HandleID <= 0 && strings.TrimSpace(req.Agente) == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("debes indicar handle_id o agente"))
		return
	}

	trace, err := runtimesService.ReadRuntimeTrace(req)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, runtimesapp.ErrRuntimeTraceHandleNoEncontrado):
			status = http.StatusNotFound
		case strings.Contains(strings.ToLower(err.Error()), "debes indicar"),
			strings.Contains(strings.ToLower(err.Error()), "proyecto"):
			status = http.StatusBadRequest
		}
		apiError(w, status, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeTraceResponse{Trace: trace})
}

func init() {
	runtimeTrazaCmd.Flags().Int64("handle-id", 0, "Runtime handle a inspeccionar")
	runtimeTrazaCmd.Flags().String("agente", "", "Agente del que tomar la traza activa")
	runtimeTrazaCmd.Flags().String("proyecto", "", "Proyecto del runtime activo")
	runtimeTrazaCmd.Flags().Int("bytes", 8192, "Numero maximo de bytes raw a devolver")
	runtimeCmd.AddCommand(runtimeTrazaCmd)
}
