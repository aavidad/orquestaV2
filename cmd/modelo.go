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
	"orquesta/capacidadapp"
	"orquesta/db"
)

var politicaModeloCmd = &cobra.Command{
	Use:   "politica-modelo",
	Short: "Gestion de politicas de seleccion de pool, modelo y reasoning",
}

var politicaModeloListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista politicas de modelo registradas",
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeTipo, _ := cmd.Flags().GetString("scope-tipo")
		scopeRef, _ := cmd.Flags().GetString("scope-ref")
		var activaPtr *bool
		if cmd.Flags().Changed("activa") {
			activa, _ := cmd.Flags().GetBool("activa")
			activaPtr = &activa
		}

		items, ok, err := cargarPoliticasModeloDesdeAPI(scopeTipo, scopeRef, activaPtr)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("politica-modelo listar")
		}
		if len(items) == 0 {
			fmt.Println("No hay politicas de modelo.")
			return nil
		}
		fmt.Printf("%-10s %-16s %-16s %-12s %-22s %-8s %s\n",
			"SCOPE", "REF", "PERFIL", "POOL", "MODELO", "PRIO", "REASONING")
		fmt.Printf("%-10s %-16s %-16s %-12s %-22s %-8s %s\n",
			"──────────", "────────────────", "────────────────", "────────────", "──────────────────────", "────────", "──────────")
		for _, item := range items {
			fmt.Printf("%-10s %-16s %-16s %-12s %-22s %-8d %s\n",
				item.ScopeTipo, emptyDash(item.ScopeRef), item.PerfilTarea,
				emptyDash(item.PoolSlug), emptyDash(item.ModelSlug), item.Prioridad, emptyDash(item.ReasoningEffort))
		}
		return nil
	},
}

var politicaModeloGuardarCmd = &cobra.Command{
	Use:   "guardar",
	Short: "Crea una politica de resolucion de modelo",
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeTipo, _ := cmd.Flags().GetString("scope-tipo")
		scopeRef, _ := cmd.Flags().GetString("scope-ref")
		perfil, _ := cmd.Flags().GetString("perfil")
		poolSlug, _ := cmd.Flags().GetString("pool")
		modelSlug, _ := cmd.Flags().GetString("modelo")
		reasoning, _ := cmd.Flags().GetString("reasoning")
		prioridad, _ := cmd.Flags().GetInt("prioridad")
		metadataJSON, _ := cmd.Flags().GetString("metadata-json")
		activa, _ := cmd.Flags().GetBool("activa")

		politica := &db.PoliticaModelo{
			ScopeTipo:       scopeTipo,
			ScopeRef:        scopeRef,
			PerfilTarea:     perfil,
			PoolSlug:        poolSlug,
			ModelSlug:       modelSlug,
			ReasoningEffort: reasoning,
			Prioridad:       prioridad,
			Activa:          activa,
			MetadataJSON:    metadataJSON,
		}
		id, ok, err := guardarPoliticaModeloPorAPI(politica)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("politica-modelo guardar")
		}
		fmt.Printf("✓ Politica de modelo guardada (id: %d)\n", id)
		return nil
	},
}

var politicaModeloSeedCmd = &cobra.Command{
	Use:   "seed-inicial",
	Short: "Carga politicas base para los perfiles recomendados",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ok, err := seedInicialPoliticasModeloPorAPI(); err != nil {
			return err
		} else if !ok {
			return serverFirstCommandError("politica-modelo seed-inicial")
		}
		fmt.Println("✓ Politicas iniciales de modelo cargadas")
		return nil
	},
}

var modeloCmd = &cobra.Command{
	Use:   "modelo",
	Short: "Resolucion de pool, modelo y reasoning para una tarea",
}

var modeloResolverCmd = &cobra.Command{
	Use:   "resolver",
	Short: "Resuelve pool, modelo y reasoning segun politicas y fallback",
	RunE: func(cmd *cobra.Command, args []string) error {
		var tareaIDPtr *int64
		if cmd.Flags().Changed("tarea-id") {
			tareaID, _ := cmd.Flags().GetInt64("tarea-id")
			tareaIDPtr = &tareaID
		}
		proyecto, _ := cmd.Flags().GetString("proyecto")
		fase, _ := cmd.Flags().GetString("fase")
		perfil, _ := cmd.Flags().GetString("perfil")

		input := db.ResolverPoliticaInput{
			TareaID:      tareaIDPtr,
			ProyectoSlug: proyecto,
			Fase:         fase,
			PerfilTarea:  perfil,
		}
		res, ok, err := resolverModeloPorAPI(input)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("modelo resolver")
		}

		fmt.Printf("Perfil:      %s\n", res.PerfilTarea)
		fmt.Printf("Pool:        %s (%s)\n", res.PoolSlug, res.FuentePool)
		fmt.Printf("Modelo:      %s (%s)\n", res.ModelSlug, res.FuenteModelo)
		fmt.Printf("Reasoning:   %s (%s)\n", res.ReasoningEffort, res.FuenteReasoning)
		if len(res.PoliticasAplicadas) > 0 {
			fmt.Println("\nPoliticas aplicadas:")
			for _, item := range res.PoliticasAplicadas {
				fmt.Printf("  - %s ref=%s perfil=%s prio=%d\n",
					item.ScopeTipo, emptyDash(item.ScopeRef), item.PerfilTarea, item.Prioridad)
			}
		}
		return nil
	},
}

var modeloPipelineLocalCmd = &cobra.Command{
	Use:   "pipeline-local",
	Short: "Describe el pipeline local determinista de orquestacion por fases",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")

		pipeline, ok, err := cargarPipelineLocalDeterministaDesdeAPI(proyecto)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("modelo pipeline-local")
		}

		imprimirEtapa := func(etapa capacidadapp.EtapaPipelineLocal) {
			fmt.Printf("- %s: perfil=%s modo=%s", etapa.Fase, emptyDash(etapa.PerfilTarea), emptyDash(etapa.ModoEjecucion))
			if etapa.RequiereModelo {
				fmt.Printf(" modelo=%s", emptyDash(etapa.ObjetivoModelo))
				if etapa.ModeloFallback != "" {
					fmt.Printf(" fallback=%s", etapa.ModeloFallback)
				}
			} else {
				fmt.Printf(" modelo=determinista_app")
			}
			fmt.Println()
		}

		fmt.Println("Pipeline local determinista:")
		if pipeline.ProyectoSlug != "" {
			fmt.Printf("Proyecto: %s\n", pipeline.ProyectoSlug)
		}
		for _, etapa := range pipeline.Fases {
			imprimirEtapa(etapa)
		}
		if pipeline.Revision != nil {
			fmt.Println("\nRevision escalonada:")
			for _, revisor := range pipeline.Revision.Revisores {
				fmt.Printf("- %s: clase=%s perfil=%s modelo=%s", revisor.NombreRol, emptyDash(revisor.Clase), emptyDash(revisor.PerfilTarea), emptyDash(revisor.ObjetivoModelo))
				if revisor.ModeloFallback != "" {
					fmt.Printf(" fallback=%s", revisor.ModeloFallback)
				}
				if revisor.Premium {
					fmt.Printf(" premium=true")
				}
				fmt.Println()
			}
			if len(pipeline.Revision.AbrirSegundaOpinionCuando) > 0 {
				fmt.Println("\nGates de segunda opinion:")
				for _, gate := range pipeline.Revision.AbrirSegundaOpinionCuando {
					fmt.Printf("- %s\n", gate)
				}
			}
		}
		return nil
	},
}

var modeloPipelinePasoCmd = &cobra.Command{
	Use:   "pipeline-paso",
	Short: "Calcula el siguiente paso determinista del pipeline local",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")

		paso, ok, err := calcularSiguientePasoPipelineLocalDeterministaDesdeAPI(proyecto)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("modelo pipeline-paso")
		}
		fmt.Printf("Accion:       %s\n", paso.Accion)
		fmt.Printf("Motivo:       %s\n", paso.Motivo)
		if paso.FaseActual != "" {
			fmt.Printf("Fase actual:  %s\n", paso.FaseActual)
		}
		if paso.FaseObjetivo != "" {
			fmt.Printf("Fase objetivo: %s\n", paso.FaseObjetivo)
		}
		if paso.EtapaObjetivo != nil && paso.EtapaObjetivo.RequiereModelo {
			fmt.Printf("Modelo:       %s\n", emptyDash(paso.EtapaObjetivo.ObjetivoModelo))
			if paso.EtapaObjetivo.ModeloFallback != "" {
				fmt.Printf("Fallback:     %s\n", paso.EtapaObjetivo.ModeloFallback)
			}
		}
		if paso.EtapaObjetivo != nil {
			fmt.Printf("Carril:       %s\n", emptyDash(paso.EtapaObjetivo.Carril))
			fmt.Printf("Entrega:      %s\n", emptyDash(paso.EtapaObjetivo.EntregaCanonica))
			if paso.EtapaObjetivo.RequiereWorktree {
				fmt.Printf("Worktree:     si\n")
			} else {
				fmt.Printf("Worktree:     no\n")
			}
			if paso.EtapaObjetivo.UsaMicroprograma {
				fmt.Printf("Microprog.:   si\n")
			} else {
				fmt.Printf("Microprog.:   no\n")
			}
		}
		if paso.TareaObjetivo != nil {
			fmt.Printf("Tarea:        #%d %s (%s)\n", paso.TareaObjetivo.ID, paso.TareaObjetivo.Titulo, paso.TareaObjetivo.Estado)
		}
		if paso.GateBloqueante != nil {
			fmt.Printf("Gate:         %d (%s)\n", paso.GateBloqueante.ID, paso.GateBloqueante.Estado)
		}
		return nil
	},
}

var modeloPipelineEjecutarCmd = &cobra.Command{
	Use:   "pipeline-ejecutar",
	Short: "Ejecuta el siguiente paso determinista del pipeline local",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")

		resultado, ok, err := ejecutarSiguientePasoPipelineLocalDeterministaDesdeAPI(proyecto)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("modelo pipeline-ejecutar")
		}
		if resultado == nil || resultado.Paso == nil {
			fmt.Println("No hay paso ejecutable.")
			return nil
		}
		fmt.Printf("Accion: %s\n", resultado.Paso.Accion)
		if resultado.FaseActivada != nil {
			fmt.Printf("Fase:   %s\n", *resultado.FaseActivada)
		}
		if resultado.TareaActualizada != nil {
			fmt.Printf("Tarea:  #%d %s (%s)\n", resultado.TareaActualizada.ID, resultado.TareaActualizada.Titulo, resultado.TareaActualizada.Estado)
		}
		if resultado.Despacho != nil {
			fmt.Printf("Carril: %s\n", emptyDash(resultado.Despacho.Carril))
			fmt.Printf("Entrega: %s\n", emptyDash(resultado.Despacho.EntregaCanonica))
			if resultado.Despacho.RequiereModelo {
				fmt.Printf("Modelo: %s\n", emptyDash(resultado.Despacho.ObjetivoModelo))
				if resultado.Despacho.ModeloFallback != "" {
					fmt.Printf("Fallback: %s\n", resultado.Despacho.ModeloFallback)
				}
			}
			fmt.Printf("Agente sugerido: %s\n", emptyDash(resultado.Despacho.AgenteSugerido))
			imprimirModoDespachoCLI(resultado.Despacho)
			imprimirSeleccionAgenteCLI(resultado.Despacho.SeleccionAgente)
		}
		return nil
	},
}

var modeloPipelineDespacharCmd = &cobra.Command{
	Use:   "pipeline-despachar",
	Short: "Ejecuta el siguiente paso del pipeline local y lo encola por la vía canónica cuando el carril lo permite",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")

		resultado, ok, err := despacharSiguientePasoPipelineLocalDeterministaDesdeAPI(proyecto)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("modelo pipeline-despachar")
		}
		if resultado == nil || resultado.Paso == nil {
			fmt.Println("No hay paso despachable.")
			return nil
		}
		fmt.Printf("Accion: %s\n", resultado.Paso.Accion)
		if resultado.Despacho != nil {
			fmt.Printf("Carril: %s\n", emptyDash(resultado.Despacho.Carril))
			fmt.Printf("Agente sugerido: %s\n", emptyDash(resultado.Despacho.AgenteSugerido))
			imprimirModoDespachoCLI(resultado.Despacho)
			imprimirSeleccionAgenteCLI(resultado.Despacho.SeleccionAgente)
		}
		if resultado.DispatchRuntime != nil {
			fmt.Printf("Estado dispatch: %s\n", emptyDash(resultado.DispatchRuntime.Estado))
			if strings.TrimSpace(resultado.DispatchRuntime.Motivo) != "" {
				fmt.Printf("Motivo: %s\n", strings.TrimSpace(resultado.DispatchRuntime.Motivo))
			}
			if resultado.DispatchRuntime.StartOrderID != nil {
				fmt.Printf("Start order: #%d\n", *resultado.DispatchRuntime.StartOrderID)
			}
			if resultado.DispatchRuntime.RuntimeOrderID != nil {
				fmt.Printf("Runtime order: #%d\n", *resultado.DispatchRuntime.RuntimeOrderID)
			}
		}
		return nil
	},
}

var modeloRuntimeActivosCmd = &cobra.Command{
	Use:   "runtime-activos",
	Short: "Lista modelos activos del runtime local por la via canonica de la app",
	RunE: func(cmd *cobra.Command, args []string) error {
		items, ok, err := cargarModelosRuntimeActivosDesdeAPI()
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("modelo runtime-activos")
		}
		if len(items) == 0 {
			fmt.Println("No hay modelos activos en el runtime local.")
			return nil
		}
		fmt.Printf("%-28s %-10s %-10s %-10s %s\n", "MODELO", "RUNTIME", "PROCES.", "CONTEXTO", "HASTA")
		fmt.Printf("%-28s %-10s %-10s %-10s %s\n", "────────────────────────────", "──────────", "──────────", "──────────", "────────────")
		for _, item := range items {
			fmt.Printf("%-28s %-10s %-10s %-10s %s\n",
				emptyDash(item.Modelo),
				emptyDash(item.Runtime),
				emptyDash(item.Procesador),
				emptyDash(item.Contexto),
				emptyDash(item.Hasta))
		}
		return nil
	},
}

var modeloRuntimeDescargarCmd = &cobra.Command{
	Use:   "runtime-descargar",
	Short: "Descarga de memoria todos los modelos activos del runtime local",
	RunE: func(cmd *cobra.Command, args []string) error {
		descargados, ok, err := descargarModelosRuntimeActivosPorAPI()
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("modelo runtime-descargar")
		}
		if len(descargados) == 0 {
			fmt.Println("No habia modelos activos para descargar.")
			return nil
		}
		fmt.Println("Modelos descargados:")
		for _, item := range descargados {
			fmt.Printf("- %s\n", item)
		}
		return nil
	},
}

func init() {
	politicaModeloListarCmd.Flags().String("scope-tipo", "", "Filtrar por scope_tipo")
	politicaModeloListarCmd.Flags().String("scope-ref", "", "Filtrar por scope_ref")
	politicaModeloListarCmd.Flags().Bool("activa", true, "Filtrar por politicas activas")

	politicaModeloGuardarCmd.Flags().String("scope-tipo", "global", "Scope de la politica: global|perfil|proyecto|fase|tarea")
	politicaModeloGuardarCmd.Flags().String("scope-ref", "", "Referencia del scope")
	politicaModeloGuardarCmd.Flags().String("perfil", "*", "Perfil de tarea afectado")
	politicaModeloGuardarCmd.Flags().String("pool", "", "Pool preferido")
	politicaModeloGuardarCmd.Flags().String("modelo", "", "Modelo preferido")
	politicaModeloGuardarCmd.Flags().String("reasoning", "", "Reasoning effort: low|medium|high (xhigh solo para arquitectura, seguridad o riesgo alto)")
	politicaModeloGuardarCmd.Flags().Int("prioridad", 100, "Prioridad de la politica")
	politicaModeloGuardarCmd.Flags().String("metadata-json", "{}", "Metadata JSON")
	politicaModeloGuardarCmd.Flags().Bool("activa", true, "Si la politica queda activa")

	modeloResolverCmd.Flags().Int64("tarea-id", 0, "ID de tarea concreta")
	modeloResolverCmd.Flags().String("proyecto", "", "Slug del proyecto")
	modeloResolverCmd.Flags().String("fase", "", "Fase de trabajo")
	modeloResolverCmd.Flags().String("perfil", "", "Perfil de tarea")
	modeloPipelineLocalCmd.Flags().String("proyecto", "", "Slug del proyecto")
	modeloPipelinePasoCmd.Flags().String("proyecto", "", "Slug del proyecto")
	modeloPipelineEjecutarCmd.Flags().String("proyecto", "", "Slug del proyecto")
	modeloPipelineDespacharCmd.Flags().String("proyecto", "", "Slug del proyecto")

	politicaModeloCmd.AddCommand(politicaModeloListarCmd, politicaModeloGuardarCmd, politicaModeloSeedCmd)
	modeloCmd.AddCommand(modeloResolverCmd, modeloPipelineLocalCmd, modeloPipelinePasoCmd, modeloPipelineEjecutarCmd, modeloPipelineDespacharCmd, modeloRuntimeActivosCmd, modeloRuntimeDescargarCmd)
	rootCmd.AddCommand(politicaModeloCmd, modeloCmd)
}

func emptyDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}
