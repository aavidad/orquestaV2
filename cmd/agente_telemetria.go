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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/internal/controlruntime"
)

var agentePresupuestoCmd = &cobra.Command{
	Use:   "presupuesto",
	Short: "Lista la telemetría de presupuesto visible por agente",
	RunE: func(cmd *cobra.Command, args []string) error {
		activos, _ := cmd.Flags().GetBool("activos")
		jsonOut, _ := cmd.Flags().GetBool("json")
		refresh, _ := cmd.Flags().GetBool("refresh")
		cached, _ := cmd.Flags().GetBool("cached")
		agenteFiltro, _ := cmd.Flags().GetString("agente")
		if refresh || !cached {
			if _, ok, err := refrescarAgentesPresupuestoPorAPI(agenteFiltro); err != nil {
				return err
			} else if !ok {
				return agenteErrorServerFirst()
			}
		}
		resp, ok, err := listarAgentesPresupuestoPorAPI(activos, agenteFiltro)
		if err != nil {
			return err
		}
		if !ok {
			return agenteErrorServerFirst()
		}
		if jsonOut {
			return imprimirJSON(resp)
		}
		for _, agente := range resp.Agentes {
			if agente == nil {
				continue
			}
			fmt.Printf("%-16s", agente.Nombre)
			if detalle := resumenCuotaAgente(agente); detalle != "" {
				fmt.Printf(" %s", detalle)
			}
			fmt.Println()
		}
		return nil
	},
}

var agenteCuentasCmd = &cobra.Command{
	Use:   "cuentas",
	Short: "Lista nombre de agente y cuenta observada",
	RunE: func(cmd *cobra.Command, args []string) error {
		activos, _ := cmd.Flags().GetBool("activos")
		jsonOut, _ := cmd.Flags().GetBool("json")
		cached, _ := cmd.Flags().GetBool("cached")
		if !cached {
			if _, ok, err := refrescarAgentesPresupuestoPorAPI(""); err != nil {
				return err
			} else if !ok {
				return agenteErrorServerFirst()
			}
		}
		resp, ok, err := listarAgentesCuentasPorAPI(activos)
		if err != nil {
			return err
		}
		if !ok {
			return agenteErrorServerFirst()
		}
		if jsonOut {
			return imprimirJSON(resp)
		}
		for _, agente := range resp.Agentes {
			fmt.Printf("%-16s", agente.Nombre)
			fmt.Printf(" %s", resumenCuentaObservadaAgente(agente))
			fmt.Println()
		}
		return nil
	},
}

var agenteRankingCuentasCmd = &cobra.Command{
	Use:   "ranking-cuentas",
	Short: "Lista cuentas observadas ordenadas por presupuesto disponible",
	RunE: func(cmd *cobra.Command, args []string) error {
		activos, _ := cmd.Flags().GetBool("activos")
		jsonOut, _ := cmd.Flags().GetBool("json")
		cached, _ := cmd.Flags().GetBool("cached")
		if !cached {
			if _, ok, err := refrescarAgentesPresupuestoPorAPI(""); err != nil {
				return err
			} else if !ok {
				return agenteErrorServerFirst()
			}
		}
		resp, ok, err := listarAgentesRankingCuentasPorAPI(activos)
		if err != nil {
			return err
		}
		if !ok {
			return agenteErrorServerFirst()
		}
		if jsonOut {
			return imprimirJSON(resp)
		}
		for i, cuenta := range resp.Cuentas {
			fmt.Printf("%d. %-18s %s", i+1, cuenta.CuentaClave, resumenRankingCuenta(cuenta))
			fmt.Println()
		}
		return nil
	},
}

var agenteObservarCuentaCmd = &cobra.Command{
	Use:   "observar-cuenta <agente>",
	Short: "Registra de forma canónica la cuenta observada de un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := strings.TrimSpace(args[0])
		email, _ := cmd.Flags().GetString("email")
		usuario, _ := cmd.Flags().GetString("usuario")
		fuente, _ := cmd.Flags().GetString("fuente")
		if strings.TrimSpace(fuente) == "" {
			fuente = "manual_observed_identity"
		}
		var observedAt *time.Time
		if raw, _ := cmd.Flags().GetString("observed-at"); strings.TrimSpace(raw) != "" {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
			if err != nil {
				return fmt.Errorf("observed-at debe ir en RFC3339: %w", err)
			}
			parsed = parsed.UTC()
			observedAt = &parsed
		}
		ok, err := observarCuentaAgentePorAPI(agente, email, usuario, fuente, observedAt)
		if err != nil {
			return err
		}
		if !ok {
			return agenteErrorServerFirst()
		}
		resp, ok, err := listarAgentesCuentasPorAPI(false)
		if err != nil {
			return err
		}
		if !ok {
			return agenteErrorServerFirst()
		}
		for _, item := range resp.Agentes {
			if strings.EqualFold(strings.TrimSpace(item.Nombre), agente) {
				fmt.Printf("%-16s %s\n", item.Nombre, resumenCuentaObservadaAgente(item))
				return nil
			}
		}
		return nil
	},
}

type agenteStatusVivoItem struct {
	Nombre        string     `json:"nombre"`
	HandleID      int64      `json:"handle_id,omitempty"`
	HandleKind    string     `json:"handle_kind,omitempty"`
	HandleRef     string     `json:"handle_ref,omitempty"`
	HandleEstado  string     `json:"handle_estado,omitempty"`
	Driver        string     `json:"driver,omitempty"`
	Fuente        string     `json:"fuente,omitempty"`
	CuentaEmail   string     `json:"cuenta_email,omitempty"`
	PlanType      string     `json:"plan_type,omitempty"`
	SessionID     string     `json:"session_id,omitempty"`
	FiveHourLeft  *float64   `json:"five_hour_left,omitempty"`
	FiveHourReset *time.Time `json:"five_hour_reset,omitempty"`
	WeeklyLeft    *float64   `json:"weekly_left,omitempty"`
	WeeklyReset   *time.Time `json:"weekly_reset,omitempty"`
	ObservedAt    *time.Time `json:"observed_at,omitempty"`
	RawOutput     string     `json:"raw_output,omitempty"`
	Error         string     `json:"error,omitempty"`
}

type agenteStatusVivoResponse struct {
	Generado string                 `json:"generado"`
	Agentes  []agenteStatusVivoItem `json:"agentes"`
}

var agenteStatusVivoCmd = &cobra.Command{
	Use:        "status-vivo",
	Short:      "Depuración legacy de /status vivo por PTY",
	Hidden:     true,
	Deprecated: "superficie legacy de depuración; use presupuesto/cuentas basados en tmux, profile-status y artefactos estructurados",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !legacyLiveStatusEnabled() {
			return fmt.Errorf("status-vivo legacy deshabilitado; exporte ORQUESTA_ALLOW_LEGACY_LIVE_STATUS=1 solo para depuración puntual")
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		persist, _ := cmd.Flags().GetBool("persist")
		agenteFiltro, _ := cmd.Flags().GetString("agente")

		nombres, err := listarAgentesStatusVivo(agenteFiltro)
		if err != nil {
			return err
		}
		resp := agenteStatusVivoResponse{
			Generado: time.Now().UTC().Format(time.RFC3339Nano),
			Agentes:  make([]agenteStatusVivoItem, 0, len(nombres)),
		}
		for _, nombre := range nombres {
			item, err := consultarStatusVivoAgente(nombre, persist)
			if err != nil {
				item = agenteStatusVivoItem{Nombre: nombre, Error: err.Error()}
			}
			resp.Agentes = append(resp.Agentes, item)
		}
		if jsonOut {
			return imprimirJSON(resp)
		}
		for _, item := range resp.Agentes {
			fmt.Printf("%-10s ", item.Nombre)
			switch {
			case strings.TrimSpace(item.Error) != "":
				fmt.Printf("ERROR %s\n", strings.TrimSpace(item.Error))
			default:
				fmt.Printf("%s · semanal %s · 5h %s · fuente %s\n",
					valorTexto(item.CuentaEmail, "sin cuenta"),
					fmtPercent(item.WeeklyLeft),
					fmtPercent(item.FiveHourLeft),
					valorTexto(item.Fuente, "desconocida"),
				)
			}
		}
		return nil
	},
}

func legacyLiveStatusEnabled() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv("ORQUESTA_ALLOW_LEGACY_LIVE_STATUS"))) {
	case "1", "true", "yes", "si", "on":
		return true
	default:
		return false
	}
}

func listarAgentesStatusVivo(filtro string) ([]string, error) {
	if nombre := strings.TrimSpace(filtro); nombre != "" {
		return []string{nombre}, nil
	}
	agentes, err := db.ListarAgentes()
	if err != nil {
		return nil, err
	}
	nombres := make([]string, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil || strings.TrimSpace(agente.Nombre) == "" {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if strings.HasPrefix(strings.ToLower(nombre), "codex") {
			nombres = append(nombres, nombre)
		}
	}
	sort.Slice(nombres, func(i, j int) bool {
		return naturalLessCodex(nombres[i], nombres[j])
	})
	return nombres, nil
}

func naturalLessCodex(a, b string) bool {
	ai := codexSuffix(a)
	bi := codexSuffix(b)
	if ai >= 0 && bi >= 0 && ai != bi {
		return ai < bi
	}
	return strings.ToLower(a) < strings.ToLower(b)
}

func codexSuffix(nombre string) int {
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(nombre), "Codex%d", &n); err == nil {
		return n
	}
	return -1
}

func consultarStatusVivoAgente(nombre string, persist bool) (agenteStatusVivoItem, error) {
	item := agenteStatusVivoItem{Nombre: strings.TrimSpace(nombre)}
	if !legacyLiveStatusEnabled() {
		return item, fmt.Errorf("status-vivo legacy deshabilitado")
	}
	handles, err := db.ListarRuntimeHandles(ptrStringTrimmed(item.Nombre))
	if err != nil {
		return item, err
	}
	if len(handles) == 0 {
		return item, fmt.Errorf("sin runtime handles")
	}
	ordenarHandlesParaPresupuestoVivo(handles)
	var errs []string
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if syncedHandle, _, _, err := db.SincronizarRuntimeHandleSupervisado(handle, nil, "status_live_cli"); err == nil && syncedHandle != nil {
			handle = syncedHandle
		}
		estado := strings.TrimSpace(handle.Estado)
		if !strings.EqualFold(estado, "activo") && !strings.EqualFold(estado, "pausado") {
			errs = append(errs, fmt.Sprintf("handle %d %s", handle.ID, estado))
			continue
		}
		obj := controlruntime.ObjetivoProceso{
			PID:          int64PtrFromHandleRef(handle.HandleKind, handle.HandleRef),
			HandleKind:   strings.TrimSpace(handle.HandleKind),
			HandleRef:    strings.TrimSpace(handle.HandleRef),
			MetadataJSON: strings.TrimSpace(handle.MetadataJSON),
		}
		restaurarPausa := false
		if strings.EqualFold(estado, "pausado") {
			if aplicado, _, err := controlruntime.ContinuarProceso(obj); err == nil && aplicado {
				restaurarPausa = true
				time.Sleep(1200 * time.Millisecond)
			}
		}
		result, err := controlruntime.RunSlashCommandLive(obj, "/status", controlruntime.SlashCommandOptions{})
		if restaurarPausa {
			_, _, _ = controlruntime.PausarProceso(obj)
		}
		if err != nil {
			if errors.Is(err, controlruntime.ErrSlashCommandBrokenPipe) {
				_ = db.MarcarRuntimeHandleCanalRoto(handle, nil, err.Error())
			}
			errs = append(errs, fmt.Sprintf("handle %d slash: %v", handle.ID, err))
			continue
		}
		if result == nil || strings.TrimSpace(result.RawOutput) == "" {
			errs = append(errs, fmt.Sprintf("handle %d sin salida", handle.ID))
			continue
		}
		status, err := controlruntime.ParseCodexStatusLive(result.RawOutput, result.FinishedAt)
		if err != nil {
			errs = append(errs, fmt.Sprintf("handle %d parse: %v", handle.ID, err))
			continue
		}
		if status == nil {
			errs = append(errs, fmt.Sprintf("handle %d status nil", handle.ID))
			continue
		}
		item.HandleID = handle.ID
		item.HandleKind = strings.TrimSpace(handle.HandleKind)
		item.HandleRef = strings.TrimSpace(handle.HandleRef)
		item.HandleEstado = strings.TrimSpace(handle.Estado)
		item.Driver = strings.TrimSpace(stringValueFromJSON(handle.MetadataJSON, "driver"))
		item.Fuente = "codex_status_live"
		item.CuentaEmail = strings.TrimSpace(status.AccountEmail)
		item.PlanType = strings.TrimSpace(status.PlanType)
		item.SessionID = strings.TrimSpace(status.SessionID)
		item.FiveHourLeft = status.FiveHourLeft
		item.FiveHourReset = status.FiveHourReset
		item.WeeklyLeft = status.WeeklyLeft
		item.WeeklyReset = status.WeeklyReset
		item.ObservedAt = &result.FinishedAt
		item.RawOutput = strings.TrimSpace(status.RawOutput)
		if persist {
			if observed, err := controlruntime.ObserveCodexStatusLive(obj); err == nil && observed != nil {
				_ = persistirPresupuestoSesionObservado(handle, observed)
			}
		}
		return item, nil
	}
	if len(errs) == 0 {
		return item, fmt.Errorf("sin handle PTY operativo")
	}
	return item, errors.New(strings.Join(errs, " | "))
}

func ptrStringTrimmed(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func fmtPercent(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", *v)
}

func valorTexto(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func resumenCuentaObservadaAgente(agente apiAgenteCuentaItem) string {
	if texto := resumenCuentaCanonica(agente.CuentaID, agente.CuentaEmail, agente.CuentaUsuario); texto != "" {
		return texto
	}
	return "—"
}

func resumenRankingCuenta(cuenta apiCuentaPresupuestoItem) string {
	base := resumenCuentaObservadaAgente(apiAgenteCuentaItem{
		CuentaID:      cuenta.CuentaID,
		CuentaEmail:   cuenta.CuentaEmail,
		CuentaUsuario: cuenta.CuentaUsuario,
	})
	detalle := ""
	switch cuenta.Criterio {
	case "remaining_tokens":
		if cuenta.RemainingTokens != nil {
			detalle = fmt.Sprintf("tokens %d", *cuenta.RemainingTokens)
		}
	case "remaining_credits":
		if cuenta.RemainingCredits != nil {
			detalle = fmt.Sprintf("credits %.2f", *cuenta.RemainingCredits)
		}
	case "remaining_messages":
		if cuenta.RemainingMessages != nil {
			detalle = fmt.Sprintf("mensajes %d", *cuenta.RemainingMessages)
		}
	case "remaining_seconds":
		if cuenta.RemainingSeconds != nil {
			detalle = fmt.Sprintf("segundos %d", *cuenta.RemainingSeconds)
		}
	case "cuota_pct":
		if cuenta.CuotaRestantePct != nil {
			detalle = fmt.Sprintf("%s %d%%", etiquetaCuotaVisibleCuenta(cuenta.PresupuestoStale, cuenta.PresupuestoFuente), *cuenta.CuotaRestantePct)
		}
	case "observed_usage":
		if cuenta.ObservedUsageTokens != nil {
			detalle = fmt.Sprintf("uso observado %d tok", *cuenta.ObservedUsageTokens)
		}
		if cuenta.ObservedUsageCostUSD != nil {
			if detalle != "" {
				detalle += " · "
			}
			detalle += fmt.Sprintf("coste est. $%.4f", *cuenta.ObservedUsageCostUSD)
		}
	default:
		if cuenta.ObservedUsageTokens != nil {
			detalle = fmt.Sprintf("uso %d tok", *cuenta.ObservedUsageTokens)
		}
		if cuenta.ObservedUsageCostUSD != nil {
			if detalle != "" {
				detalle += " · "
			}
			detalle += fmt.Sprintf("coste est. $%.4f", *cuenta.ObservedUsageCostUSD)
		}
	}
	if cuenta.PresupuestoVentana != "" {
		if detalle != "" {
			detalle += " · "
		}
		detalle += "ventana " + cuenta.PresupuestoVentana
	}
	if cuenta.PresupuestoStale {
		if detalle != "" {
			detalle += " · "
		}
		detalle += "stale"
		if cuenta.PresupuestoCheckedAt != nil && !cuenta.PresupuestoCheckedAt.IsZero() {
			if age := edadPresupuestoObservado(cuenta.PresupuestoCheckedAt); age != "" {
				detalle += " (" + age + ")"
			}
		}
	} else if cuenta.ObservedUsageAt != nil && !cuenta.ObservedUsageAt.IsZero() {
		if detalle != "" {
			detalle += " · "
		}
		if age := edadPresupuestoObservado(cuenta.ObservedUsageAt); age != "" {
			detalle += "uso observado " + age
		}
	}
	if cuenta.ObservedUsageMessages != nil || cuenta.ObservedUsageTurns != nil {
		if detalle != "" {
			detalle += " · "
		}
		partes := make([]string, 0, 2)
		if cuenta.ObservedUsageMessages != nil {
			partes = append(partes, fmt.Sprintf("%d msg", *cuenta.ObservedUsageMessages))
		}
		if cuenta.ObservedUsageTurns != nil {
			partes = append(partes, fmt.Sprintf("%d turns", *cuenta.ObservedUsageTurns))
		}
		detalle += strings.Join(partes, " · ")
	}
	if path := strings.TrimSpace(cuenta.ObservedSessionPath); path != "" {
		if detalle != "" {
			detalle += " · "
		}
		detalle += "sesion " + filepath.Base(path)
	}
	if len(cuenta.Agentes) > 0 {
		if detalle != "" {
			detalle += " · "
		}
		detalle += "agentes " + strings.Join(cuenta.Agentes, ",")
	}
	if detalle == "" {
		return base
	}
	if base == "—" {
		return detalle
	}
	return base + " · " + detalle
}

func init() {
	for _, sub := range []*cobra.Command{agentePresupuestoCmd, agenteCuentasCmd, agenteRankingCuentasCmd, agenteStatusVivoCmd} {
		sub.Flags().Bool("activos", false, "Mostrar solo agentes activos")
		sub.Flags().Bool("json", false, "Emitir JSON crudo")
		sub.Flags().Bool("cached", false, "No refrescar la telemetría viva antes de listar")
		if sub == agentePresupuestoCmd {
			sub.Flags().Bool("refresh", false, "Refrescar telemetría observada antes de listar")
			sub.Flags().String("agente", "", "Refrescar solo este agente")
		}
		if sub == agenteStatusVivoCmd {
			sub.Flags().String("agente", "", "Consultar solo este agente")
			sub.Flags().Bool("persist", true, "Persistir el /status observado en la telemetría canónica")
		}
		agenteCmd.AddCommand(sub)
	}
	agenteObservarCuentaCmd.Flags().String("email", "", "Correo observado para el agente")
	agenteObservarCuentaCmd.Flags().String("usuario", "", "Usuario observado para el agente")
	agenteObservarCuentaCmd.Flags().String("fuente", "manual_observed_identity", "Fuente de la observación")
	agenteObservarCuentaCmd.Flags().String("observed-at", "", "Timestamp RFC3339 opcional de la observación")
	agenteCmd.AddCommand(agenteObservarCuentaCmd)
}
