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
		}

		var filtro *string
		if strings.TrimSpace(agente) != "" {
			filtro = &agente
		}
		handles, err := db.ListarRuntimeHandles(filtro)
		if err != nil {
			return err
		}
		return imprimirRuntimeHandles(handles)
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

var runtimeOrdenesCmd = &cobra.Command{
	Use:   "ordenes",
	Short: "Lista órdenes persistidas del control plane",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		estado, _ := cmd.Flags().GetString("estado")
		filter := db.FiltroRuntimeOrders{}
		if strings.TrimSpace(agente) != "" {
			filter.Agente = &agente
		}
		if strings.TrimSpace(estado) != "" {
			filter.Estado = &estado
		}
		query := url.Values{}
		if strings.TrimSpace(agente) != "" {
			query.Set("agente", agente)
		}
		if strings.TrimSpace(estado) != "" {
			query.Set("estado", estado)
		}
		if orders, ok, err := cargarRuntimeOrdersDesdeAPI(query); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeOrders(orders)
		}
		orders, err := db.ListarRuntimeOrders(filter)
		if err != nil {
			return err
		}
		return imprimirRuntimeOrders(orders)
	},
}

func imprimirRuntimeOrders(orders []*db.RuntimeOrder) error {
	if len(orders) == 0 {
		fmt.Println("No hay órdenes con ese filtro.")
		return nil
	}
	fmt.Printf("%-5s %-12s %-18s %-12s %s\n", "ID", "AGENTE", "TIPO", "ESTADO", "CREADA")
	for _, o := range orders {
		fmt.Printf("%-5d %-12s %-18s %-12s %s\n",
			o.ID, truncar(o.Agente, 12), truncar(o.Tipo, 18), truncar(o.Estado, 12),
			o.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
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

		var proyectoID *int64
		if strings.TrimSpace(proyectoRef) != "" {
			p, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			proyectoID = &p.ID
		}
		id, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
			Agente:      agente,
			ProyectoID:  proyectoID,
			Tipo:        tipo,
			PayloadJSON: payload,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Orden runtime #%d encolada para %s (%s)\n", id, agente, tipo)
		return nil
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

	var proyectoID *int64
	if strings.TrimSpace(proyectoRef) != "" {
		p, err := db.GetProyecto(proyectoRef)
		if err != nil {
			return 0, err
		}
		proyectoID = &p.ID
	}
	return db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      agenteDestino,
		ProyectoID:  proyectoID,
		Tipo:        tipo,
		PayloadJSON: string(payloadJSON),
	})
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
		} else if cp, ok, err := cargarCheckpointRuntimeDesdeAPI(agente, proyectoRef); ok {
			if err != nil {
				return err
			}
			return imprimirRuntimeCheckpoint(cp)
		}

		var proyectoID *int64
		if strings.TrimSpace(proyectoRef) != "" {
			p, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			proyectoID = &p.ID
		}
		if usarHistorialLocal {
			filter := db.FiltroRuntimeCheckpoints{
				Agente:     &agente,
				ProyectoID: proyectoID,
				Limit:      limit,
			}
			if strings.TrimSpace(checkpointKind) != "" {
				filter.CheckpointKind = &checkpointKind
			}
			if strings.TrimSpace(source) != "" {
				filter.Source = &source
			}
			checkpoints, err := db.ListarRuntimeCheckpoints(filter)
			if err != nil {
				return err
			}
			return imprimirRuntimeCheckpointLista(checkpoints)
		}
		cp, err := db.UltimoRuntimeCheckpoint(agente, proyectoID)
		if err != nil {
			return err
		}
		return imprimirRuntimeCheckpoint(cp)
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

		var proyectoID *int64
		if strings.TrimSpace(proyectoRef) != "" {
			p, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			proyectoID = &p.ID
		}
		var sesionIDPtr *int64
		if sesionID > 0 {
			sesionIDPtr = &sesionID
		}
		var runtimeIDPtr *int64
		if runtimeID > 0 {
			runtimeIDPtr = &runtimeID
		}
		id, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
			Agente:         agente,
			ProyectoID:     proyectoID,
			SesionID:       sesionIDPtr,
			RuntimeID:      runtimeIDPtr,
			CheckpointKind: checkpointKind,
			Resumen:        strings.TrimSpace(resumen),
			Branch:         strings.TrimSpace(branch),
			CWD:            strings.TrimSpace(cwd),
			PayloadJSON:    payload,
			ResumeStrategy: strings.TrimSpace(resumeStrategy),
			Source:         strings.TrimSpace(source),
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Checkpoint runtime #%d creado para %s (%s)\n", id, agente, checkpointKind)
		return nil
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
		}
		cp, err := db.GetRuntimeCheckpoint(id)
		if err != nil {
			return err
		}
		return imprimirRuntimeCheckpoint(cp)
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
		}

		filter := db.FiltroRuntimeMailbox{}
		if strings.TrimSpace(toAgente) != "" {
			filter.ToAgente = &toAgente
		}
		if strings.TrimSpace(fromAgente) != "" {
			filter.FromAgente = &fromAgente
		}
		if strings.TrimSpace(estado) != "" {
			filter.Estado = &estado
		}
		if strings.TrimSpace(proyectoRef) != "" {
			p, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			filter.ProyectoID = &p.ID
		}
		mailbox, err := db.ListarRuntimeMailbox(filter)
		if err != nil {
			return err
		}
		return imprimirRuntimeMailbox(mailbox)
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

		var proyectoID *int64
		if strings.TrimSpace(proyectoRef) != "" {
			p, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			proyectoID = &p.ID
		}
		var runtimeOrderIDPtr *int64
		if runtimeOrderID > 0 {
			runtimeOrderIDPtr = &runtimeOrderID
		}
		id, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:     fromAgente,
			ToAgente:       toAgente,
			ProyectoID:     proyectoID,
			RuntimeOrderID: runtimeOrderIDPtr,
			Kind:           kind,
			PayloadJSON:    payload,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Mensaje mailbox #%d enviado de %s a %s (%s)\n", id, fromAgente, toAgente, kind)
		return nil
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
		if err := db.MarcarRuntimeMailboxEntregado(id); err != nil {
			return err
		}
		fmt.Printf("✓ Mensaje mailbox #%d marcado como entregado\n", id)
		return nil
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
		if err := db.MarcarRuntimeMailboxConsumido(id); err != nil {
			return err
		}
		fmt.Printf("✓ Mensaje mailbox #%d marcado como consumido\n", id)
		return nil
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
		}

		filter := db.FiltroRuntimes{}
		if strings.TrimSpace(agente) != "" {
			filter.Agente = &agente
		}
		if strings.TrimSpace(proyecto) != "" {
			p, err := db.GetProyecto(proyecto)
			if err != nil {
				return err
			}
			filter.ProyectoID = &p.ID
		}
		if strings.TrimSpace(activos) != "" {
			switch activos {
			case "true":
				v := true
				filter.Activos = &v
			case "false":
				v := false
				filter.Activos = &v
			default:
				return fmt.Errorf("activos inválido")
			}
		}
		treeLocal, err := db.ConstruirArbolRuntimes(filter)
		if err != nil {
			return err
		}
		return imprimirArbolRuntimes(aplanarArbolLocal(treeLocal), false)
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
		}

		runtime, err := db.GetRuntime(id)
		if err != nil {
			return err
		}
		samples, err := db.ListarMuestrasRuntime(id, 20)
		if err != nil {
			return err
		}
		return imprimirDetalleRuntime(runtime, samples)
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
	runtimeOrdenesCmd.Flags().String("agente", "", "Filtrar órdenes por agente")
	runtimeOrdenesCmd.Flags().String("estado", "", "Filtrar órdenes por estado")
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
	runtimeCmd.AddCommand(runtimeListarCmd, runtimeVerCmd, runtimeHandlesCmd, runtimeOrdenesCmd, runtimeOrdenNuevaCmd, runtimeNudgeCmd, runtimeDiscordiaCmd, runtimeCheckpointsCmd, runtimeCheckpointNuevoCmd, runtimeCheckpointVerCmd, runtimeMailboxCmd, runtimeMailboxEnviarCmd, runtimeMailboxEntregarCmd, runtimeMailboxConsumirCmd)
	rootCmd.AddCommand(runtimeCmd)
}
