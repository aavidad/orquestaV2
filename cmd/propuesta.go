/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
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
		var f *db.EstadoPropuesta
		var proyectoID *int64
		if estadoStr != "" {
			e := db.EstadoPropuesta(estadoStr)
			f = &e
		}
		if proyectoRef != "" {
			p, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			proyectoID = &p.ID
		}
		propuestas, err := db.ListarPropuestas(f, proyectoID)
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
		p, err := db.GetPropuesta(args[0])
		if err != nil {
			return fmt.Errorf("propuesta '%s' no encontrada", args[0])
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

		votos, err := db.ResumenVotos(p.ID)
		if err != nil {
			return err
		}
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

		p := &db.Propuesta{
			Codigo:       codigo,
			Titulo:       titulo,
			Descripcion:  desc,
			Tipo:         tipo,
			PropuestoPor: por,
			Distribuidor: "claude",
		}
		if proyectoRef != "" {
			proyecto, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			p.ProyectoID = &proyecto.ID
		}
		id, err := db.CrearPropuesta(p)
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
		if err := db.CerrarPropuesta(args[0], args[1], agente); err != nil {
			return err
		}
		fmt.Printf("✓ Propuesta %s cerrada como '%s'\n", args[0], args[1])
		return nil
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
		insertados, err := db.ReabrirPropuesta(args[0], agente)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Propuesta %s reabierta\n", args[0])
		fmt.Printf("  Votos pendientes reconstruidos: %d\n", insertados)
		return nil
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
		insertados, err := db.RepararVotosPendientesPropuesta(args[0], agente)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Votos pendientes reparados en %s\n", args[0])
		fmt.Printf("  Filas insertadas: %d\n", insertados)
		return nil
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

	propuestaCerrarCmd.Flags().String("por", "alberto", "Agente que cierra")
	propuestaCerrarCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")
	propuestaReabrirCmd.Flags().String("por", "alberto", "Agente que reabre")
	propuestaReabrirCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")
	propuestaRepararVotosCmd.Flags().String("por", "alberto", "Agente que repara votos")
	propuestaRepararVotosCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")

	propuestaCmd.AddCommand(
		propuestaListarCmd,
		propuestaVerCmd,
		propuestaNuevaCmd,
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
		p, err := db.GetPropuesta(args[0])
		if err != nil {
			return fmt.Errorf("propuesta '%s' no encontrada", args[0])
		}
		votos, err := db.VotosDePropuesta(p.ID)
		if err != nil {
			return err
		}
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

