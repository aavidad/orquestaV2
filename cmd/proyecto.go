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
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("proyecto listar")
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
		proyectos, ok, err := descubrirProyectosPorAPI(ruta)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("proyecto descubrir")
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
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("proyecto ver")
		}
		fmt.Printf("Proyecto #%d — %s\n", p.ID, p.Nombre)
		fmt.Printf("  Slug:      %s\n", p.Slug)
		fmt.Printf("  Tipo:      %s\n", p.Tipo)
		fmt.Printf("  Ruta ABS:  %s\n", p.RutaAbs)
		padre := "—"
		if p.ParentID != nil {
			if ok {
				if pad, ok, err := cargarProyectoDesdeAPI(fmt.Sprintf("%d", *p.ParentID)); ok && err == nil && pad != nil {
					padre = fmt.Sprintf("%d (%s)", *p.ParentID, pad.Slug)
				} else {
					padre = fmt.Sprintf("%d", *p.ParentID)
				}
			}
		}
		fmt.Printf("  Padre:     %s\n", padre)
		fmt.Printf("  Activo:    %v\n", p.Activo)
		fmt.Printf("  Creado:    %s\n", p.CreatedAt.Format("2006-01-02 15:04"))

		return nil
	},
}

var proyectoFusionarCmd = &cobra.Command{
	Use:   "fusionar <origen> <destino>",
	Short: "Fusiona un proyecto duplicado dentro del proyecto canónico y archiva el origen",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		archivarOrigen, _ := cmd.Flags().GetBool("archivar-origen")
		resultado, ok, err := fusionarProyectoPorAPI(args[0], args[1], archivarOrigen)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("proyecto fusionar")
		}
		fmt.Printf("✓ Proyecto %s fusionado en %s\n", args[0], args[1])
		if resultado != nil {
			var conflictos int64
			for _, n := range resultado.Conflictos {
				conflictos += n
			}
			fmt.Printf("  Origen ID:        %d\n", resultado.OrigenID)
			fmt.Printf("  Destino ID:       %d\n", resultado.DestinoID)
			if resultado.SlugArchivado != "" {
				fmt.Printf("  Slug archivado:   %s\n", resultado.SlugArchivado)
			}
			if resultado.RutaArchivada != "" {
				fmt.Printf("  Ruta archivada:   %s\n", resultado.RutaArchivada)
			}
			fmt.Printf("  Actualizaciones:  %d\n", resultado.TotalActualizaciones())
			if conflictos > 0 {
				fmt.Printf("  Conflictos:       %d\n", conflictos)
			}
		}
		return nil
	},
}

var proyectoActualizarCmd = &cobra.Command{
	Use:   "actualizar <slug|id>",
	Short: "Actualiza la ruta u otros campos de un proyecto registrado",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruta, _ := cmd.Flags().GetString("ruta")
		nombre, _ := cmd.Flags().GetString("nombre")
		tipo, _ := cmd.Flags().GetString("tipo")
		req := apiProyectoActualizarRequest{
			RutaAbs: ruta,
			Nombre:  nombre,
			Tipo:    tipo,
		}
		p, ok, err := actualizarProyectoPorAPI(args[0], req)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("proyecto actualizar")
		}
		fmt.Printf("✓ Proyecto #%d actualizado\n", p.ID)
		fmt.Printf("  Slug:     %s\n", p.Slug)
		fmt.Printf("  Ruta ABS: %s\n", p.RutaAbs)
		return nil
	},
}

var proyectoMicrocicloCmd = &cobra.Command{
	Use:   "microciclo [slug|id|ruta]",
	Short: "Activa un ciclo de micro-refactorización autónoma sobre el proyecto actual",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := ""
		if len(args) == 1 {
			ref = args[0]
		}
		agente, _ := cmd.Flags().GetString("agente")
		objetivo, _ := cmd.Flags().GetString("objetivo")
		definitionOfDone, _ := cmd.Flags().GetString("dod")
		titulo, _ := cmd.Flags().GetString("titulo")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		modulo, _ := cmd.Flags().GetString("modulo")
		notas, _ := cmd.Flags().GetString("notas")
		limpiarPruebas, _ := cmd.Flags().GetBool("limpiar-pruebas")

		req := apiProyectoMicrocicloRequest{
			Agente:               strings.TrimSpace(agente),
			ObjetivoGeneral:      strings.TrimSpace(objetivo),
			DefinitionOfDoneJSON: strings.TrimSpace(definitionOfDone),
			Titulo:               strings.TrimSpace(titulo),
			Descripcion:          strings.TrimSpace(descripcion),
			Modulo:               strings.TrimSpace(modulo),
			Notas:                strings.TrimSpace(notas),
			LimpiarPruebas:       limpiarPruebas,
		}
		if req.Agente == "" {
			req.Agente = "Codex1"
		}

		if shouldPreferAPIClient([]string{"proyecto", "microciclo"}) {
			proyecto, ok, err := resolverProyectoMicrocicloPorAPI(ref)
			if err != nil {
				return err
			}
			if !ok || proyecto == nil {
				return serverFirstCommandError("proyecto microciclo")
			}
			resultado, ok, err := activarMicrocicloProyectoPorAPI(proyecto.Slug, req)
			if err != nil {
				return err
			}
			if !ok || resultado == nil {
				return serverFirstCommandError("proyecto microciclo")
			}
			fmt.Printf("✓ Microciclo activado en %s con %s\n", resultado.Proyecto.Slug, resultado.Policy.SupervisorAgente)
			fmt.Printf("  Fase:      %s\n", resultado.Fase.Nombre)
			fmt.Printf("  Tarea:     #%d %s\n", resultado.Tarea.ID, resultado.Tarea.Titulo)
			fmt.Printf("  Reutiliza: %v\n", resultado.TareaReutilizada)
			if resultado.Dispatch != nil && resultado.Dispatch.DispatchRuntime != nil {
				fmt.Printf("  Dispatch:  %s\n", resultado.Dispatch.DispatchRuntime.Estado)
			}
			return nil
		}

		proyecto, err := resolverProyectoMicrocicloLocal(ref)
		if err != nil {
			return err
		}
		resultado, err := activarMicrocicloProyecto(proyecto.Slug, proyectoMicrocicloRequest{
			Agente:               req.Agente,
			ObjetivoGeneral:      req.ObjetivoGeneral,
			DefinitionOfDoneJSON: req.DefinitionOfDoneJSON,
			Titulo:               req.Titulo,
			Descripcion:          req.Descripcion,
			Modulo:               req.Modulo,
			Notas:                req.Notas,
			LimpiarPruebas:       req.LimpiarPruebas,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Microciclo activado en %s con %s\n", resultado.Proyecto.Slug, resultado.Policy.SupervisorAgente)
		fmt.Printf("  Fase:      %s\n", resultado.Fase.Nombre)
		fmt.Printf("  Tarea:     #%d %s\n", resultado.Tarea.ID, resultado.Tarea.Titulo)
		fmt.Printf("  Reutiliza: %v\n", resultado.TareaReutilizada)
		if resultado.Dispatch != nil && resultado.Dispatch.DispatchRuntime != nil {
			fmt.Printf("  Dispatch:  %s\n", resultado.Dispatch.DispatchRuntime.Estado)
		}
		return nil
	},
}

func init() {
	proyectoFusionarCmd.Flags().Bool("archivar-origen", true, "Archiva el proyecto origen tras la fusión")
	proyectoActualizarCmd.Flags().String("ruta", "", "Nueva ruta absoluta del proyecto")
	proyectoActualizarCmd.Flags().String("nombre", "", "Nuevo nombre del proyecto")
	proyectoActualizarCmd.Flags().String("tipo", "", "Nuevo tipo del proyecto")
	proyectoMicrocicloCmd.Flags().String("agente", "Codex1", "Agente que llevará el microciclo")
	proyectoMicrocicloCmd.Flags().String("objetivo", "", "Objetivo general de autonomía; por defecto usa el de micro-refactorización")
	proyectoMicrocicloCmd.Flags().String("dod", "", "Definition of done en JSON")
	proyectoMicrocicloCmd.Flags().String("titulo", "", "Título de la tarea semilla")
	proyectoMicrocicloCmd.Flags().String("descripcion", "", "Descripción de la tarea semilla")
	proyectoMicrocicloCmd.Flags().String("modulo", "controlplane", "Módulo de la tarea semilla")
	proyectoMicrocicloCmd.Flags().String("notas", "", "Notas extra para la tarea semilla")
	proyectoMicrocicloCmd.Flags().Bool("limpiar-pruebas", false, "Limpia frente y runtime residual antes de arrancar el microciclo")
	proyectoCmd.AddCommand(proyectoListarCmd, proyectoDescubrirCmd, proyectoVerCmd, proyectoFusionarCmd, proyectoActualizarCmd, proyectoMicrocicloCmd)
}
