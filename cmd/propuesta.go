/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var propuestaCmd = &cobra.Command{
	Use:   "propuesta",
	Short: "Gestión de propuestas (OPs)",
}

var propuestaListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista propuestas (por defecto: todas)",
	RunE: func(cmd *cobra.Command, args []string) error {
		estadoStr, _ := cmd.Flags().GetString("estado")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		var propuestas []*db.Propuesta
		params := url.Values{}
		if estadoStr != "" {
			params.Set("estado", estadoStr)
		}
		if proyectoRef != "" {
			params.Set("proyecto", proyectoRef)
		}
		var resp apiPropuestasResponse
		if ok, err := apiGetQuery("/api/propuestas", params, &resp); err != nil {
			return err
		} else if ok {
			propuestas = resp.Propuestas
		} else {
			return serverFirstCommandError("propuesta listar")
		}
		if len(propuestas) == 0 {
			fmt.Println("No hay propuestas.")
			return nil
		}
		fmt.Printf("%-8s %-10s %-15s %s\n", "CÓDIGO", "ESTADO", "TIPO", "TÍTULO")
		fmt.Printf("%-8s %-10s %-15s %s\n", "────────", "──────────", "───────────────", "────────────────────────────────────")
		for _, p := range propuestas {
			fmt.Printf("%-8s %-10s %-15s %s\n", p.Codigo, p.Estado, p.Tipo, truncar(p.Titulo, 40))
		}
		return nil
	},
}

var propuestaVerCmd = &cobra.Command{
	Use:   "ver <codigo>",
	Short: "Muestra el detalle y votos de una propuesta",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var p *db.Propuesta
		var resp apiPropuestaDetalleResponse
		if ok, err := apiGet("/api/propuestas/"+args[0], &resp); err != nil {
			return err
		} else if ok {
			p = resp.Propuesta
		} else {
			return serverFirstCommandError("propuesta ver")
		}

		fmt.Printf("╔══════════════════════════════════════════════╗\n")
		fmt.Printf("║  %s — %s\n", p.Codigo, p.Titulo)
		fmt.Printf("╚══════════════════════════════════════════════╝\n")
		fmt.Printf("Estado:       %s\n", p.Estado)
		fmt.Printf("Tipo:         %s\n", p.Tipo)
		fmt.Printf("Propuesto por:%s\n", p.PropuestoPor)
		fmt.Printf("Creada:       %s\n", p.CreatedAt.Format("2006-01-02 15:04"))
		if p.CerradaAt != nil {
			fmt.Printf("Cerrada:      %s\n", p.CerradaAt.Format("2006-01-02 15:04"))
		}
		if p.Descripcion != "" {
			fmt.Printf("\nDescripción:\n  %s\n", strings.ReplaceAll(p.Descripcion, "\n", "\n  "))
		}

		votos := p.Votos
		if len(votos) > 0 {
			fmt.Printf("\nVotos:\n")
			for _, v := range votos {
				comentario := ""
				if v.Comentario != "" {
					comentario = " — " + v.Comentario
				}
				fmt.Printf("  %-15s %-12s %s\n", v.Agente, v.Posicion, comentario)
			}
		}
		return nil
	},
}

var propuestaNuevaCmd = &cobra.Command{
	Use:   "nueva [titulo]",
	Short: "Crea una nueva propuesta",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		titulo, _ := cmd.Flags().GetString("titulo")
		desc, _ := cmd.Flags().GetString("descripcion")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		tipo, _ := cmd.Flags().GetString("tipo")
		codigo, _ := cmd.Flags().GetString("codigo")
		if strings.TrimSpace(titulo) == "" && len(args) == 1 {
			titulo = strings.TrimSpace(args[0])
		}
		por, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}

		if titulo == "" {
			return fmt.Errorf("debe indicar el título con --titulo o como argumento posicional")
		}

		var resp struct {
			OK        bool          `json:"ok"`
			ID        int64         `json:"id"`
			Propuesta *db.Propuesta `json:"propuesta"`
		}
		if ok, err := apiPost("/api/propuestas", apiPropuestaCrearRequest{
			Codigo:       codigo,
			Titulo:       titulo,
			Descripcion:  desc,
			Proyecto:     proyectoRef,
			Tipo:         tipo,
			PropuestoPor: por,
			Distribuidor: "claude",
		}, &resp); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Propuesta %s creada (id: %d)\n", resp.Propuesta.Codigo, resp.ID)
			fmt.Println("  Los agentes deben votar con: orquesta votar <codigo> <posicion>")
			return nil
		}
		return serverFirstCommandError("propuesta nueva")
	},
}

var propuestaActualizarCmd = &cobra.Command{
	Use:   "actualizar <codigo>",
	Short: "Actualiza una propuesta existente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}

		var (
			titulo     *string
			desc       *string
			appendDesc *string
			tipo       *string
		)
		if cmd.Flags().Changed("titulo") {
			v, _ := cmd.Flags().GetString("titulo")
			titulo = &v
		}
		if cmd.Flags().Changed("descripcion") {
			v, _ := cmd.Flags().GetString("descripcion")
			desc = &v
		}
		if cmd.Flags().Changed("anexar-descripcion") {
			v, _ := cmd.Flags().GetString("anexar-descripcion")
			appendDesc = &v
		}
		if cmd.Flags().Changed("tipo") {
			v, _ := cmd.Flags().GetString("tipo")
			tipo = &v
		}
		if titulo == nil && desc == nil && appendDesc == nil && tipo == nil {
			return fmt.Errorf("debes indicar al menos un cambio con --titulo, --descripcion, --anexar-descripcion o --tipo")
		}

		if ok, err := apiPost("/api/propuestas/"+args[0]+"/accion", apiPropuestaAccionRequest{
			Accion:      "actualizar",
			Agente:      agente,
			Titulo:      titulo,
			Descripcion: desc,
			AnexarDesc:  appendDesc,
			Tipo:        tipo,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Propuesta %s actualizada\n", args[0])
			return nil
		}
		return serverFirstCommandError("propuesta actualizar")
	},
}

var propuestaCerrarCmd = &cobra.Command{
	Use:   "cerrar <codigo> <estado>",
	Short: "Cierra manualmente una propuesta (consenso|rechazada|backlog)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		if ok, err := apiPost("/api/propuestas/"+args[0]+"/accion", apiPropuestaAccionRequest{
			Accion:       "cerrar",
			Agente:       agente,
			EstadoCierre: args[1],
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Propuesta %s cerrada como '%s'\n", args[0], args[1])
			return nil
		}
		return serverFirstCommandError("propuesta cerrar")
	},
}

var propuestaReabrirCmd = &cobra.Command{
	Use:   "reabrir <codigo>",
	Short: "Reabre una propuesta y reconstruye votos pendientes faltantes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		if ok, err := apiPost("/api/propuestas/"+args[0]+"/accion", apiPropuestaAccionRequest{
			Accion: "reabrir",
			Agente: agente,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Propuesta %s reabierta\n", args[0])
			return nil
		}
		return serverFirstCommandError("propuesta reabrir")
	},
}

var propuestaRepararVotosCmd = &cobra.Command{
	Use:   "reparar-votos <codigo>",
	Short: "Reconstruye filas de voto pendiente faltantes en una propuesta",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		if ok, err := apiPost("/api/propuestas/"+args[0]+"/accion", apiPropuestaAccionRequest{
			Accion: "reparar_votos",
			Agente: agente,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Votos pendientes reparados en %s\n", args[0])
			return nil
		}
		return serverFirstCommandError("propuesta reparar-votos")
	},
}

func init() {
	propuestaListarCmd.Flags().String("estado", "", "Filtrar por estado (abierta, consenso, rechazada, backlog)")
	propuestaListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")

	propuestaNuevaCmd.Flags().String("titulo", "", "Título de la propuesta (obligatorio)")
	propuestaNuevaCmd.Flags().String("descripcion", "", "Descripción detallada")
	propuestaNuevaCmd.Flags().String("proyecto", "", "Proyecto al que aplica la propuesta")
	propuestaNuevaCmd.Flags().String("tipo", "implementacion", "Tipo: implementacion, arquitectura, seguridad, backlog, otro")
	propuestaNuevaCmd.Flags().String("por", "alberto", "Propuesto por")
	propuestaNuevaCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")
	propuestaNuevaCmd.Flags().String("codigo", "", "Código manual (si se omite, se auto-genera OP-XXX)")
	propuestaActualizarCmd.Flags().String("titulo", "", "Nuevo título")
	propuestaActualizarCmd.Flags().String("descripcion", "", "Nueva descripción completa")
	propuestaActualizarCmd.Flags().String("anexar-descripcion", "", "Texto a anexar a la descripción existente")
	propuestaActualizarCmd.Flags().String("tipo", "", "Nuevo tipo")
	propuestaActualizarCmd.Flags().String("por", "alberto", "Agente que actualiza")
	propuestaActualizarCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")

	propuestaCerrarCmd.Flags().String("por", "alberto", "Agente que cierra")
	propuestaCerrarCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")
	propuestaReabrirCmd.Flags().String("por", "alberto", "Agente que reabre")
	propuestaReabrirCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")
	propuestaRepararVotosCmd.Flags().String("por", "alberto", "Agente que repara votos")
	propuestaRepararVotosCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")
	propuestaVotosCmd.Flags().String("agente", "", "Filtrar votos por agente (case-insensitive)")

	propuestaCmd.AddCommand(
		propuestaListarCmd,
		propuestaVerCmd,
		propuestaNuevaCmd,
		propuestaActualizarCmd,
		propuestaCerrarCmd,
		propuestaReabrirCmd,
		propuestaRepararVotosCmd,
		propuestaVotosCmd,
	)
}

// propuestaVotosCmd lista los votos de una propuesta con detalle de comentarios.
var propuestaVotosCmd = &cobra.Command{
	Use:   "votos <codigo>",
	Short: "Lista los votos de una propuesta con comentarios completos",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agenteFiltro, _ := cmd.Flags().GetString("agente")
		var p *db.Propuesta
		var votos []*db.Voto
		var resp apiPropuestaDetalleResponse
		if ok, err := apiGet("/api/propuestas/"+args[0], &resp); err != nil {
			return err
		} else if ok {
			p = resp.Propuesta
			votos = p.Votos
		} else {
			return serverFirstCommandError("propuesta votos")
		}
		votos = filtrarVotosPorAgente(votos, agenteFiltro)
		fmt.Printf("Votos de %s — %s\n", p.Codigo, p.Titulo)
		fmt.Printf("Estado: %s\n", p.Estado)
		fmt.Println("─────────────────────────────────────────")
		if len(votos) == 0 {
			fmt.Println("(sin votos registrados)")
			return nil
		}
		for _, v := range votos {
			icono := "⏳"
			switch v.Posicion {
			case "acuerdo":
				icono = "✓"
			case "desacuerdo":
				icono = "✗"
			case "abstencion":
				icono = "～"
			}
			fmt.Printf("  %s %-16s [%s]\n", icono, v.Agente, v.Posicion)
			if v.Comentario != "" {
				fmt.Printf("     %s\n", v.Comentario)
			}
		}
		return nil
	},
}

func filtrarVotosPorAgente(votos []*db.Voto, agente string) []*db.Voto {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return votos
	}
	filtrados := make([]*db.Voto, 0, len(votos))
	for _, voto := range votos {
		if strings.EqualFold(voto.Agente, agente) {
			filtrados = append(filtrados, voto)
		}
	}
	return filtrados
}
