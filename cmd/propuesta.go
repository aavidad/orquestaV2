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
	"orquesta/proposalapp"
)

var propuestaCmd = &cobra.Command{
	Use:   "propuesta",
	Short: "Gestión de propuestas (OPs)",
}

var proposalService = proposalapp.NewService(proposalapp.Repository{})

var propuestaListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista propuestas (por defecto: todas)",
	RunE: func(cmd *cobra.Command, args []string) error {
		estadoStr, _ := cmd.Flags().GetString("estado")
		var f *db.EstadoPropuesta
		if estadoStr != "" {
			e := db.EstadoPropuesta(estadoStr)
			f = &e
		}
		var (
			propuestas []*db.Propuesta
			err        error
		)
		if serverURL := activeServerURL(); serverURL != "" {
			query := url.Values{}
			if estadoStr != "" {
				query.Set("estado", estadoStr)
			}
			propuestas, err = fetchServerProposals(serverURL, query)
		} else {
			propuestas, err = proposalService.List(f)
		}
		if err != nil {
			return err
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
		var (
			detail *proposalapp.ProposalDetail
			err    error
		)
		if serverURL := activeServerURL(); serverURL != "" {
			detail, err = fetchServerProposalDetail(serverURL, args[0])
		} else {
			detail, err = proposalService.GetDetail(args[0])
		}
		if err != nil {
			return fmt.Errorf("propuesta '%s' no encontrada", args[0])
		}
		p := detail.Proposal

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

		votos := detail.Votes
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

		if serverURL := activeServerURL(); serverURL != "" {
			id, codigo, err := submitServerCreateProposal(serverURL, map[string]any{
				"codigo":        codigo,
				"titulo":        titulo,
				"descripcion":   desc,
				"tipo":          tipo,
				"propuesto_por": por,
				"distribuidor":  "claude",
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Propuesta %s creada (id: %d)\n", codigo, id)
			fmt.Println("  Los agentes deben votar con: orquesta votar <codigo> <posicion>")
			return nil
		}

		id, p, err := proposalService.Create(proposalapp.CreateProposalInput{
			Codigo:       codigo,
			Titulo:       titulo,
			Descripcion:  desc,
			Tipo:         tipo,
			PropuestoPor: por,
			Distribuidor: "claude",
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Propuesta %s creada (id: %d)\n", p.Codigo, id)
		fmt.Println("  Los agentes deben votar con: orquesta votar <codigo> <posicion>")
		return nil
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
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerCloseProposal(serverURL, args[0], args[1], agente); err != nil {
				return err
			}
			fmt.Printf("✓ Propuesta %s cerrada como '%s'\n", args[0], args[1])
			return nil
		}
		if err := proposalService.Close(args[0], args[1], agente); err != nil {
			return err
		}
		fmt.Printf("✓ Propuesta %s cerrada como '%s'\n", args[0], args[1])
		return nil
	},
}

func init() {
	propuestaListarCmd.Flags().String("estado", "", "Filtrar por estado (abierta, consenso, rechazada, backlog)")

	propuestaNuevaCmd.Flags().String("titulo", "", "Título de la propuesta (obligatorio)")
	propuestaNuevaCmd.Flags().String("descripcion", "", "Descripción detallada")
	propuestaNuevaCmd.Flags().String("tipo", "implementacion", "Tipo: implementacion, arquitectura, seguridad, backlog, otro")
	propuestaNuevaCmd.Flags().String("por", "alberto", "Propuesto por")
	propuestaNuevaCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")
	propuestaNuevaCmd.Flags().String("codigo", "", "Código manual (si se omite, se auto-genera OP-XXX)")

	propuestaCerrarCmd.Flags().String("por", "alberto", "Agente que cierra")
	propuestaCerrarCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")

	propuestaCmd.AddCommand(propuestaListarCmd, propuestaVerCmd, propuestaNuevaCmd, propuestaCerrarCmd)
}
