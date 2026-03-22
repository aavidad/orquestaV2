/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var proyectoCmd = &cobra.Command{
	Use:   "proyecto",
	Short: "Gestión de proyectos del workspace",
}

var proyectoListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista los proyectos registrados",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyectos, ok, err := cargarProyectosDesdeAPI()
		if !ok {
			proyectos, err = db.ListarProyectos(db.FiltroProyectos{})
		}
		if err != nil {
			return err
		}
		if len(proyectos) == 0 {
			fmt.Println("No hay proyectos registrados.")
			return nil
		}
		fmt.Printf("%-5s %-12s %-10s %-8s %s\n", "ID", "SLUG", "TIPO", "PADRE", "RUTA")
		fmt.Printf("%-5s %-12s %-10s %-8s %s\n", "─────", "────────────", "──────────", "────────", "────────────────────────────")
		for _, p := range proyectos {
			padre := "—"
			if p.ParentID != nil {
				padre = fmt.Sprintf("%d", *p.ParentID)
			}
			fmt.Printf("%-5d %-12s %-10s %-8s %s\n", p.ID, p.Slug, p.Tipo, padre, p.RutaAbs)
		}
		return nil
	},
}

var proyectoDescubrirCmd = &cobra.Command{
	Use:   "descubrir [ruta]",
	Short: "Descubre proyectos bajo una ruta del workspace y los registra",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruta := ""
		if len(args) == 1 {
			ruta = args[0]
		}
		proyectos, err := db.DescubrirProyectos(ruta)
		if err != nil {
			return err
		}
		if ruta != "" {
			abs, err := filepath.Abs(ruta)
			if err == nil {
				_ = db.ConfigSet("workspace_root", abs)
			}
		}
		fmt.Printf("✓ %d proyectos registrados/actualizados\n", len(proyectos))
		for _, p := range proyectos {
			fmt.Printf("  [%d] %-10s %-12s %s\n", p.ID, p.Tipo, p.Slug, p.RutaAbs)
		}
		return nil
	},
}

var proyectoVerCmd = &cobra.Command{
	Use:   "ver <slug|id>",
	Short: "Muestra el detalle de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, ok, err := cargarProyectoDesdeAPI(args[0])
		if !ok {
			p, err = db.GetProyecto(args[0])
		}
		if err != nil {
			return err
		}
		fmt.Printf("Proyecto #%d — %s\n", p.ID, p.Nombre)
		fmt.Printf("  Slug:      %s\n", p.Slug)
		fmt.Printf("  Tipo:      %s\n", p.Tipo)
		fmt.Printf("  Ruta ABS:  %s\n", p.RutaAbs)
		padre := "—"
		if p.ParentID != nil {
			if pad, err := db.GetProyecto(fmt.Sprintf("%d", *p.ParentID)); err == nil {
				padre = fmt.Sprintf("%d (%s)", *p.ParentID, pad.Slug)
			} else {
				padre = fmt.Sprintf("%d", *p.ParentID)
			}
		}
		fmt.Printf("  Padre:     %s\n", padre)
		fmt.Printf("  Activo:    %v\n", p.Activo)
		fmt.Printf("  Creado:    %s\n", p.CreatedAt.Format("2006-01-02 15:04"))

		return nil
	},
}

func init() {
	proyectoCmd.AddCommand(proyectoListarCmd, proyectoDescubrirCmd, proyectoVerCmd)
}
