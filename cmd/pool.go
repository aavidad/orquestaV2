/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/capacidadapp"
	"orquesta/db"
)

var poolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Gestion de pools de capacidad y modelos",
}

func capacidadModoRecuperacionLocalExplicito() bool {
	return strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1"
}

func capacidadErrorServerFirst() error {
	return fmt.Errorf("este comando exige servidor/daemon de Orquesta; usa --local solo en recuperacion explicita o exporta ORQUESTA_FORCE_LOCAL_DB=1")
}

var capacidadService = capacidadapp.NewService(capacidadapp.Repository{})

var poolListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista pools de capacidad",
	RunE: func(cmd *cobra.Command, args []string) error {
		activo, err := cmd.Flags().GetBool("activo")
		if err != nil {
			return err
		}
		var filtro *bool
		if cmd.Flags().Changed("activo") {
			filtro = &activo
		}

		pools, ok, err := cargarPoolsDesdeAPI(filtro)
		if err != nil {
			return err
		}
		if !ok {
			if !capacidadModoRecuperacionLocalExplicito() {
				return capacidadErrorServerFirst()
			}
			pools, err = capacidadService.ListPoolsSummary(filtro)
			if err != nil {
				return err
			}
		}
		if len(pools) == 0 {
			fmt.Println("No hay pools.")
			return nil
		}
		fmt.Printf("%-12s %-12s %-12s %-6s %-6s %-6s %s\n",
			"SLUG", "PROVEEDOR", "RUNTIME", "TOTAL", "USO", "LIBRE", "PLAN")
		fmt.Printf("%-12s %-12s %-12s %-6s %-6s %-6s %s\n",
			"────────────", "────────────", "────────────", "──────", "──────", "──────", "────────────")
		for _, p := range pools {
			fmt.Printf("%-12s %-12s %-12s %-6d %-6d %-6d %s\n",
				p.Pool.Slug, p.Pool.Proveedor, p.Pool.Runtime,
				p.Pool.CapacidadTotal, p.SesionesActivas, p.CapacidadDisponible, p.Pool.Plan)
		}
		return nil
	},
}

var poolVerCmd = &cobra.Command{
	Use:   "ver <slug>",
	Short: "Muestra detalle de un pool y sus modelos",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		detail, ok, err := cargarPoolDetalleDesdeAPI(args[0])
		if err != nil {
			return err
		}
		if !ok {
			if !capacidadModoRecuperacionLocalExplicito() {
				return capacidadErrorServerFirst()
			}
			detail, err = capacidadService.GetPoolDetail(args[0])
			if err != nil {
				return err
			}
		}
		pool := detail.Pool
		fmt.Printf("Pool %s\n", pool.Slug)
		fmt.Printf("  Proveedor:            %s\n", pool.Proveedor)
		fmt.Printf("  Runtime:              %s\n", pool.Runtime)
		fmt.Printf("  Plan:                 %s\n", pool.Plan)
		fmt.Printf("  Capacidad total:      %d\n", pool.CapacidadTotal)
		fmt.Printf("  Capacidad reservada:  %d\n", pool.CapacidadReservada)
		fmt.Printf("  Sesiones activas:     %d\n", detail.SesionesActivas)
		fmt.Printf("  Capacidad disponible: %d\n", detail.CapacidadDisponible)
		fmt.Printf("  Politica handoff:     %s\n", pool.PoliticaHandoff)
		fmt.Printf("  Fuente telemetria:    %s\n", pool.FuenteTelemetria)
		fmt.Printf("  Activo:               %t\n", pool.Activo)

		if len(detail.Modelos) > 0 {
			fmt.Println("\nModelos:")
			for _, m := range detail.Modelos {
				fmt.Printf("  - %s  prioridad=%d  coste=%.2f  activo=%t\n",
					m.ModelSlug, m.Prioridad, m.CosteRelativo, m.Activo)
			}
		}
		return nil
	},
}

var poolGuardarCmd = &cobra.Command{
	Use:   "guardar <slug>",
	Short: "Crea o actualiza un pool de capacidad",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proveedor, _ := cmd.Flags().GetString("proveedor")
		runtime, _ := cmd.Flags().GetString("runtime")
		plan, _ := cmd.Flags().GetString("plan")
		esDePago, _ := cmd.Flags().GetBool("es-de-pago")
		capacidadTotal, _ := cmd.Flags().GetInt("capacidad-total")
		capacidadReservada, _ := cmd.Flags().GetInt("capacidad-reservada")
		permiteHijos, _ := cmd.Flags().GetBool("permite-hijos")
		permiteModelosMulti, _ := cmd.Flags().GetBool("permite-modelos-multi")
		permiteSobrecoste, _ := cmd.Flags().GetBool("permite-sobrecoste")
		politicaHandoff, _ := cmd.Flags().GetString("politica-handoff")
		fuenteTelemetria, _ := cmd.Flags().GetString("fuente-telemetria")
		metadataJSON, _ := cmd.Flags().GetString("metadata-json")
		activo, _ := cmd.Flags().GetBool("activo")

		pool := &db.PoolCapacidad{
			Slug:                args[0],
			Proveedor:           proveedor,
			Runtime:             runtime,
			Plan:                plan,
			EsDePago:            esDePago,
			CapacidadTotal:      capacidadTotal,
			CapacidadReservada:  capacidadReservada,
			PermiteHijos:        permiteHijos,
			PermiteModelosMulti: permiteModelosMulti,
			PermiteSobrecoste:   permiteSobrecoste,
			PoliticaHandoff:     politicaHandoff,
			FuenteTelemetria:    fuenteTelemetria,
			MetadataJSON:        metadataJSON,
			Activo:              activo,
		}
		id, ok, err := guardarPoolPorAPI(pool)
		if err != nil {
			return err
		}
		if !ok {
			if !capacidadModoRecuperacionLocalExplicito() {
				return capacidadErrorServerFirst()
			}
			id, err = capacidadService.SavePool(pool)
			if err != nil {
				return err
			}
		}
		fmt.Printf("✓ Pool %s guardado (id: %d)\n", args[0], id)
		return nil
	},
}

var poolModeloCmd = &cobra.Command{
	Use:   "modelo",
	Short: "Gestion de modelos dentro de un pool",
}

var poolModeloListarCmd = &cobra.Command{
	Use:   "listar <pool-slug>",
	Short: "Lista modelos de un pool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelos, ok, err := cargarModelosPoolDesdeAPI(args[0])
		if err != nil {
			return err
		}
		if !ok {
			if !capacidadModoRecuperacionLocalExplicito() {
				return capacidadErrorServerFirst()
			}
			modelos, err = capacidadService.ListPoolModels(args[0])
			if err != nil {
				return err
			}
		}
		if len(modelos) == 0 {
			fmt.Println("No hay modelos para ese pool.")
			return nil
		}
		fmt.Printf("%-24s %-10s %-8s %s\n", "MODEL", "PRIORIDAD", "ACTIVO", "COSTE")
		fmt.Printf("%-24s %-10s %-8s %s\n", "────────────────────────", "──────────", "────────", "────────")
		for _, m := range modelos {
			fmt.Printf("%-24s %-10d %-8t %.2f\n", m.ModelSlug, m.Prioridad, m.Activo, m.CosteRelativo)
		}
		return nil
	},
}

var poolModeloGuardarCmd = &cobra.Command{
	Use:   "guardar <pool-slug> <model-slug>",
	Short: "Crea o actualiza un modelo dentro de un pool",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		activo, _ := cmd.Flags().GetBool("activo")
		prioridad, _ := cmd.Flags().GetInt("prioridad")
		costeRelativo, _ := cmd.Flags().GetFloat64("coste-relativo")
		limiteJSON, _ := cmd.Flags().GetString("limite-json")

		modelo := &db.PoolModelo{
			ModelSlug:          args[1],
			Activo:             activo,
			Prioridad:          prioridad,
			CosteRelativo:      costeRelativo,
			LimiteConocidoJSON: limiteJSON,
		}

		id, ok, err := guardarModeloPoolPorAPI(args[0], modelo)
		if err != nil {
			return err
		}
		if !ok {
			if !capacidadModoRecuperacionLocalExplicito() {
				return capacidadErrorServerFirst()
			}
			id, err = capacidadService.SavePoolModel(args[0], modelo)
			if err != nil {
				return err
			}
		}
		fmt.Printf("✓ Modelo %s guardado en pool %s (id: %d)\n", args[1], args[0], id)
		return nil
	},
}

var poolModeloSeedCmd = &cobra.Command{
	Use:   "seed-inicial",
	Short: "Carga modelos iniciales conocidos para los pools base",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ok, err := seedInicialModelosPoolPorAPI(); err != nil {
			return err
		} else if !ok {
			if !capacidadModoRecuperacionLocalExplicito() {
				return capacidadErrorServerFirst()
			}
			if err := capacidadService.SeedInitialModels(); err != nil {
				return err
			}
		}
		fmt.Println("✓ Modelos iniciales cargados")
		return nil
	},
}

var poolSeedCmd = &cobra.Command{
	Use:   "seed-inicial",
	Short: "Carga pools iniciales conocidos",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ok, err := seedInicialPoolsPorAPI(); err != nil {
			return err
		} else if ok {
			fmt.Println("✓ Pools iniciales cargados")
			return nil
		}
		if !capacidadModoRecuperacionLocalExplicito() {
			return capacidadErrorServerFirst()
		}
		seeds := []db.PoolCapacidad{
			{
				Slug:                "codex",
				Proveedor:           "OpenAI",
				Runtime:             "codex",
				Plan:                "default",
				EsDePago:            true,
				CapacidadTotal:      4,
				CapacidadReservada:  0,
				PermiteHijos:        true,
				PermiteModelosMulti: true,
				PermiteSobrecoste:   false,
				PoliticaHandoff:     "preventivo",
				FuenteTelemetria:    "manual",
				MetadataJSON:        "{}",
				Activo:              true,
			},
			{
				Slug:                "claude",
				Proveedor:           "Anthropic",
				Runtime:             "claude",
				Plan:                "default",
				EsDePago:            true,
				CapacidadTotal:      1,
				CapacidadReservada:  0,
				PermiteHijos:        true,
				PermiteModelosMulti: true,
				PermiteSobrecoste:   false,
				PoliticaHandoff:     "preventivo",
				FuenteTelemetria:    "manual",
				MetadataJSON:        "{}",
				Activo:              true,
			},
			{
				Slug:                "android",
				Proveedor:           "Android",
				Runtime:             "android",
				Plan:                "default",
				EsDePago:            false,
				CapacidadTotal:      1,
				CapacidadReservada:  0,
				PermiteHijos:        false,
				PermiteModelosMulti: false,
				PermiteSobrecoste:   false,
				PoliticaHandoff:     "preventivo",
				FuenteTelemetria:    "manual",
				MetadataJSON:        "{}",
				Activo:              true,
			},
		}
		for _, seed := range seeds {
			if _, err := capacidadService.SavePool(&seed); err != nil {
				return err
			}
		}
		fmt.Printf("✓ Pools iniciales cargados: %s\n", strconv.Itoa(len(seeds)))
		return nil
	},
}

func init() {
	poolListarCmd.Flags().Bool("activo", true, "Filtrar por pools activos")

	poolGuardarCmd.Flags().String("proveedor", "", "Proveedor del pool")
	poolGuardarCmd.Flags().String("runtime", "", "Runtime del pool")
	poolGuardarCmd.Flags().String("plan", "", "Plan del pool")
	poolGuardarCmd.Flags().Bool("es-de-pago", false, "Si el pool es de pago")
	poolGuardarCmd.Flags().Int("capacidad-total", 1, "Capacidad total del pool")
	poolGuardarCmd.Flags().Int("capacidad-reservada", 0, "Capacidad reservada del pool")
	poolGuardarCmd.Flags().Bool("permite-hijos", true, "Permite agentes hijos")
	poolGuardarCmd.Flags().Bool("permite-modelos-multi", true, "Permite varios modelos")
	poolGuardarCmd.Flags().Bool("permite-sobrecoste", false, "Permite sobrecoste")
	poolGuardarCmd.Flags().String("politica-handoff", "preventivo", "Politica de handoff")
	poolGuardarCmd.Flags().String("fuente-telemetria", "manual", "Fuente de telemetria")
	poolGuardarCmd.Flags().String("metadata-json", "{}", "Metadata JSON")
	poolGuardarCmd.Flags().Bool("activo", true, "Si el pool esta activo")

	poolModeloGuardarCmd.Flags().Bool("activo", true, "Si el modelo esta activo")
	poolModeloGuardarCmd.Flags().Int("prioridad", 100, "Prioridad del modelo")
	poolModeloGuardarCmd.Flags().Float64("coste-relativo", 1.0, "Coste relativo del modelo")
	poolModeloGuardarCmd.Flags().String("limite-json", "{}", "JSON con limite conocido")

	poolModeloCmd.AddCommand(poolModeloListarCmd, poolModeloGuardarCmd, poolModeloSeedCmd)
	poolCmd.AddCommand(poolListarCmd, poolVerCmd, poolGuardarCmd, poolSeedCmd, poolModeloCmd)
	rootCmd.AddCommand(poolCmd)
}
