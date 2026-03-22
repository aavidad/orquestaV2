/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var poolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Gestión de pools de capacidad y selección de modelo",
}

var poolListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista pools de capacidad",
	RunE: func(cmd *cobra.Command, args []string) error {
		todos, _ := cmd.Flags().GetBool("todos")
		pools, err := db.ListarPoolsCapacidad(todos)
		if err != nil {
			return err
		}
		if len(pools) == 0 {
			fmt.Println("No hay pools con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-20s %-12s %-12s %-8s %s\n", "ID", "SLUG", "PROVEEDOR", "RUNTIME", "ACTIVO", "CAPACIDAD")
		for _, pool := range pools {
			fmt.Printf("%-5d %-20s %-12s %-12s %-8s %d/%d\n",
				pool.ID, pool.Slug, pool.Proveedor, pool.Runtime, siNo(pool.Activo),
				pool.CapacidadReservada, pool.CapacidadTotal,
			)
		}
		return nil
	},
}

var poolCrearCmd = &cobra.Command{
	Use:   "crear",
	Short: "Crea un pool de capacidad",
	RunE: func(cmd *cobra.Command, args []string) error {
		pool, err := poolDesdeFlags(cmd, nil)
		if err != nil {
			return err
		}
		id, err := db.CrearPoolCapacidad(pool)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Pool #%d creado\n", id)
		return nil
	},
}

var poolEditarCmd = &cobra.Command{
	Use:   "editar <id>",
	Short: "Edita un pool de capacidad",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		actual, err := db.GetPoolCapacidad(id)
		if err != nil {
			return err
		}
		pool, err := poolDesdeFlags(cmd, actual)
		if err != nil {
			return err
		}
		pool.ID = id
		if err := db.ActualizarPoolCapacidad(pool); err != nil {
			return err
		}
		fmt.Printf("✓ Pool #%d actualizado\n", id)
		return nil
	},
}

var poolActivarCmd = activarPoolCmd(true)
var poolDesactivarCmd = activarPoolCmd(false)

var poolModeloCmd = &cobra.Command{
	Use:   "modelo",
	Short: "Gestión de modelos por pool",
}

var poolModeloListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista modelos de un pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		poolID, err := cmd.Flags().GetInt64("pool")
		if err != nil || poolID <= 0 {
			return fmt.Errorf("--pool es obligatorio")
		}
		todos, _ := cmd.Flags().GetBool("todos")
		modelos, err := db.ListarPoolModelos(poolID, todos)
		if err != nil {
			return err
		}
		if len(modelos) == 0 {
			fmt.Println("No hay modelos con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-8s %-10s %-16s %s\n", "ID", "ACTIVO", "PRIORIDAD", "COSTE", "MODEL")
		for _, modelo := range modelos {
			fmt.Printf("%-5d %-8s %-10d %-16.2f %s\n", modelo.ID, siNo(modelo.Activo), modelo.Prioridad, modelo.CosteRelativo, modelo.ModelSlug)
		}
		return nil
	},
}

var poolModeloRegistrarCmd = &cobra.Command{
	Use:   "registrar",
	Short: "Registra un modelo en un pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		modelo, err := modeloDesdeFlags(cmd, nil)
		if err != nil {
			return err
		}
		id, err := db.RegistrarPoolModelo(modelo)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Modelo #%d registrado\n", id)
		return nil
	},
}

var poolModeloEditarCmd = &cobra.Command{
	Use:   "editar <id>",
	Short: "Edita un modelo de pool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		actual, err := db.GetPoolModelo(id)
		if err != nil {
			return err
		}
		modelo, err := modeloDesdeFlags(cmd, actual)
		if err != nil {
			return err
		}
		modelo.ID = id
		if err := db.ActualizarPoolModelo(modelo); err != nil {
			return err
		}
		fmt.Printf("✓ Modelo #%d actualizado\n", id)
		return nil
	},
}

var poolModeloActivarCmd = activarPoolModeloCmd(true)
var poolModeloDesactivarCmd = activarPoolModeloCmd(false)

var poolModeloSeleccionarCmd = &cobra.Command{
	Use:   "seleccionar",
	Short: "Selecciona el mejor modelo para un agente y pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		poolID, err := int64FlagOpt(cmd, "pool")
		if err != nil {
			return err
		}
		perfil, _ := cmd.Flags().GetString("perfil")

		sel, err := db.SeleccionarModeloParaAgente(strings.TrimSpace(agente), poolID, strings.TrimSpace(perfil))
		if err != nil {
			return err
		}
		fmt.Printf("Agente:     %s\n", sel.Agente)
		fmt.Printf("Perfil:     %s\n", sel.Perfil)
		fmt.Printf("Pool:       %s\n", sel.Pool.Slug)
		fmt.Printf("Modelo:     %s\n", sel.Modelo.ModelSlug)
		fmt.Printf("Criterio:   %s\n", sel.Criterio)
		if sel.Evaluacion != nil {
			fmt.Printf("Presupuesto:%s", " "+sel.Evaluacion.Estado)
			if sel.Evaluacion.Motivo != "" {
				fmt.Printf(" (%s)", sel.Evaluacion.Motivo)
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	poolListarCmd.Flags().Bool("todos", false, "Incluir pools inactivos")
	poolCrearCmd.Flags().String("slug", "", "Slug único del pool")
	poolCrearCmd.Flags().String("proveedor", "", "Proveedor del pool")
	poolCrearCmd.Flags().String("runtime", "", "Runtime del pool")
	poolCrearCmd.Flags().String("plan", "", "Plan del pool")
	poolCrearCmd.Flags().Bool("es-de-pago", false, "Indica si es un pool de pago")
	poolCrearCmd.Flags().Int64("capacidad-total", 1, "Capacidad total")
	poolCrearCmd.Flags().Int64("capacidad-reservada", 0, "Capacidad reservada")
	poolCrearCmd.Flags().Bool("permite-hijos", true, "Permite jerarquía")
	poolCrearCmd.Flags().Bool("permite-modelos-multi", true, "Permite varios modelos")
	poolCrearCmd.Flags().Bool("permite-sobrecoste", false, "Permite sobrecoste")
	poolCrearCmd.Flags().String("politica-handoff", "preventivo", "Política de handoff")
	poolCrearCmd.Flags().String("fuente-telemetria", "manual", "Fuente de telemetría")
	poolCrearCmd.Flags().String("metadata", "{}", "Metadata JSON")

	poolEditarCmd.Flags().String("slug", "", "Nuevo slug")
	poolEditarCmd.Flags().String("proveedor", "", "Nuevo proveedor")
	poolEditarCmd.Flags().String("runtime", "", "Nuevo runtime")
	poolEditarCmd.Flags().String("plan", "", "Nuevo plan")
	poolEditarCmd.Flags().Bool("es-de-pago", false, "Indica si es un pool de pago")
	poolEditarCmd.Flags().Int64("capacidad-total", 0, "Nueva capacidad total")
	poolEditarCmd.Flags().Int64("capacidad-reservada", 0, "Nueva capacidad reservada")
	poolEditarCmd.Flags().Bool("permite-hijos", true, "Permite jerarquía")
	poolEditarCmd.Flags().Bool("permite-modelos-multi", true, "Permite varios modelos")
	poolEditarCmd.Flags().Bool("permite-sobrecoste", false, "Permite sobrecoste")
	poolEditarCmd.Flags().String("politica-handoff", "", "Nueva política de handoff")
	poolEditarCmd.Flags().String("fuente-telemetria", "", "Nueva fuente de telemetría")
	poolEditarCmd.Flags().String("metadata", "", "Nueva metadata JSON")

	poolModeloListarCmd.Flags().Int64("pool", 0, "ID del pool")
	poolModeloListarCmd.Flags().Bool("todos", false, "Incluir modelos inactivos")
	poolModeloRegistrarCmd.Flags().Int64("pool", 0, "ID del pool")
	poolModeloRegistrarCmd.Flags().String("slug", "", "Slug del modelo")
	poolModeloRegistrarCmd.Flags().Int64("prioridad", 100, "Prioridad del modelo")
	poolModeloRegistrarCmd.Flags().Float64("coste-relativo", 1.0, "Coste relativo")
	poolModeloRegistrarCmd.Flags().String("limite-conocido", "{}", "Límite conocido JSON")
	poolModeloEditarCmd.Flags().Int64("pool", 0, "ID del pool")
	poolModeloEditarCmd.Flags().String("slug", "", "Nuevo slug del modelo")
	poolModeloEditarCmd.Flags().Int64("prioridad", 0, "Nueva prioridad")
	poolModeloEditarCmd.Flags().Float64("coste-relativo", 0, "Nuevo coste relativo")
	poolModeloEditarCmd.Flags().String("limite-conocido", "", "Nuevo límite conocido JSON")
	poolModeloSeleccionarCmd.Flags().String("agente", "", "Agente objetivo")
	poolModeloSeleccionarCmd.Flags().Int64("pool", 0, "ID del pool")
	poolModeloSeleccionarCmd.Flags().String("perfil", "", "Perfil de selección: implementacion, economico, revision...")

	poolModeloCmd.AddCommand(poolModeloListarCmd, poolModeloRegistrarCmd, poolModeloEditarCmd, poolModeloActivarCmd, poolModeloDesactivarCmd, poolModeloSeleccionarCmd)
	poolCmd.AddCommand(poolListarCmd, poolCrearCmd, poolEditarCmd, poolActivarCmd, poolDesactivarCmd, poolModeloCmd)
	rootCmd.AddCommand(poolCmd)
}

func activarPoolCmd(activo bool) *cobra.Command {
	uso := "activar"
	short := "Activa un pool"
	if !activo {
		uso = "desactivar"
		short = "Desactiva un pool"
	}
	return &cobra.Command{
		Use:   uso + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("id inválido")
			}
			if err := db.SetPoolCapacidadActivo(id, activo); err != nil {
				return err
			}
			fmt.Printf("✓ Pool #%d %s\n", id, estadoVerbo(activo))
			return nil
		},
	}
}

func activarPoolModeloCmd(activo bool) *cobra.Command {
	uso := "activar"
	short := "Activa un modelo"
	if !activo {
		uso = "desactivar"
		short = "Desactiva un modelo"
	}
	return &cobra.Command{
		Use:   uso + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("id inválido")
			}
			if err := db.SetPoolModeloActivo(id, activo); err != nil {
				return err
			}
			fmt.Printf("✓ Modelo #%d %s\n", id, estadoVerbo(activo))
			return nil
		},
	}
}

func poolDesdeFlags(cmd *cobra.Command, base *db.PoolCapacidad) (*db.PoolCapacidad, error) {
	var pool db.PoolCapacidad
	if base != nil {
		pool = *base
	}
	if cmd.Flags().Changed("slug") || base == nil {
		v, _ := cmd.Flags().GetString("slug")
		pool.Slug = strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("proveedor") || base == nil {
		v, _ := cmd.Flags().GetString("proveedor")
		pool.Proveedor = strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("runtime") || base == nil {
		v, _ := cmd.Flags().GetString("runtime")
		pool.Runtime = strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("plan") || base == nil {
		v, _ := cmd.Flags().GetString("plan")
		pool.Plan = strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("es-de-pago") || base == nil {
		v, _ := cmd.Flags().GetBool("es-de-pago")
		pool.EsDePago = v
	}
	if cmd.Flags().Changed("capacidad-total") || base == nil {
		v, _ := cmd.Flags().GetInt64("capacidad-total")
		pool.CapacidadTotal = v
	}
	if cmd.Flags().Changed("capacidad-reservada") || base == nil {
		v, _ := cmd.Flags().GetInt64("capacidad-reservada")
		pool.CapacidadReservada = v
	}
	if cmd.Flags().Changed("permite-hijos") || base == nil {
		v, _ := cmd.Flags().GetBool("permite-hijos")
		pool.PermiteHijos = v
	}
	if cmd.Flags().Changed("permite-modelos-multi") || base == nil {
		v, _ := cmd.Flags().GetBool("permite-modelos-multi")
		pool.PermiteModelosMulti = v
	}
	if cmd.Flags().Changed("permite-sobrecoste") || base == nil {
		v, _ := cmd.Flags().GetBool("permite-sobrecoste")
		pool.PermiteSobrecoste = v
	}
	if cmd.Flags().Changed("politica-handoff") || base == nil {
		v, _ := cmd.Flags().GetString("politica-handoff")
		pool.PoliticaHandoff = strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("fuente-telemetria") || base == nil {
		v, _ := cmd.Flags().GetString("fuente-telemetria")
		pool.FuenteTelemetria = strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("metadata") || base == nil {
		v, _ := cmd.Flags().GetString("metadata")
		pool.MetadataJSON = strings.TrimSpace(v)
	}
	pool.Activo = true
	if base != nil {
		pool.Activo = base.Activo
	}
	return &pool, nil
}

func modeloDesdeFlags(cmd *cobra.Command, base *db.PoolModelo) (*db.PoolModelo, error) {
	var modelo db.PoolModelo
	if base != nil {
		modelo = *base
	}
	if cmd.Flags().Changed("pool") || base == nil {
		v, _ := cmd.Flags().GetInt64("pool")
		modelo.PoolID = v
	}
	if cmd.Flags().Changed("slug") || base == nil {
		v, _ := cmd.Flags().GetString("slug")
		modelo.ModelSlug = strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("prioridad") || base == nil {
		v, _ := cmd.Flags().GetInt64("prioridad")
		modelo.Prioridad = v
	}
	if cmd.Flags().Changed("coste-relativo") || base == nil {
		v, _ := cmd.Flags().GetFloat64("coste-relativo")
		modelo.CosteRelativo = v
	}
	if cmd.Flags().Changed("limite-conocido") || base == nil {
		v, _ := cmd.Flags().GetString("limite-conocido")
		modelo.LimiteConocidoJSON = strings.TrimSpace(v)
	}
	modelo.Activo = true
	if base != nil {
		modelo.Activo = base.Activo
	}
	return &modelo, nil
}
