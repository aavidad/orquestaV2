/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/internal/runtimeobs"
)

var agenteCmd = &cobra.Command{
	Use:   "agente",
	Short: "Control activo de agentes vivos",
}

var agenteHandleCmd = &cobra.Command{
	Use:   "handle",
	Short: "Gestión de handles vivos de runtime",
}

var agenteHandleRegistrarCmd = &cobra.Command{
	Use:   "registrar <agente>",
	Short: "Registra o reemplaza el handle vivo de un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sesionID, err := int64FlagOpt(cmd, "sesion")
		if err != nil {
			return err
		}
		transporte, _ := cmd.Flags().GetString("transporte")
		handleKind, _ := cmd.Flags().GetString("kind")
		handleRef, _ := cmd.Flags().GetString("ref")
		estado, _ := cmd.Flags().GetString("estado")
		metadata, _ := cmd.Flags().GetString("metadata")

		id, err := db.RegistrarRuntimeHandle(&db.RuntimeHandle{
			Agente:       strings.TrimSpace(args[0]),
			SesionID:     sesionID,
			Transporte:   strings.TrimSpace(transporte),
			HandleKind:   strings.TrimSpace(handleKind),
			HandleRef:    strings.TrimSpace(handleRef),
			Estado:       strings.TrimSpace(estado),
			MetadataJSON: strings.TrimSpace(metadata),
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Runtime handle registrado para %s (id: %d)\n", args[0], id)
		return nil
	},
}

var agenteHandleListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista los handles vivos registrados",
	RunE: func(cmd *cobra.Command, args []string) error {
		handles, err := db.ListarRuntimeHandles()
		if err != nil {
			return err
		}
		if len(handles) == 0 {
			fmt.Println("No hay runtime handles registrados.")
			return nil
		}
		fmt.Printf("%-5s %-12s %-10s %-12s %-20s %s\n", "ID", "AGENTE", "ESTADO", "TRANSPORTE", "KIND", "REF")
		for _, h := range handles {
			fmt.Printf("%-5d %-12s %-10s %-12s %-20s %s\n", h.ID, h.Agente, h.Estado, h.Transporte, h.HandleKind, h.HandleRef)
		}
		return nil
	},
}

var agenteOrdenCmd = &cobra.Command{
	Use:   "orden",
	Short: "Gestión de órdenes hacia agentes vivos",
}

var agenteTelemetriaCmd = &cobra.Command{
	Use:   "telemetria",
	Short: "Telemetría pasiva y observabilidad de runtimes",
}

var agenteTelemetriaRegistrarCmd = &cobra.Command{
	Use:   "registrar <agente>",
	Short: "Registra una muestra de telemetría pasiva normalizada por proveedor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sesionID, err := int64FlagOpt(cmd, "sesion")
		if err != nil {
			return err
		}
		proveedor, _ := cmd.Flags().GetString("proveedor")
		payload, _ := cmd.Flags().GetString("payload")
		runtimeRef, _ := cmd.Flags().GetString("runtime-ref")
		sessionRef, _ := cmd.Flags().GetString("session-ref")

		proveedor = strings.TrimSpace(proveedor)
		payload = strings.TrimSpace(payload)
		if proveedor == "" {
			return fmt.Errorf("proveedor obligatorio")
		}
		if payload == "" {
			return fmt.Errorf("payload JSON obligatorio")
		}
		if !json.Valid([]byte(payload)) {
			return fmt.Errorf("payload JSON inválido")
		}

		sample, err := runtimeobs.Normalize(proveedor, json.RawMessage(payload))
		if err != nil {
			return err
		}
		if sample.RuntimeRef == "" {
			sample.RuntimeRef = strings.TrimSpace(runtimeRef)
		}
		if sample.SessionRef == "" {
			sample.SessionRef = strings.TrimSpace(sessionRef)
		}
		if sample.RuntimeRef == "" {
			if handle, handleErr := db.RuntimeHandleActivo(strings.TrimSpace(args[0])); handleErr == nil && handle != nil {
				sample.RuntimeRef = handle.HandleRef
			}
		}

		runtimeID, err := registrarTelemetriaPasiva(strings.TrimSpace(args[0]), sesionID, proveedor, payload, sample)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Telemetría pasiva registrada para %s (runtime: %d)\n", args[0], runtimeID)
		fmt.Printf("  proveedor: %s  kind: %s  state: %s\n", sample.Provider, sample.Kind, sample.State)
		if sample.RuntimeRef != "" {
			fmt.Printf("  runtime_ref: %s\n", sample.RuntimeRef)
		}
		return nil
	},
}

var agenteOrdenListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista órdenes de runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		estado, _ := cmd.Flags().GetString("estado")
		orders, err := db.ListarRuntimeOrders(strings.TrimSpace(agente), strings.TrimSpace(estado))
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			fmt.Println("No hay runtime orders con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-12s %-14s %-12s %s\n", "ID", "AGENTE", "TIPO", "ESTADO", "CREADA")
		for _, o := range orders {
			fmt.Printf("%-5d %-12s %-14s %-12s %s\n", o.ID, o.Agente, o.Tipo, o.Estado, o.CreatedAt.Format("2006-01-02 15:04"))
		}
		return nil
	},
}

var agenteTelemetriaListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista runtimes observados con su última actividad",
	RunE: func(cmd *cobra.Command, args []string) error {
		filtroAgente, _ := cmd.Flags().GetString("agente")
		filtroAgente = strings.TrimSpace(filtroAgente)

		runtimes, err := db.ListarRuntimesConUltimaActividad()
		if err != nil {
			return err
		}
		if len(runtimes) == 0 {
			fmt.Println("No hay runtimes observados.")
			return nil
		}

		fmt.Printf("%-5s %-12s %-14s %-14s %-8s %s\n", "ID", "AGENTE", "PROVIDER", "STATE", "PID", "ULTIMA")
		for _, runtime := range runtimes {
			if filtroAgente != "" && runtime.Agente != filtroAgente {
				continue
			}
			pid := "—"
			if runtime.PID != nil {
				pid = strconv.FormatInt(*runtime.PID, 10)
			}
			ultima := "—"
			if runtime.UltimaActividadAt != nil {
				ultima = runtime.UltimaActividadAt.Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%-5d %-12s %-14s %-14s %-8s %s\n",
				runtime.ID, runtime.Agente, runtime.Provider, runtime.LogicalState, pid, ultima)
		}
		return nil
	},
}

var agenteOrdenCrearCmd = &cobra.Command{
	Use:   "crear <agente>",
	Short: "Crea una orden de runtime para un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sesionID, err := int64FlagOpt(cmd, "sesion")
		if err != nil {
			return err
		}
		tipo, _ := cmd.Flags().GetString("tipo")
		payload, _ := cmd.Flags().GetString("payload")
		estado, _ := cmd.Flags().GetString("estado")

		agente := strings.TrimSpace(args[0])
		tipo = strings.TrimSpace(tipo)
		payload = strings.TrimSpace(payload)
		estado = strings.TrimSpace(estado)
		switch tipo {
		case "enviar_instruccion", "pausar", "continuar", "handoff":
		default:
			return fmt.Errorf("tipo inválido: %s", tipo)
		}
		if payload == "" {
			payload = "{}"
		}
		if !json.Valid([]byte(payload)) {
			return fmt.Errorf("payload JSON inválido")
		}

		id, err := db.CrearRuntimeOrder(&db.RuntimeOrder{
			Agente:      agente,
			SesionID:    sesionID,
			Tipo:        tipo,
			PayloadJSON: payload,
			Estado:      estado,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Runtime order creada para %s (id: %d)\n", agente, id)
		return nil
	},
}

var agenteOrdenActualizarCmd = &cobra.Command{
	Use:   "actualizar <id> <estado>",
	Short: "Actualiza el estado de una runtime order",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		if err := db.ActualizarRuntimeOrderEstado(id, strings.TrimSpace(args[1])); err != nil {
			return err
		}
		fmt.Printf("✓ Runtime order #%d actualizada a %s\n", id, args[1])
		return nil
	},
}

var agenteHandoffCmd = &cobra.Command{
	Use:   "handoff <agente-origen> <agente-destino>",
	Short: "Crea un handoff real y reasigna la tarea al agente destino",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := int64FlagOpt(cmd, "tarea")
		if err != nil {
			return err
		}
		resumen, _ := cmd.Flags().GetString("resumen")
		externalSessionID, _ := cmd.Flags().GetString("external-session-id")
		motivo, _ := cmd.Flags().GetString("motivo")

		orderID, err := db.CrearHandoffAgenteVivo(
			strings.TrimSpace(args[0]),
			strings.TrimSpace(args[1]),
			tareaID,
			strings.TrimSpace(motivo),
			strings.TrimSpace(resumen),
			strings.TrimSpace(externalSessionID),
		)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Handoff creado %s → %s (runtime_order: %d)\n", args[0], args[1], orderID)
		if tareaID != nil {
			fmt.Printf("  Tarea #%d reasignada al agente destino\n", *tareaID)
		}
		return nil
	},
}

var agenteReasignarVivoCmd = &cobra.Command{
	Use:   "reasignar-vivo <tarea-id> <agente-origen> <agente-destino>",
	Short: "Atajo de handoff con reasignación explícita de tarea viva",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("tarea-id inválido")
		}
		resumen, _ := cmd.Flags().GetString("resumen")
		externalSessionID, _ := cmd.Flags().GetString("external-session-id")
		motivo, _ := cmd.Flags().GetString("motivo")
		orderID, err := db.CrearHandoffAgenteVivo(
			strings.TrimSpace(args[1]),
			strings.TrimSpace(args[2]),
			&tareaID,
			strings.TrimSpace(motivo),
			strings.TrimSpace(resumen),
			strings.TrimSpace(externalSessionID),
		)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Reasignación viva creada sobre tarea #%d (runtime_order: %d)\n", tareaID, orderID)
		return nil
	},
}

var agenteVerCmd = &cobra.Command{
	Use:   "ver <agente>",
	Short: "Muestra el handle activo, sesión y presupuesto del agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := strings.TrimSpace(args[0])
		handle, handleErr := db.RuntimeHandleActivo(agente)
		sesion, sesionErr := db.SesionActivaDeAgente(agente)
		p, _, presupuestoErr := db.UltimoPresupuestoAgente(agente)
		runtime, runtimeErr := db.RuntimePrincipalAgente(agente)

		fmt.Printf("Agente: %s\n", agente)
		if sesionErr == nil && sesion != nil {
			fmt.Printf("  Sesión activa: #%d\n", sesion.ID)
		} else {
			fmt.Printf("  Sesión activa: —\n")
		}
		if handleErr == nil && handle != nil {
			fmt.Printf("  Handle activo: %s [%s/%s]\n", handle.HandleRef, handle.Transporte, handle.HandleKind)
		} else {
			fmt.Printf("  Handle activo: —\n")
		}
		if presupuestoErr == nil && p != nil {
			ev, err := db.EvaluarPresupuestoSesion(p)
			if err != nil {
				return err
			}
			fmt.Printf("  Presupuesto:   %s", ev.Estado)
			if ev.Motivo != "" {
				fmt.Printf(" (%s)", ev.Motivo)
			}
			fmt.Println()
		} else {
			fmt.Printf("  Presupuesto:   —\n")
		}
		if runtimeErr == nil && runtime != nil {
			fmt.Printf("  Runtime:       #%d %s [%s]\n", runtime.ID, runtime.Provider, runtime.LogicalState)
			if runtime.PID != nil {
				fmt.Printf("  PID:           %d\n", *runtime.PID)
			}
			if runtime.Model != "" {
				fmt.Printf("  Modelo:        %s\n", runtime.Model)
			}
			if runtime.CWD != "" {
				fmt.Printf("  CWD:           %s\n", runtime.CWD)
			}
			if sample, err := db.UltimaMuestraRuntime(runtime.ID); err == nil && sample != nil {
				fmt.Printf("  Telemetría:    cpu=%.2f rss=%s threads=%d children=%d (%s)\n",
					sample.CPUPct, humanBytes(sample.RSSBytes), sample.ThreadCount, sample.ChildCount,
					sample.CreatedAt.Format("2006-01-02 15:04:05"))
			}
		} else {
			fmt.Printf("  Runtime:       —\n")
		}

		if handleErr != nil && handleErr != sql.ErrNoRows {
			return handleErr
		}
		if sesionErr != nil && sesionErr != sql.ErrNoRows {
			return sesionErr
		}
		if runtimeErr != nil && runtimeErr != sql.ErrNoRows {
			return runtimeErr
		}
		return nil
	},
}

func init() {
	agenteHandleRegistrarCmd.Flags().Int64("sesion", 0, "Sesión activa vinculada al handle")
	agenteHandleRegistrarCmd.Flags().String("transporte", "", "Transporte: pty, process, mcp_session, remote_api")
	agenteHandleRegistrarCmd.Flags().String("kind", "", "Tipo de handle: pty, process, mcp_session, remote_api")
	agenteHandleRegistrarCmd.Flags().String("ref", "", "Referencia concreta del handle")
	agenteHandleRegistrarCmd.Flags().String("estado", "activo", "Estado del handle")
	agenteHandleRegistrarCmd.Flags().String("metadata", "{}", "Metadata JSON")

	agenteOrdenListarCmd.Flags().String("agente", "", "Filtrar por agente")
	agenteOrdenListarCmd.Flags().String("estado", "", "Filtrar por estado")
	agenteOrdenCrearCmd.Flags().Int64("sesion", 0, "Sesión activa vinculada a la orden")
	agenteOrdenCrearCmd.Flags().String("tipo", "", "Tipo de orden")
	agenteOrdenCrearCmd.Flags().String("payload", "{}", "Payload JSON de la orden")
	agenteOrdenCrearCmd.Flags().String("estado", "pendiente", "Estado inicial de la orden")
	_ = agenteOrdenCrearCmd.MarkFlagRequired("tipo")

	agenteTelemetriaRegistrarCmd.Flags().Int64("sesion", 0, "Sesión activa vinculada a la muestra")
	agenteTelemetriaRegistrarCmd.Flags().String("proveedor", "", "Proveedor/adaptador pasivo: generic_process, codex_cli")
	agenteTelemetriaRegistrarCmd.Flags().String("payload", "", "Payload JSON crudo observado")
	agenteTelemetriaRegistrarCmd.Flags().String("runtime-ref", "", "Runtime ref observado o inferido")
	agenteTelemetriaRegistrarCmd.Flags().String("session-ref", "", "Session ref observada")
	_ = agenteTelemetriaRegistrarCmd.MarkFlagRequired("proveedor")
	_ = agenteTelemetriaRegistrarCmd.MarkFlagRequired("payload")
	agenteTelemetriaListarCmd.Flags().String("agente", "", "Filtrar por agente")

	agenteHandoffCmd.Flags().Int64("tarea", 0, "Tarea a reasignar al relevo")
	agenteHandoffCmd.Flags().String("motivo", "", "Motivo del handoff")
	agenteHandoffCmd.Flags().String("resumen", "", "Resumen de continuidad para el relevo")
	agenteHandoffCmd.Flags().String("external-session-id", "", "External session id conocido")

	agenteReasignarVivoCmd.Flags().String("motivo", "", "Motivo del relevo")
	agenteReasignarVivoCmd.Flags().String("resumen", "", "Resumen de continuidad")
	agenteReasignarVivoCmd.Flags().String("external-session-id", "", "External session id conocido")

	agenteHandleCmd.AddCommand(agenteHandleRegistrarCmd, agenteHandleListarCmd)
	agenteOrdenCmd.AddCommand(agenteOrdenListarCmd, agenteOrdenCrearCmd, agenteOrdenActualizarCmd)
	agenteTelemetriaCmd.AddCommand(agenteTelemetriaRegistrarCmd, agenteTelemetriaListarCmd)
	agenteCmd.AddCommand(agenteVerCmd, agenteHandleCmd, agenteOrdenCmd, agenteTelemetriaCmd, agenteHandoffCmd, agenteReasignarVivoCmd)
	rootCmd.AddCommand(agenteCmd)
}

func registrarTelemetriaPasiva(agente string, sesionID *int64, proveedor, rawPayload string, sample *runtimeobs.Sample) (int64, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, fmt.Errorf("agente obligatorio")
	}
	proveedor = strings.TrimSpace(proveedor)
	if sample == nil {
		return 0, fmt.Errorf("sample obligatorio")
	}

	runtime, err := runtimeDesdeSample(agente, sesionID, proveedor, sample)
	if err != nil {
		return 0, err
	}
	runtimeID, err := db.RegistrarRuntimeInstance(runtime)
	if err != nil {
		return 0, err
	}

	if strings.TrimSpace(sample.State) != "" {
		eventPayload, err := json.Marshal(sample)
		if err != nil {
			return 0, err
		}
		if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
			RuntimeID:   runtimeID,
			Kind:        "telemetry.sampled",
			Level:       "info",
			Message:     fmt.Sprintf("%s/%s", sample.Provider, sample.State),
			PayloadJSON: string(eventPayload),
		}); err != nil {
			return 0, err
		}
	}

	if _, err := db.RegistrarRuntimeTelemetrySample(&db.RuntimeTelemetrySample{
		RuntimeID:    runtimeID,
		CPUPct:       mapFloat(sample.Details, "cpu_pct"),
		MemBytes:     mapInt64(sample.Details, "mem_bytes"),
		RSSBytes:     mapInt64(sample.Details, "rss_bytes"),
		OpenFDs:      mapInt64(sample.Details, "open_fds"),
		ChildCount:   int(mapInt64(sample.Details, "child_count")),
		ThreadCount:  int(mapInt64(sample.Details, "thread_count")),
		LogicalState: strings.TrimSpace(sample.State),
		Source:       proveedor,
		SampleJSON:   strings.TrimSpace(rawPayload),
	}); err != nil {
		return 0, err
	}
	return runtimeID, nil
}

func runtimeDesdeSample(agente string, sesionID *int64, proveedor string, sample *runtimeobs.Sample) (*db.RuntimeInstance, error) {
	now := sample.ObservedAt.UTC()
	instance := &db.RuntimeInstance{
		Agente:            agente,
		SesionID:          sesionID,
		Provider:          proveedor,
		ExternalSessionID: strings.TrimSpace(sample.SessionRef),
		LogicalState:      strings.TrimSpace(sample.State),
		ProcessState:      strings.TrimSpace(sample.State),
		PID:               sample.PID,
		Model:             strings.TrimSpace(sample.Model),
		CWD:               strings.TrimSpace(sample.CWD),
		LastHeartbeatAt:   &now,
	}
	if branch := mapString(sample.Details, "branch"); branch != "" {
		instance.Branch = branch
	}
	if reasoning := mapString(sample.Details, "reasoning"); reasoning != "" {
		instance.Reasoning = reasoning
	}
	if profile := mapString(sample.Details, "task_profile"); profile != "" {
		instance.TaskProfile = profile
	}
	if parentPID := mapOptionalInt64(sample.Details, "ppid"); parentPID != nil {
		instance.PPID = parentPID
	}
	instance.ChildCount = int(mapInt64(sample.Details, "child_count"))
	instance.ThreadCount = int(mapInt64(sample.Details, "thread_count"))
	if instance.ExternalSessionID == "" && sample.RuntimeRef != "" {
		instance.ExternalSessionID = sample.RuntimeRef
	}
	if instance.LogicalState == "" {
		instance.LogicalState = "observado"
	}
	if instance.ProcessState == "" {
		instance.ProcessState = "observado"
	}
	if sesionID != nil {
		if actual, err := db.RuntimeInstancePorSesionID(*sesionID); err == nil && actual != nil {
			instance.ID = actual.ID
		} else if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}
	return instance, nil
}

func mapFloat(details map[string]any, key string) float64 {
	if details == nil {
		return 0
	}
	v, ok := details[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func mapInt64(details map[string]any, key string) int64 {
	if details == nil {
		return 0
	}
	v, ok := details[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(math.Round(n))
	case float32:
		return int64(math.Round(float64(n)))
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

func mapOptionalInt64(details map[string]any, key string) *int64 {
	if details == nil {
		return nil
	}
	if _, ok := details[key]; !ok {
		return nil
	}
	v := mapInt64(details, key)
	return &v
}

func mapString(details map[string]any, key string) string {
	if details == nil {
		return ""
	}
	v, ok := details[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func humanBytes(v int64) string {
	if v <= 0 {
		return "0B"
	}
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%dB", v)
	}
	div, exp := int64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(v)/float64(div), "KMGTPE"[exp])
}
