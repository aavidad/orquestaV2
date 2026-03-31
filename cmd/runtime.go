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
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
)

func runtimeErrorServerFirst() error {
	return fmt.Errorf("este subcomando de runtime ya se sirve por Orquesta server; arranca el servidor o usa ORQUESTA_FORCE_LOCAL_DB=1 solo para recuperacion")
}

type runtimeRow struct {
	ID           int64
	Nivel        int
	Agente       string
	Proyecto     string
	Provider     string
	Connector    string
	Estado       string
	PID          string
	Hijos        int
	Branch       string
	Modelo       string
	Razonamiento string
	Ultima       string
}

type runtimeHandleRow struct {
	ID         int64
	Agente     string
	Transporte string
	Kind       string
	Ref        string
	Estado     string
	LastSeen   string
}

type runtimeOrderRow struct {
	ID        int64
	Agente    string
	Tipo      string
	Estado    string
	CreatedAt string
}

type runtimeCheckpointRow struct {
	ID        int64
	Agente    string
	Kind      string
	Branch    string
	Source    string
	CreatedAt string
	Resumen   string
}

var runtimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "Inspección read-only de runtimes y agentes hijos",
}

var runtimeHandlesCmd = &cobra.Command{
	Use:   "handles",
	Short: "Lista runtime handles del control plane",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		query := url.Values{}
		if strings.TrimSpace(agente) != "" {
			query.Set("agente", agente)
		}
		if handles, ok, err := cargarRuntimeHandlesDesdeAPI(query); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeHandles(handles)
		} else if runtimeModoRecuperacionLocalExplicito() {
			handles, err := cargarRuntimeHandlesRecuperacionLocal(query)
			if err != nil {
				return err
			}
			return imprimirRuntimeHandles(handles)
		}
		return serverFirstCommandError("runtime handles")
	},
}

var runtimePurgarHandlesCmd = &cobra.Command{
	Use:   "purgar-handles",
	Short: "Purga runtime handles inactivos de pruebas sin borrar transcript ni trazas",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		proyecto, _ := cmd.Flags().GetString("proyecto")
		estados, _ := cmd.Flags().GetStringSlice("estado")
		estados = normalizarSliceFlags(estados)
		actor, _ := cmd.Flags().GetString("actor")
		if strings.TrimSpace(agente) == "" && strings.TrimSpace(proyecto) == "" {
			return fmt.Errorf("debes indicar --agente o --proyecto")
		}
		if resp, ok, err := purgarRuntimeHandlesDesdeAPI(strings.TrimSpace(agente), strings.TrimSpace(proyecto), estados, strings.TrimSpace(actor)); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Purgados %d runtime handles inactivos", resp.Deleted)
			if len(resp.Estados) > 0 {
				fmt.Printf(" [%s]", strings.Join(resp.Estados, ","))
			}
			if len(resp.DeletedIDs) > 0 {
				fmt.Printf(": %v", resp.DeletedIDs)
			}
			fmt.Println()
			return nil
		}
		return serverFirstCommandError("runtime purgar-handles")
	},
}

var runtimePurgarOrdenesCmd = &cobra.Command{
	Use:   "purgar-ordenes",
	Short: "Purga runtime orders terminales de pruebas sin tocar órdenes vivas",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		proyecto, _ := cmd.Flags().GetString("proyecto")
		estados, _ := cmd.Flags().GetStringSlice("estado")
		tipos, _ := cmd.Flags().GetStringSlice("tipo")
		olderThanMinutes, _ := cmd.Flags().GetInt("older-than-minutes")
		actor, _ := cmd.Flags().GetString("actor")
		estados = normalizarSliceFlags(estados)
		tipos = normalizarSliceFlags(tipos)
		if strings.TrimSpace(agente) == "" && strings.TrimSpace(proyecto) == "" {
			return fmt.Errorf("debes indicar --agente o --proyecto")
		}
		if olderThanMinutes < 0 {
			return fmt.Errorf("--older-than-minutes no puede ser negativo")
		}
		if resp, ok, err := purgarRuntimeOrdersDesdeAPI(strings.TrimSpace(agente), strings.TrimSpace(proyecto), estados, tipos, olderThanMinutes, strings.TrimSpace(actor)); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Purgadas %d runtime orders terminales", resp.Deleted)
			if len(resp.Estados) > 0 {
				fmt.Printf(" [%s]", strings.Join(resp.Estados, ","))
			}
			if len(resp.Tipos) > 0 {
				fmt.Printf(" tipos=%s", strings.Join(resp.Tipos, ","))
			}
			if len(resp.DeletedIDs) > 0 {
				fmt.Printf(": %v", resp.DeletedIDs)
			}
			fmt.Println()
			return nil
		}
		return serverFirstCommandError("runtime purgar-ordenes")
	},
}

var runtimeTranscriptCmd = &cobra.Command{
	Use:   "transcript",
	Short: "Lista o busca transcript persistido de runtimes",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		stream, _ := cmd.Flags().GetString("stream")
		classification, _ := cmd.Flags().GetString("classification")
		queryText, _ := cmd.Flags().GetString("q")
		runtimeID, _ := cmd.Flags().GetInt64("runtime-id")
		handleID, _ := cmd.Flags().GetInt64("handle-id")
		signalsPending, _ := cmd.Flags().GetBool("signals-pending")
		limit, _ := cmd.Flags().GetInt("limit")

		query := url.Values{}
		if strings.TrimSpace(agente) != "" {
			query.Set("agente", agente)
		}
		if strings.TrimSpace(proyectoRef) != "" {
			query.Set("proyecto", proyectoRef)
		}
		if strings.TrimSpace(stream) != "" {
			query.Set("stream", stream)
		}
		if strings.TrimSpace(classification) != "" {
			query.Set("classification", classification)
		}
		if strings.TrimSpace(queryText) != "" {
			query.Set("q", queryText)
		}
		if runtimeID < 0 {
			return fmt.Errorf("--runtime-id inválido")
		}
		if runtimeID > 0 {
			query.Set("runtime_id", strconv.FormatInt(runtimeID, 10))
		}
		if handleID < 0 {
			return fmt.Errorf("--handle-id inválido")
		}
		if handleID > 0 {
			query.Set("handle_id", strconv.FormatInt(handleID, 10))
		}
		if signalsPending {
			query.Set("signals_pending", "true")
		}
		if limit > 0 {
			query.Set("limit", strconv.Itoa(limit))
		}

		if items, ok, err := cargarRuntimeTranscriptDesdeAPI(query); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeTranscript(items)
		} else if runtimeModoRecuperacionLocalExplicito() {
			items, err := cargarRuntimeTranscriptRecuperacionLocal(query)
			if err != nil {
				return err
			}
			return imprimirRuntimeTranscript(items)
		}
		return serverFirstCommandError("runtime transcript")
	},
}

func imprimirRuntimeHandles(handles []*db.RuntimeHandle) error {
	if len(handles) == 0 {
		fmt.Println("No hay runtime handles con ese filtro.")
		return nil
	}
	fmt.Printf("%-5s %-12s %-12s %-10s %-18s %-10s %s\n", "ID", "AGENTE", "TRANSPORTE", "KIND", "REF", "ESTADO", "LAST_SEEN")
	for _, h := range handles {
		fmt.Printf("%-5d %-12s %-12s %-10s %-18s %-10s %s\n",
			h.ID, truncar(h.Agente, 12), truncar(h.Transporte, 12), truncar(h.HandleKind, 10),
			truncar(h.HandleRef, 18), truncar(h.Estado, 10), formatoTiempoPtr(h.LastSeenAt))
	}
	return nil
}

func imprimirRuntimeTranscript(items []*db.RuntimeTranscriptEntry) error {
	if len(items) == 0 {
		fmt.Println("No hay transcript con ese filtro.")
		return nil
	}
	fmt.Printf("%-5s %-12s %-8s %-20s %-19s %s\n", "ID", "AGENTE", "STREAM", "SIGNAL", "CREADO", "TEXTO")
	for _, item := range items {
		if item == nil {
			continue
		}
		fmt.Printf("%-5d %-12s %-8s %-20s %-19s %s\n",
			item.ID,
			truncar(item.Agente, 12),
			truncar(item.Stream, 8),
			truncar(item.Classification, 20),
			item.CreatedAt.Format("2006-01-02 15:04:05"),
			truncar(item.Text, 120),
		)
	}
	return nil
}

var runtimeOrdenesCmd = &cobra.Command{
	Use:   "ordenes",
	Short: "Lista órdenes persistidas del control plane",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		estado, _ := cmd.Flags().GetString("estado")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		limit, _ := cmd.Flags().GetInt("limit")
		query := url.Values{}
		if strings.TrimSpace(agente) != "" {
			query.Set("agente", agente)
		}
		if strings.TrimSpace(estado) != "" {
			query.Set("estado", estado)
		}
		if strings.TrimSpace(proyectoRef) != "" {
			query.Set("proyecto", proyectoRef)
		}
		if limit > 0 {
			query.Set("limit", strconv.Itoa(limit))
		}
		if orders, ok, err := cargarRuntimeOrdersDesdeAPI(query); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeOrders(orders)
		} else if runtimeModoRecuperacionLocalExplicito() {
			orders, err := cargarRuntimeOrdersRecuperacionLocal(query)
			if err != nil {
				return err
			}
			return imprimirRuntimeOrders(orders)
		}
		return serverFirstCommandError("runtime ordenes")
	},
}

func imprimirRuntimeOrders(orders []*db.RuntimeOrder) error {
	if len(orders) == 0 {
		fmt.Println("No hay órdenes con ese filtro.")
		return nil
	}
	fmt.Printf("%-5s %-12s %-18s %-12s %-19s %s\n", "ID", "AGENTE", "TIPO", "ESTADO", "DISPONIBLE", "DETALLE")
	for _, o := range orders {
		fmt.Printf("%-5d %-12s %-18s %-12s %-19s %s\n",
			o.ID, truncar(o.Agente, 12), truncar(o.Tipo, 18), truncar(o.Estado, 12),
			formatoTiempoValor(o.AvailableAt), truncar(runtimeOrderDetalleResumen(o), 72))
	}
	return nil
}

func runtimeOrderDetalleResumen(order *db.RuntimeOrder) string {
	if order == nil {
		return ""
	}
	resultado := map[string]any{}
	if strings.TrimSpace(order.ResultadoJSON) != "" {
		_ = json.Unmarshal([]byte(order.ResultadoJSON), &resultado)
	}
	if motivo := strings.TrimSpace(runtimeOrderResultadoString(resultado, "deferred_reason")); motivo != "" {
		if retryAfter := strings.TrimSpace(runtimeOrderResultadoString(resultado, "retry_after")); retryAfter != "" {
			return motivo + " -> " + retryAfter
		}
		return motivo
	}
	if errText := ultimoParrafoNoVacio(order.ErrorText); errText != "" {
		return errText
	}
	if retryAfter := strings.TrimSpace(runtimeOrderResultadoString(resultado, "retry_after")); retryAfter != "" {
		return "retry_after=" + retryAfter
	}
	if available := formatoTiempoValor(order.AvailableAt); available != "—" {
		return "creada " + order.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return order.CreatedAt.Format("2006-01-02 15:04:05")
}

func runtimeOrderResultadoString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if raw, ok := payload[key]; ok {
		switch v := raw.(type) {
		case string:
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func ultimoParrafoNoVacio(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	partes := strings.Split(raw, "\n")
	for i := len(partes) - 1; i >= 0; i-- {
		if texto := strings.TrimSpace(partes[i]); texto != "" {
			return texto
		}
	}
	return ""
}

func formatoTiempoValor(ts time.Time) string {
	if ts.IsZero() {
		return "—"
	}
	return ts.Format("2006-01-02 15:04:05")
}

var runtimeOrdenNuevaCmd = &cobra.Command{
	Use:   "orden-nueva <agente> <tipo>",
	Short: "Encola una orden del control plane",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := strings.TrimSpace(args[0])
		tipo := strings.TrimSpace(args[1])
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		payload, _ := cmd.Flags().GetString("payload")
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		if id, ok, err := crearRuntimeOrderDesdeAPI(agente, tipo, proyectoRef, payload); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Orden runtime #%d encolada para %s (%s)\n", id, agente, tipo)
			return nil
		}
		return serverFirstCommandError("runtime orden-nueva")
	},
}

var runtimeNudgeCmd = &cobra.Command{
	Use:   "nudge <to_agente> <texto>",
	Short: "Encola un nudge para un agente a través del control plane",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fromAgente, _ := cmd.Flags().GetString("from")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		kind, _ := cmd.Flags().GetString("kind")
		if strings.TrimSpace(kind) == "" {
			kind = "nudge"
		}
		id, err := encolarRuntimeMensajeSimple(
			"nudge",
			strings.TrimSpace(args[0]),
			proyectoRef,
			map[string]any{
				"from_agente": strings.TrimSpace(fromAgente),
				"to_agente":   strings.TrimSpace(args[0]),
				"kind":        strings.TrimSpace(kind),
				"texto":       strings.TrimSpace(strings.Join(args[1:], " ")),
			},
		)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Nudge runtime #%d encolado para %s\n", id, strings.TrimSpace(args[0]))
		return nil
	},
}

var runtimeDiscordiaCmd = &cobra.Command{
	Use:   "discordia <supervisor> <contra_agente> <motivo...>",
	Short: "Escala una discordia técnica al supervisor a través del control plane",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		supervisor := strings.TrimSpace(args[0])
		contraAgente := strings.TrimSpace(args[1])
		motivo := strings.TrimSpace(strings.Join(args[2:], " "))
		fromAgente, _ := cmd.Flags().GetString("from")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		id, err := encolarRuntimeMensajeSimple(
			"discordia",
			supervisor,
			proyectoRef,
			map[string]any{
				"from_agente":   strings.TrimSpace(fromAgente),
				"to_agente":     supervisor,
				"kind":          "discordia",
				"contra_agente": contraAgente,
				"motivo":        motivo,
			},
		)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Discordia runtime #%d encolada para %s\n", id, supervisor)
		return nil
	},
}

func encolarRuntimeMensajeSimple(tipo, agenteDestino, proyectoRef string, payload map[string]any) (int64, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	if id, ok, err := crearRuntimeOrderDesdeAPI(agenteDestino, tipo, proyectoRef, string(payloadJSON)); ok {
		return id, err
	}
	return 0, serverFirstCommandError("runtime control")
}

var runtimeCheckpointsCmd = &cobra.Command{
	Use:   "checkpoints",
	Short: "Muestra el último checkpoint de un agente",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		checkpointKind, _ := cmd.Flags().GetString("kind")
		source, _ := cmd.Flags().GetString("source")
		limit, _ := cmd.Flags().GetInt("limit")
		if strings.TrimSpace(agente) == "" {
			return fmt.Errorf("--agente es obligatorio")
		}
		if limit <= 0 {
			limit = 1
		}
		usarHistorialLocal := strings.TrimSpace(checkpointKind) != "" || strings.TrimSpace(source) != "" || limit > 1
		if usarHistorialLocal {
			if checkpoints, ok, err := cargarRuntimeCheckpointsDesdeAPI(agente, proyectoRef, checkpointKind, source, limit); ok {
				if err != nil {
					return err
				}
				return imprimirRuntimeCheckpointLista(checkpoints)
			}
			if runtimeModoRecuperacionLocalExplicito() {
				checkpoints, err := cargarRuntimeCheckpointsRecuperacionLocal(agente, proyectoRef, checkpointKind, source, limit)
				if err != nil {
					return err
				}
				return imprimirRuntimeCheckpointLista(checkpoints)
			}
		} else if cp, ok, err := cargarCheckpointRuntimeDesdeAPI(agente, proyectoRef); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeCheckpoint(cp)
		} else if runtimeModoRecuperacionLocalExplicito() {
			cp, err := cargarCheckpointRuntimeRecuperacionLocal(agente, proyectoRef)
			if err != nil {
				return err
			}
			return imprimirRuntimeCheckpoint(cp)
		}
		return serverFirstCommandError("runtime checkpoints")
	},
}

var runtimeCheckpointNuevoCmd = &cobra.Command{
	Use:   "checkpoint-nuevo <agente>",
	Short: "Crea un checkpoint manual para un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := strings.TrimSpace(args[0])
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		sesionID, _ := cmd.Flags().GetInt64("sesion-id")
		runtimeID, _ := cmd.Flags().GetInt64("runtime-id")
		checkpointKind, _ := cmd.Flags().GetString("kind")
		resumen, _ := cmd.Flags().GetString("resumen")
		branch, _ := cmd.Flags().GetString("branch")
		cwd, _ := cmd.Flags().GetString("cwd")
		payload, _ := cmd.Flags().GetString("payload")
		resumeStrategy, _ := cmd.Flags().GetString("resume-strategy")
		source, _ := cmd.Flags().GetString("source")
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		if strings.TrimSpace(checkpointKind) == "" {
			checkpointKind = "manual"
		}
		if strings.TrimSpace(resumeStrategy) == "" {
			resumeStrategy = "resumen_y_payload"
		}
		if strings.TrimSpace(source) == "" {
			source = "runtime-cli"
		}

		if id, ok, err := crearRuntimeCheckpointDesdeAPI(agente, proyectoRef, sesionID, runtimeID, checkpointKind, resumen, branch, cwd, payload, resumeStrategy, source); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Checkpoint runtime #%d creado para %s (%s)\n", id, agente, checkpointKind)
			return nil
		}
		return serverFirstCommandError("runtime checkpoint-nuevo")
	},
}

var runtimeCheckpointVerCmd = &cobra.Command{
	Use:   "checkpoint-ver <id>",
	Short: "Muestra un checkpoint concreto por id",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		if cp, ok, err := cargarRuntimeCheckpointPorIDDesdeAPI(id); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeCheckpoint(cp)
		} else if runtimeModoRecuperacionLocalExplicito() {
			cp, err := cargarRuntimeCheckpointPorIDRecuperacionLocal(id)
			if err != nil {
				return err
			}
			return imprimirRuntimeCheckpoint(cp)
		}
		return serverFirstCommandError("runtime checkpoint-ver")
	},
}

var runtimeMailboxCmd = &cobra.Command{
	Use:   "mailbox",
	Short: "Lista mensajes del runtime mailbox",
	RunE: func(cmd *cobra.Command, args []string) error {
		toAgente, _ := cmd.Flags().GetString("to")
		fromAgente, _ := cmd.Flags().GetString("from")
		estado, _ := cmd.Flags().GetString("estado")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")

		query := url.Values{}
		if strings.TrimSpace(toAgente) != "" {
			query.Set("to_agente", toAgente)
		}
		if strings.TrimSpace(fromAgente) != "" {
			query.Set("from_agente", fromAgente)
		}
		if strings.TrimSpace(estado) != "" {
			query.Set("estado", estado)
		}
		if strings.TrimSpace(proyectoRef) != "" {
			query.Set("proyecto", proyectoRef)
		}
		if mailbox, ok, err := cargarRuntimeMailboxDesdeAPI(query); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeMailbox(mailbox)
		} else if runtimeModoRecuperacionLocalExplicito() {
			mailbox, err := cargarRuntimeMailboxRecuperacionLocal(query)
			if err != nil {
				return err
			}
			return imprimirRuntimeMailbox(mailbox)
		}
		return serverFirstCommandError("runtime mailbox")
	},
}

var runtimeMailboxEnviarCmd = &cobra.Command{
	Use:   "mailbox-enviar <from_agente> <to_agente> <kind>",
	Short: "Envía un mensaje al runtime mailbox",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		fromAgente := strings.TrimSpace(args[0])
		toAgente := strings.TrimSpace(args[1])
		kind := strings.TrimSpace(args[2])
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		payload, _ := cmd.Flags().GetString("payload")
		runtimeOrderID, _ := cmd.Flags().GetInt64("runtime-order-id")
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		if id, ok, err := crearRuntimeMailboxDesdeAPI(fromAgente, toAgente, proyectoRef, runtimeOrderID, kind, payload); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Mensaje mailbox #%d enviado de %s a %s (%s)\n", id, fromAgente, toAgente, kind)
			return nil
		}
		return serverFirstCommandError("runtime mailbox-enviar")
	},
}

var runtimeMailboxEntregarCmd = &cobra.Command{
	Use:   "mailbox-entregar <id>",
	Short: "Marca un mensaje del mailbox como entregado",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		if ok, err := marcarRuntimeMailboxEntregadoPorAPI(id); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Mensaje mailbox #%d marcado como entregado\n", id)
			return nil
		}
		return serverFirstCommandError("runtime mailbox-entregar")
	},
}

var runtimeMailboxConsumirCmd = &cobra.Command{
	Use:   "mailbox-consumir <id>",
	Short: "Marca un mensaje del mailbox como consumido",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		if ok, err := marcarRuntimeMailboxConsumidoPorAPI(id); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Mensaje mailbox #%d marcado como consumido\n", id)
			return nil
		}
		return serverFirstCommandError("runtime mailbox-consumir")
	},
}

func imprimirRuntimeCheckpoint(cp *db.RuntimeCheckpoint) error {
	if cp == nil {
		fmt.Println("No hay checkpoints para ese agente.")
		return nil
	}
	fmt.Printf("Checkpoint #%d — %s\n", cp.ID, cp.Agente)
	fmt.Printf("  Tipo:      %s\n", cp.CheckpointKind)
	fmt.Printf("  Branch:    %s\n", valorVacio(cp.Branch))
	fmt.Printf("  CWD:       %s\n", valorVacio(cp.CWD))
	fmt.Printf("  Estrategia:%s\n", valorVacio(cp.ResumeStrategy))
	fmt.Printf("  Fuente:    %s\n", valorVacio(cp.Source))
	fmt.Printf("  Creado:    %s\n", cp.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Resumen:   %s\n", valorVacio(cp.Resumen))
	return nil
}

func imprimirRuntimeCheckpointLista(checkpoints []*db.RuntimeCheckpoint) error {
	if len(checkpoints) == 0 {
		fmt.Println("No hay checkpoints para ese filtro.")
		return nil
	}
	fmt.Printf("%-5s %-12s %-16s %-18s %-16s %s\n", "ID", "AGENTE", "KIND", "SOURCE", "BRANCH", "CREADO")
	for _, cp := range checkpoints {
		fmt.Printf("%-5d %-12s %-16s %-18s %-16s %s\n",
			cp.ID, truncar(cp.Agente, 12), truncar(cp.CheckpointKind, 16), truncar(cp.Source, 18),
			truncar(cp.Branch, 16), cp.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func imprimirRuntimeMailbox(mailbox []*db.RuntimeMailboxMessage) error {
	if len(mailbox) == 0 {
		fmt.Println("No hay mensajes runtime mailbox con ese filtro.")
		return nil
	}
	fmt.Printf("%-5s %-12s %-12s %-16s %-12s %s\n", "ID", "FROM", "TO", "KIND", "ESTADO", "CREADO")
	for _, msg := range mailbox {
		fmt.Printf("%-5d %-12s %-12s %-16s %-12s %s\n",
			msg.ID, truncar(msg.FromAgente, 12), truncar(msg.ToAgente, 12), truncar(msg.Kind, 16),
			truncar(msg.Estado, 12), msg.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}

var runtimeListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista runtimes con estado pasivo y jerarquía",
	RunE: func(cmd *cobra.Command, args []string) error {
		query := url.Values{}
		agente, _ := cmd.Flags().GetString("agente")
		proyecto, _ := cmd.Flags().GetString("proyecto")
		activos, _ := cmd.Flags().GetString("activos")
		if strings.TrimSpace(agente) != "" {
			query.Set("agente", agente)
		}
		if strings.TrimSpace(proyecto) != "" {
			query.Set("proyecto", proyecto)
		}
		if strings.TrimSpace(activos) != "" {
			query.Set("activos", activos)
		}

		if tree, ok, err := cargarArbolRuntimesDesdeAPI(query); ok {
			if err != nil {
				return err
			}
			return imprimirArbolRuntimes(aplanarArbolAPI(tree), false)
		} else if runtimeModoRecuperacionLocalExplicito() {
			tree, err := cargarArbolRuntimesRecuperacionLocal(query)
			if err != nil {
				return err
			}
			return imprimirArbolRuntimes(aplanarArbolLocal(tree), false)
		}
		return serverFirstCommandError("runtime listar")
	},
}

var runtimeVerCmd = &cobra.Command{
	Use:   "ver <id>",
	Short: "Muestra el detalle de un runtime y sus muestras recientes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}

		if detail, ok, err := cargarDetalleRuntimeDesdeAPI(id); ok {
			if err != nil {
				return err
			}
			return imprimirDetalleRuntime(detail.Runtime, detail.Samples)
		} else if runtimeModoRecuperacionLocalExplicito() {
			detail, err := cargarDetalleRuntimeRecuperacionLocal(id)
			if err != nil {
				return err
			}
			return imprimirDetalleRuntime(detail.Runtime, detail.Samples)
		}
		return serverFirstCommandError("runtime ver")
	},
}

func cargarArbolRuntimesDesdeAPI(query url.Values) ([]*apiRuntimeTreeNode, bool, error) {
	var resp apiRuntimeTreeResponse
	ok, err := apiGetQuery("/api/runtimes/tree", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Runtimes, true, nil
}

func cargarDetalleRuntimeDesdeAPI(id int64) (*apiRuntimeDetailResponse, bool, error) {
	var resp apiRuntimeDetailResponse
	ok, err := apiGet(fmt.Sprintf("/api/runtimes/%d", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func cargarRuntimeHandlesDesdeAPI(query url.Values) ([]*db.RuntimeHandle, bool, error) {
	var resp apiRuntimeHandlesResponse
	ok, err := apiGetQuery("/api/runtime-handles", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Handles, true, nil
}

func purgarRuntimeHandlesDesdeAPI(agente, proyecto string, estados []string, actor string) (*apiRuntimeHandlesPurgeResponse, bool, error) {
	var resp apiRuntimeHandlesPurgeResponse
	ok, err := apiPost("/api/runtime-handles/purgar", map[string]any{
		"agente":   strings.TrimSpace(agente),
		"proyecto": strings.TrimSpace(proyecto),
		"estados":  estados,
		"actor":    strings.TrimSpace(actor),
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func purgarRuntimeOrdersDesdeAPI(agente, proyecto string, estados, tipos []string, olderThanMinutes int, actor string) (*apiRuntimeOrdersPurgeResponse, bool, error) {
	var resp apiRuntimeOrdersPurgeResponse
	ok, err := apiPost("/api/runtime-orders/purgar", map[string]any{
		"agente":             strings.TrimSpace(agente),
		"proyecto":           strings.TrimSpace(proyecto),
		"estados":            estados,
		"tipos":              tipos,
		"older_than_minutes": olderThanMinutes,
		"actor":              strings.TrimSpace(actor),
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func normalizarSliceFlags(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.TrimPrefix(value, "[")
		value = strings.TrimSuffix(value, "]")
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func cargarRuntimeTranscriptDesdeAPI(query url.Values) ([]*db.RuntimeTranscriptEntry, bool, error) {
	var resp apiRuntimeTranscriptResponse
	ok, err := apiGetQuery("/api/runtime-transcript", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Transcript, true, nil
}

func cargarRuntimeOrdersDesdeAPI(query url.Values) ([]*db.RuntimeOrder, bool, error) {
	var resp apiRuntimeOrdersResponse
	ok, err := apiGetQuery("/api/runtime-orders", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Orders, true, nil
}

func crearRuntimeOrderDesdeAPI(agente, tipo, proyecto, payload string) (int64, bool, error) {
	var resp apiRuntimeOrderCreateResponse
	ok, err := apiPost("/api/runtime-orders", map[string]any{
		"agente":   strings.TrimSpace(agente),
		"tipo":     strings.TrimSpace(tipo),
		"proyecto": strings.TrimSpace(proyecto),
		"payload":  payload,
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func cargarCheckpointRuntimeDesdeAPI(agente, proyecto string) (*db.RuntimeCheckpoint, bool, error) {
	query := url.Values{}
	query.Set("agente", strings.TrimSpace(agente))
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	var resp apiRuntimeCheckpointResponse
	ok, err := apiGetQuery("/api/runtime-checkpoints/latest", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Checkpoint, true, nil
}

func cargarRuntimeCheckpointsDesdeAPI(agente, proyecto, checkpointKind, source string, limit int) ([]*db.RuntimeCheckpoint, bool, error) {
	query := url.Values{}
	query.Set("agente", strings.TrimSpace(agente))
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	if strings.TrimSpace(checkpointKind) != "" {
		query.Set("kind", strings.TrimSpace(checkpointKind))
	}
	if strings.TrimSpace(source) != "" {
		query.Set("source", strings.TrimSpace(source))
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	var resp apiRuntimeCheckpointsResponse
	ok, err := apiGetQuery("/api/runtime-checkpoints", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Checkpoints, true, nil
}

func cargarRuntimeCheckpointPorIDDesdeAPI(id int64) (*db.RuntimeCheckpoint, bool, error) {
	var resp apiRuntimeCheckpointResponse
	ok, err := apiGet(fmt.Sprintf("/api/runtime-checkpoints/%d", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Checkpoint, true, nil
}

func cargarRuntimeHandlesRecuperacionLocal(query url.Values) ([]*db.RuntimeHandle, error) {
	var agente *string
	if value := strings.TrimSpace(query.Get("agente")); value != "" {
		agente = &value
	}
	return db.ListarRuntimeHandles(agente)
}

func cargarRuntimeTranscriptRecuperacionLocal(query url.Values) ([]*db.RuntimeTranscriptEntry, error) {
	proyectoID, err := runtimeProyectoIDDesdeQuery(query)
	if err != nil {
		return nil, err
	}
	runtimeID, err := runtimeOptionalInt64Query(query, "runtime_id")
	if err != nil {
		return nil, err
	}
	handleID, err := runtimeOptionalInt64Query(query, "handle_id")
	if err != nil {
		return nil, err
	}
	limit, err := runtimeOptionalIntQuery(query, "limit")
	if err != nil {
		return nil, err
	}
	filter := db.FiltroRuntimeTranscript{
		ProyectoID:      proyectoID,
		RuntimeID:       runtimeID,
		HandleID:        handleID,
		SoloSenalesPend: strings.EqualFold(strings.TrimSpace(query.Get("signals_pending")), "true"),
		Limit:           limit,
	}
	if value := strings.TrimSpace(query.Get("agente")); value != "" {
		filter.Agente = &value
	}
	if value := strings.TrimSpace(query.Get("stream")); value != "" {
		filter.Stream = &value
	}
	if value := strings.TrimSpace(query.Get("classification")); value != "" {
		filter.Classification = &value
	}
	if value := strings.TrimSpace(query.Get("q")); value != "" {
		filter.Query = &value
	}
	return db.ListarRuntimeTranscript(filter)
}

func cargarRuntimeOrdersRecuperacionLocal(query url.Values) ([]*db.RuntimeOrder, error) {
	proyectoID, err := runtimeProyectoIDDesdeQuery(query)
	if err != nil {
		return nil, err
	}
	filter := db.FiltroRuntimeOrders{ProyectoID: proyectoID}
	if value := strings.TrimSpace(query.Get("agente")); value != "" {
		filter.Agente = &value
	}
	if value := strings.TrimSpace(query.Get("estado")); value != "" {
		filter.Estado = &value
	}
	limit, err := runtimeOptionalIntQuery(query, "limit")
	if err != nil {
		return nil, err
	}
	filter.Limit = limit
	return db.ListarRuntimeOrders(filter)
}

func cargarCheckpointRuntimeRecuperacionLocal(agente, proyecto string) (*db.RuntimeCheckpoint, error) {
	proyectoID, err := runtimeProyectoIDRef(proyecto)
	if err != nil {
		return nil, err
	}
	return db.UltimoRuntimeCheckpoint(strings.TrimSpace(agente), proyectoID)
}

func cargarRuntimeCheckpointsRecuperacionLocal(agente, proyecto, checkpointKind, source string, limit int) ([]*db.RuntimeCheckpoint, error) {
	proyectoID, err := runtimeProyectoIDRef(proyecto)
	if err != nil {
		return nil, err
	}
	filter := db.FiltroRuntimeCheckpoints{
		ProyectoID: proyectoID,
		Limit:      limit,
	}
	agente = strings.TrimSpace(agente)
	if agente != "" {
		filter.Agente = &agente
	}
	checkpointKind = strings.TrimSpace(checkpointKind)
	if checkpointKind != "" {
		filter.CheckpointKind = &checkpointKind
	}
	source = strings.TrimSpace(source)
	if source != "" {
		filter.Source = &source
	}
	return db.ListarRuntimeCheckpoints(filter)
}

func cargarRuntimeCheckpointPorIDRecuperacionLocal(id int64) (*db.RuntimeCheckpoint, error) {
	return db.GetRuntimeCheckpoint(id)
}

func cargarRuntimeMailboxRecuperacionLocal(query url.Values) ([]*db.RuntimeMailboxMessage, error) {
	proyectoID, err := runtimeProyectoIDDesdeQuery(query)
	if err != nil {
		return nil, err
	}
	filter := db.FiltroRuntimeMailbox{ProyectoID: proyectoID}
	if value := strings.TrimSpace(query.Get("to_agente")); value != "" {
		filter.ToAgente = &value
	}
	if value := strings.TrimSpace(query.Get("from_agente")); value != "" {
		filter.FromAgente = &value
	}
	if value := strings.TrimSpace(query.Get("estado")); value != "" {
		filter.Estado = &value
	}
	return db.ListarRuntimeMailbox(filter)
}

func cargarArbolRuntimesRecuperacionLocal(query url.Values) ([]*db.RuntimeTreeNode, error) {
	proyectoID, err := runtimeProyectoIDDesdeQuery(query)
	if err != nil {
		return nil, err
	}
	filter := db.FiltroRuntimes{ProyectoID: proyectoID}
	if value := strings.TrimSpace(query.Get("agente")); value != "" {
		filter.Agente = &value
	}
	if value := strings.TrimSpace(query.Get("activos")); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "yes", "si", "on":
			v := true
			filter.Activos = &v
		case "0", "false", "no", "off":
			v := false
			filter.Activos = &v
		default:
			return nil, fmt.Errorf("--activos invalido: %s", value)
		}
	}
	return db.ConstruirArbolRuntimes(filter)
}

func cargarDetalleRuntimeRecuperacionLocal(id int64) (*apiRuntimeDetailResponse, error) {
	runtime, err := db.GetRuntime(id)
	if err != nil {
		return nil, err
	}
	samples, err := db.ListarMuestrasRuntime(id, 10)
	if err != nil {
		return nil, err
	}
	return &apiRuntimeDetailResponse{
		Runtime: runtime,
		Samples: samples,
	}, nil
}

func runtimeProyectoIDDesdeQuery(query url.Values) (*int64, error) {
	return runtimeProyectoIDRef(query.Get("proyecto"))
}

func runtimeProyectoIDRef(ref string) (*int64, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, nil
	}
	proyecto, err := db.GetProyecto(ref)
	if err != nil {
		return nil, err
	}
	return &proyecto.ID, nil
}

func runtimeOptionalInt64Query(query url.Values, key string) (*int64, error) {
	raw := strings.TrimSpace(query.Get(key))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s invalido", key)
	}
	return &value, nil
}

func runtimeOptionalIntQuery(query url.Values, key string) (int, error) {
	raw := strings.TrimSpace(query.Get(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s invalido", key)
	}
	return value, nil
}

func aplanarArbolAPI(nodes []*apiRuntimeTreeNode) []runtimeRow {
	rows := make([]runtimeRow, 0, len(nodes))
	aplanarNodosRuntimeAPI(nodes, 0, &rows)
	return rows
}

func aplanarNodosRuntimeAPI(nodes []*apiRuntimeTreeNode, nivel int, out *[]runtimeRow) {
	for _, node := range nodes {
		if node == nil || node.Runtime == nil {
			continue
		}
		*out = append(*out, runtimeRowDesdeModelo(node.Runtime, nivel, len(node.Hijos)))
		aplanarNodosRuntimeAPI(node.Hijos, nivel+1, out)
	}
}

func aplanarArbolLocal(nodes []*db.RuntimeTreeNode) []runtimeRow {
	rows := make([]runtimeRow, 0, len(nodes))
	aplanarNodosRuntimeLocal(nodes, 0, &rows)
	return rows
}

func aplanarNodosRuntimeLocal(nodes []*db.RuntimeTreeNode, nivel int, out *[]runtimeRow) {
	for _, node := range nodes {
		if node == nil || node.Runtime == nil {
			continue
		}
		*out = append(*out, runtimeRowDesdeModelo(node.Runtime, nivel, len(node.Hijos)))
		aplanarNodosRuntimeLocal(node.Hijos, nivel+1, out)
	}
}

func runtimeRowDesdeModelo(r *db.RuntimeInstance, nivel, hijos int) runtimeRow {
	if r == nil {
		return runtimeRow{}
	}
	pid := "—"
	if r.PID != nil {
		pid = strconv.FormatInt(*r.PID, 10)
	}
	proyecto := r.ProyectoSlug
	if strings.TrimSpace(proyecto) == "" {
		proyecto = "—"
	}
	ultima := "—"
	if r.LastHeartbeatAt != nil {
		ultima = r.LastHeartbeatAt.Format("2006-01-02 15:04:05")
	} else if r.LastEventAt != nil {
		ultima = r.LastEventAt.Format("2006-01-02 15:04:05")
	}
	return runtimeRow{
		ID:           r.ID,
		Nivel:        nivel,
		Agente:       r.Agente,
		Proyecto:     proyecto,
		Provider:     valorVacio(r.Provider),
		Connector:    valorVacio(r.Connector),
		Estado:       valorVacio(r.LogicalState),
		PID:          pid,
		Hijos:        hijos,
		Branch:       valorVacio(r.Branch),
		Modelo:       valorVacio(r.Model),
		Razonamiento: valorVacio(r.Reasoning),
		Ultima:       ultima,
	}
}

func imprimirArbolRuntimes(rows []runtimeRow, jsonOut bool) error {
	if len(rows) == 0 {
		fmt.Println("No hay runtimes con ese filtro.")
		return nil
	}
	if jsonOut {
		for _, row := range rows {
			fmt.Printf("%+v\n", row)
		}
		return nil
	}
	fmt.Printf("%-5s %-11s %-14s %-15s %-12s %-14s %-8s %-5s %-18s %s\n",
		"ID", "ESTADO", "AGENTE", "PROYECTO", "PROVIDER", "CONNECTOR", "PID", "HIJOS", "BRANCH", "ULTIMA")
	for _, row := range rows {
		indent := strings.Repeat("  ", row.Nivel)
		agente := indent + row.Agente
		fmt.Printf("%-5d %-11s %-14s %-15s %-12s %-14s %-8s %-5d %-18s %s\n",
			row.ID, truncar(row.Estado, 11), truncar(agente, 14), truncar(row.Proyecto, 15),
			truncar(row.Provider, 12), truncar(row.Connector, 14), row.PID, row.Hijos,
			truncar(row.Branch, 18), row.Ultima)
	}
	return nil
}

func imprimirDetalleRuntime(r *db.RuntimeInstance, samples []*db.RuntimeTelemetrySample) error {
	if r == nil {
		return fmt.Errorf("runtime no encontrado")
	}
	fmt.Printf("Runtime #%d — %s\n", r.ID, r.Agente)
	fmt.Printf("  Proyecto:     %s\n", valorVacio(r.ProyectoSlug))
	fmt.Printf("  Estado:       %s\n", valorVacio(r.LogicalState))
	fmt.Printf("  Proceso:      %s\n", valorVacio(r.ProcessState))
	fmt.Printf("  Provider:     %s\n", valorVacio(r.Provider))
	fmt.Printf("  Connector:    %s\n", valorVacio(r.Connector))
	fmt.Printf("  PID:          %s\n", formatoIntPtr(r.PID))
	fmt.Printf("  PPID:         %s\n", formatoIntPtr(r.PPID))
	fmt.Printf("  Hijos:        %d\n", r.ChildCount)
	fmt.Printf("  Hilos:        %d\n", r.ThreadCount)
	fmt.Printf("  Modelo:       %s\n", valorVacio(r.Model))
	fmt.Printf("  Razonamiento: %s\n", valorVacio(r.Reasoning))
	fmt.Printf("  Perfil:       %s\n", valorVacio(r.TaskProfile))
	fmt.Printf("  CWD:          %s\n", valorVacio(r.CWD))
	fmt.Printf("  Branch:       %s\n", valorVacio(r.Branch))
	fmt.Printf("  Sesión:       %s\n", valorVacio(r.ExternalSessionID))
	fmt.Printf("  Última señal:  %s\n", formatoTiempoPtr(r.LastHeartbeatAt))
	fmt.Printf("  Último evento: %s\n", formatoTiempoPtr(r.LastEventAt))

	if len(samples) > 0 {
		fmt.Println("\nMuestras recientes:")
		fmt.Printf("%-18s %-9s %-9s %-9s %-8s %-8s %-10s %s\n", "CREADA", "CPU%", "MEM", "RSS", "HIJOS", "HILOS", "FUENTE", "ESTADO")
		for _, s := range samples {
			fmt.Printf("%-18s %-9.1f %-9d %-9d %-8d %-8d %-10s %s\n",
				s.CreatedAt.Format("2006-01-02 15:04:05"), s.CPUPct, s.MemBytes, s.RSSBytes,
				s.ChildCount, s.ThreadCount, truncar(s.Source, 10), truncar(s.LogicalState, 18))
		}
	}
	return nil
}

func formatoIntPtr(v *int64) string {
	if v == nil {
		return "—"
	}
	return strconv.FormatInt(*v, 10)
}

func formatoTiempoPtr(v *time.Time) string {
	if v == nil {
		return "—"
	}
	return v.Format("2006-01-02 15:04:05")
}

func init() {
	runtimeListarCmd.Flags().String("agente", "", "Filtrar por agente")
	runtimeListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")
	runtimeListarCmd.Flags().String("activos", "", "Filtrar por activos=true|false")
	runtimeHandlesCmd.Flags().String("agente", "", "Filtrar runtime handles por agente")
	runtimePurgarHandlesCmd.Flags().String("agente", "", "Purga handles del agente indicado")
	runtimePurgarHandlesCmd.Flags().String("proyecto", "", "Purga handles del proyecto indicado")
	runtimePurgarHandlesCmd.Flags().StringSlice("estado", []string{"cerrado", "fallido"}, "Estados purgables; por seguridad solo cerrado/fallido")
	runtimePurgarHandlesCmd.Flags().String("actor", "orquesta", "Actor que solicita la purga")
	runtimePurgarOrdenesCmd.Flags().String("agente", "", "Purga órdenes del agente indicado")
	runtimePurgarOrdenesCmd.Flags().String("proyecto", "", "Purga órdenes del proyecto indicado")
	runtimePurgarOrdenesCmd.Flags().StringSlice("estado", []string{"completada", "fallida", "expirada", "cancelada"}, "Estados purgables; por seguridad solo terminales")
	runtimePurgarOrdenesCmd.Flags().StringSlice("tipo", nil, "Tipos de runtime order a purgar")
	runtimePurgarOrdenesCmd.Flags().Int("older-than-minutes", 60, "Purga solo órdenes terminales creadas hace más de N minutos")
	runtimePurgarOrdenesCmd.Flags().String("actor", "orquesta", "Actor que solicita la purga")
	runtimeTranscriptCmd.Flags().String("agente", "", "Filtrar transcript por agente")
	runtimeTranscriptCmd.Flags().String("proyecto", "", "Filtrar transcript por proyecto")
	runtimeTranscriptCmd.Flags().String("stream", "", "Filtrar transcript por stream")
	runtimeTranscriptCmd.Flags().String("classification", "", "Filtrar transcript por clasificación")
	runtimeTranscriptCmd.Flags().String("q", "", "Buscar texto libre en transcript")
	runtimeTranscriptCmd.Flags().Int64("runtime-id", 0, "Filtrar transcript por runtime_id")
	runtimeTranscriptCmd.Flags().Int64("handle-id", 0, "Filtrar transcript por handle_id")
	runtimeTranscriptCmd.Flags().Bool("signals-pending", false, "Mostrar solo señales pendientes de gestionar")
	runtimeTranscriptCmd.Flags().Int("limit", 50, "Número máximo de líneas de transcript")
	runtimeOrdenesCmd.Flags().String("agente", "", "Filtrar órdenes por agente")
	runtimeOrdenesCmd.Flags().String("estado", "", "Filtrar órdenes por estado")
	runtimeOrdenesCmd.Flags().String("proyecto", "", "Filtrar órdenes por proyecto")
	runtimeOrdenesCmd.Flags().Int("limit", 0, "Limitar órdenes devueltas")
	runtimeOrdenNuevaCmd.Flags().String("proyecto", "", "Proyecto asociado a la orden")
	runtimeOrdenNuevaCmd.Flags().String("payload", "{}", "Payload JSON de la orden")
	runtimeNudgeCmd.Flags().String("from", "server", "Agente o actor que emite el nudge")
	runtimeNudgeCmd.Flags().String("proyecto", "", "Proyecto asociado al nudge")
	runtimeNudgeCmd.Flags().String("kind", "nudge", "Kind del mensaje mailbox generado")
	runtimeDiscordiaCmd.Flags().String("from", "server", "Agente o actor que escala la discordia")
	runtimeDiscordiaCmd.Flags().String("proyecto", "", "Proyecto asociado a la discordia")
	runtimeCheckpointsCmd.Flags().String("agente", "", "Agente del checkpoint")
	runtimeCheckpointsCmd.Flags().String("proyecto", "", "Proyecto del checkpoint")
	runtimeCheckpointsCmd.Flags().String("kind", "", "Filtrar checkpoints por tipo")
	runtimeCheckpointsCmd.Flags().String("source", "", "Filtrar checkpoints por source")
	runtimeCheckpointsCmd.Flags().Int("limit", 1, "Numero maximo de checkpoints a mostrar")
	runtimeCheckpointNuevoCmd.Flags().String("proyecto", "", "Proyecto asociado al checkpoint")
	runtimeCheckpointNuevoCmd.Flags().Int64("sesion-id", 0, "Sesion asociada al checkpoint")
	runtimeCheckpointNuevoCmd.Flags().Int64("runtime-id", 0, "Runtime asociado al checkpoint")
	runtimeCheckpointNuevoCmd.Flags().String("kind", "manual", "Tipo de checkpoint")
	runtimeCheckpointNuevoCmd.Flags().String("resumen", "", "Resumen del checkpoint")
	runtimeCheckpointNuevoCmd.Flags().String("branch", "", "Branch asociado al checkpoint")
	runtimeCheckpointNuevoCmd.Flags().String("cwd", "", "Directorio de trabajo asociado al checkpoint")
	runtimeCheckpointNuevoCmd.Flags().String("payload", "{}", "Payload JSON del checkpoint")
	runtimeCheckpointNuevoCmd.Flags().String("resume-strategy", "resumen_y_payload", "Estrategia de reanudacion del checkpoint")
	runtimeCheckpointNuevoCmd.Flags().String("source", "runtime-cli", "Origen del checkpoint")
	runtimeMailboxCmd.Flags().String("to", "", "Filtrar mensajes por agente destino")
	runtimeMailboxCmd.Flags().String("from", "", "Filtrar mensajes por agente origen")
	runtimeMailboxCmd.Flags().String("estado", "", "Filtrar mensajes por estado")
	runtimeMailboxCmd.Flags().String("proyecto", "", "Filtrar mensajes por proyecto")
	runtimeMailboxEnviarCmd.Flags().String("proyecto", "", "Proyecto asociado al mensaje")
	runtimeMailboxEnviarCmd.Flags().String("payload", "{}", "Payload JSON del mensaje")
	runtimeMailboxEnviarCmd.Flags().Int64("runtime-order-id", 0, "Orden runtime asociada al mensaje")
	runtimeCmd.AddCommand(runtimeListarCmd, runtimeVerCmd, runtimeHandlesCmd, runtimePurgarHandlesCmd, runtimePurgarOrdenesCmd, runtimeTranscriptCmd, runtimeOrdenesCmd, runtimeOrdenNuevaCmd, runtimeNudgeCmd, runtimeDiscordiaCmd, runtimeCheckpointsCmd, runtimeCheckpointNuevoCmd, runtimeCheckpointVerCmd, runtimeMailboxCmd, runtimeMailboxEnviarCmd, runtimeMailboxEntregarCmd, runtimeMailboxConsumirCmd)
	rootCmd.AddCommand(runtimeCmd)
}
