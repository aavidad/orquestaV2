/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
			items, err = capacidadService.ListModelPolicies(scopeTipo, scopeRef, activaPtr)
			if err != nil {
				return err
			}
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
			id, err = capacidadService.SaveModelPolicy(politica)
			if err != nil {
				return err
			}
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
			if err := capacidadService.SeedInitialModelPolicies(); err != nil {
				return err
			}
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
			res, err = capacidadService.ResolveModelPolicy(input)
			if err != nil {
				return err
			}
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

func init() {
	politicaModeloListarCmd.Flags().String("scope-tipo", "", "Filtrar por scope_tipo")
	politicaModeloListarCmd.Flags().String("scope-ref", "", "Filtrar por scope_ref")
	politicaModeloListarCmd.Flags().Bool("activa", true, "Filtrar por politicas activas")

	politicaModeloGuardarCmd.Flags().String("scope-tipo", "global", "Scope de la politica: global|perfil|proyecto|fase|tarea")
	politicaModeloGuardarCmd.Flags().String("scope-ref", "", "Referencia del scope")
	politicaModeloGuardarCmd.Flags().String("perfil", "*", "Perfil de tarea afectado")
	politicaModeloGuardarCmd.Flags().String("pool", "", "Pool preferido")
	politicaModeloGuardarCmd.Flags().String("modelo", "", "Modelo preferido")
	politicaModeloGuardarCmd.Flags().String("reasoning", "", "Reasoning effort: low|medium|high|xhigh")
	politicaModeloGuardarCmd.Flags().Int("prioridad", 100, "Prioridad de la politica")
	politicaModeloGuardarCmd.Flags().String("metadata-json", "{}", "Metadata JSON")
	politicaModeloGuardarCmd.Flags().Bool("activa", true, "Si la politica queda activa")

	modeloResolverCmd.Flags().Int64("tarea-id", 0, "ID de tarea concreta")
	modeloResolverCmd.Flags().String("proyecto", "", "Slug del proyecto")
	modeloResolverCmd.Flags().String("fase", "", "Fase de trabajo")
	modeloResolverCmd.Flags().String("perfil", "", "Perfil de tarea")

	politicaModeloCmd.AddCommand(politicaModeloListarCmd, politicaModeloGuardarCmd, politicaModeloSeedCmd)
	modeloCmd.AddCommand(modeloResolverCmd)
	rootCmd.AddCommand(politicaModeloCmd, modeloCmd)
}

func emptyDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}
