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
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

func ptrBool(v bool) *bool {
	return &v
}

var reglasCmd = &cobra.Command{
	Use:   "reglas",
	Short: "Gestión CLI de reglas de briefing",
}

var reglasListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista reglas por rol o de todos los roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		rol, _ := cmd.Flags().GetString("rol")
		todas, _ := cmd.Flags().GetBool("todas")
		var activa *bool
		if !todas {
			activa = ptrBool(true)
		}
		reglas, err := db.ListarReglas(strings.TrimSpace(rol), activa)
		if err != nil {
			return err
		}
		if len(reglas) == 0 {
			fmt.Println("No hay reglas con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-15s %-8s %-16s %s\n", "ID", "ROL", "ACTIVA", "CATEGORIA", "TITULO")
		for _, r := range reglas {
			fmt.Printf("%-5d %-15s %-8s %-16s %s\n", r.ID, r.TipoAgente, siNo(r.Activa), r.Categoria, r.Titulo)
		}
		return nil
	},
}

var reglasCrearCmd = &cobra.Command{
	Use:   "crear",
	Short: "Crea una nueva regla",
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		rol, _ := cmd.Flags().GetString("rol")
		categoria, _ := cmd.Flags().GetString("categoria")
		titulo, _ := cmd.Flags().GetString("titulo")
		descripcion, _ := cmd.Flags().GetString("descripcion")

		id, err := db.CrearRegla(actor, &db.Regla{
			TipoAgente:  strings.TrimSpace(rol),
			Categoria:   strings.TrimSpace(categoria),
			Titulo:      strings.TrimSpace(titulo),
			Descripcion: strings.TrimSpace(descripcion),
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Regla #%d creada\n", id)
		return nil
	},
}

var reglasEditarCmd = &cobra.Command{
	Use:   "editar <id>",
	Short: "Edita una regla existente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		r, err := db.GetRegla(id)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("rol") {
			rol, _ := cmd.Flags().GetString("rol")
			r.TipoAgente = strings.TrimSpace(rol)
		}
		if cmd.Flags().Changed("categoria") {
			categoria, _ := cmd.Flags().GetString("categoria")
			r.Categoria = strings.TrimSpace(categoria)
		}
		if cmd.Flags().Changed("titulo") {
			titulo, _ := cmd.Flags().GetString("titulo")
			r.Titulo = strings.TrimSpace(titulo)
		}
		if cmd.Flags().Changed("descripcion") {
			descripcion, _ := cmd.Flags().GetString("descripcion")
			r.Descripcion = strings.TrimSpace(descripcion)
		}
		if err := db.ActualizarRegla(actor, r); err != nil {
			return err
		}
		fmt.Printf("✓ Regla #%d actualizada\n", id)
		return nil
	},
}

var reglasActivarCmd = activarReglaCmd(true)
var reglasDesactivarCmd = activarReglaCmd(false)
var reglasVersionesCmd = &cobra.Command{
	Use:   "versiones <id>",
	Short: "Lista el historial de versiones de una regla",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		versiones, err := db.ListarVersionesRegla(id)
		if err != nil {
			return err
		}
		if len(versiones) == 0 {
			fmt.Println("No hay versiones registradas para esa regla.")
			return nil
		}
		fmt.Printf("%-5s %-8s %-15s %-12s %-24s %s\n", "VER", "ACTIVA", "ACTOR", "ACCION", "ROL", "TITULO")
		for _, v := range versiones {
			fmt.Printf("%-5d %-8s %-15s %-12s %-24s %s\n", v.VersionNum, siNo(v.Activa), v.Actor, v.Accion, v.TipoAgente, v.Titulo)
		}
		return nil
	},
}

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Gestión CLI de skills",
}

var skillsListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista skills por rol o de todos los roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		rol, _ := cmd.Flags().GetString("rol")
		todos, _ := cmd.Flags().GetBool("todos")
		var activa *bool
		if !todos {
			activa = ptrBool(true)
		}
		skills, err := db.ListarSkills(strings.TrimSpace(rol), activa)
		if err != nil {
			return err
		}
		if len(skills) == 0 {
			fmt.Println("No hay skills con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-15s %-8s %-24s %s\n", "ID", "ROL", "ACTIVO", "NOMBRE", "CUANDO USAR")
		for _, s := range skills {
			fmt.Printf("%-5d %-15s %-8s %-24s %s\n", s.ID, s.TipoAgente, siNo(s.Activa), s.Nombre, truncarTexto(s.CuandoUsar, 48))
		}
		return nil
	},
}

var skillsCrearCmd = &cobra.Command{
	Use:   "crear",
	Short: "Crea un nuevo skill",
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		rol, _ := cmd.Flags().GetString("rol")
		nombre, _ := cmd.Flags().GetString("nombre")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		cuandoUsar, _ := cmd.Flags().GetString("cuando-usar")

		id, err := db.CrearSkill(actor, &db.Skill{
			TipoAgente:  strings.TrimSpace(rol),
			Nombre:      strings.TrimSpace(nombre),
			Descripcion: strings.TrimSpace(descripcion),
			CuandoUsar:  strings.TrimSpace(cuandoUsar),
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Skill #%d creado\n", id)
		return nil
	},
}

var skillsEditarCmd = &cobra.Command{
	Use:   "editar <id>",
	Short: "Edita un skill existente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		s, err := db.GetSkill(id)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("rol") {
			rol, _ := cmd.Flags().GetString("rol")
			s.TipoAgente = strings.TrimSpace(rol)
		}
		if cmd.Flags().Changed("nombre") {
			nombre, _ := cmd.Flags().GetString("nombre")
			s.Nombre = strings.TrimSpace(nombre)
		}
		if cmd.Flags().Changed("descripcion") {
			descripcion, _ := cmd.Flags().GetString("descripcion")
			s.Descripcion = strings.TrimSpace(descripcion)
		}
		if cmd.Flags().Changed("cuando-usar") {
			cuandoUsar, _ := cmd.Flags().GetString("cuando-usar")
			s.CuandoUsar = strings.TrimSpace(cuandoUsar)
		}
		if err := db.ActualizarSkill(actor, s); err != nil {
			return err
		}
		fmt.Printf("✓ Skill #%d actualizado\n", id)
		return nil
	},
}

var skillsActivarCmd = activarSkillCmd(true)
var skillsDesactivarCmd = activarSkillCmd(false)
var skillsVersionesCmd = &cobra.Command{
	Use:   "versiones <id>",
	Short: "Lista el historial de versiones de un skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		versiones, err := db.ListarVersionesSkill(id)
		if err != nil {
			return err
		}
		if len(versiones) == 0 {
			fmt.Println("No hay versiones registradas para ese skill.")
			return nil
		}
		fmt.Printf("%-5s %-8s %-15s %-12s %-24s %s\n", "VER", "ACTIVO", "ACTOR", "ACCION", "ROL", "NOMBRE")
		for _, v := range versiones {
			fmt.Printf("%-5d %-8s %-15s %-12s %-24s %s\n", v.VersionNum, siNo(v.Activa), v.Actor, v.Accion, v.TipoAgente, v.Nombre)
		}
		return nil
	},
}

var workflowsCmd = &cobra.Command{
	Use:   "workflows",
	Short: "Gestión CLI de workflows",
}

var workflowsListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista workflows por rol o de todos los roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		rol, _ := cmd.Flags().GetString("rol")
		todos, _ := cmd.Flags().GetBool("todos")
		var activo *bool
		if !todos {
			activo = ptrBool(true)
		}
		workflows, err := db.ListarWorkflows(strings.TrimSpace(rol), activo)
		if err != nil {
			return err
		}
		if len(workflows) == 0 {
			fmt.Println("No hay workflows con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-15s %-8s %-24s %s\n", "ID", "ROL", "ACTIVO", "NOMBRE", "PASOS")
		for _, w := range workflows {
			fmt.Printf("%-5d %-15s %-8s %-24s %d\n", w.ID, w.TipoAgente, siNo(w.Activo), w.Nombre, contarPasos(w.Pasos))
		}
		return nil
	},
}

var workflowsVerCmd = &cobra.Command{
	Use:   "ver <id>",
	Short: "Muestra el detalle de un workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		w, err := db.GetWorkflowByID(id)
		if err != nil {
			return err
		}
		fmt.Printf("Workflow #%d — %s\n", w.ID, w.Nombre)
		fmt.Printf("  Rol:         %s\n", w.TipoAgente)
		fmt.Printf("  Activo:      %s\n", siNo(w.Activo))
		if w.Descripcion != "" {
			fmt.Printf("  Descripción: %s\n", w.Descripcion)
		}
		fmt.Printf("  Pasos:\n")
		for _, paso := range decodificarPasos(w.Pasos) {
			fmt.Printf("    %s\n", paso)
		}
		return nil
	},
}

var workflowsCrearCmd = &cobra.Command{
	Use:   "crear",
	Short: "Crea un nuevo workflow",
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		rol, _ := cmd.Flags().GetString("rol")
		nombre, _ := cmd.Flags().GetString("nombre")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		pasosJSON, _, err := pasosWorkflowDesdeFlags(cmd)
		if err != nil {
			return err
		}
		id, err := db.CrearWorkflow(actor, &db.Workflow{
			TipoAgente:  strings.TrimSpace(rol),
			Nombre:      strings.TrimSpace(nombre),
			Descripcion: strings.TrimSpace(descripcion),
			Pasos:       pasosJSON,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Workflow #%d creado\n", id)
		return nil
	},
}

var workflowsEditarCmd = &cobra.Command{
	Use:   "editar <id>",
	Short: "Edita un workflow existente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		w, err := db.GetWorkflowByID(id)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("rol") {
			rol, _ := cmd.Flags().GetString("rol")
			w.TipoAgente = strings.TrimSpace(rol)
		}
		if cmd.Flags().Changed("nombre") {
			nombre, _ := cmd.Flags().GetString("nombre")
			w.Nombre = strings.TrimSpace(nombre)
		}
		if cmd.Flags().Changed("descripcion") {
			descripcion, _ := cmd.Flags().GetString("descripcion")
			w.Descripcion = strings.TrimSpace(descripcion)
		}
		if pasosJSON, reemplazar, err := pasosWorkflowDesdeFlags(cmd); err != nil {
			return err
		} else if reemplazar {
			w.Pasos = pasosJSON
		}
		if err := db.ActualizarWorkflow(actor, w); err != nil {
			return err
		}
		fmt.Printf("✓ Workflow #%d actualizado\n", id)
		return nil
	},
}

var workflowsActivarCmd = activarWorkflowCmd(true)
var workflowsDesactivarCmd = activarWorkflowCmd(false)
var workflowsVersionesCmd = &cobra.Command{
	Use:   "versiones <id>",
	Short: "Lista el historial de versiones de un workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		versiones, err := db.ListarVersionesWorkflow(id)
		if err != nil {
			return err
		}
		if len(versiones) == 0 {
			fmt.Println("No hay versiones registradas para ese workflow.")
			return nil
		}
		fmt.Printf("%-5s %-8s %-15s %-12s %-24s %s\n", "VER", "ACTIVO", "ACTOR", "ACCION", "ROL", "NOMBRE")
		for _, v := range versiones {
			fmt.Printf("%-5d %-8s %-15s %-12s %-24s %s\n", v.VersionNum, siNo(v.Activo), v.Actor, v.Accion, v.TipoAgente, v.Nombre)
		}
		return nil
	},
}

var permisosCmd = &cobra.Command{
	Use:   "permisos",
	Short: "Muestra la matriz de permisos de edición del catálogo",
}

var permisosListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista permisos por entidad",
	RunE: func(cmd *cobra.Command, args []string) error {
		entidad, _ := cmd.Flags().GetString("entidad")
		permisos, err := db.ListarPermisosEdicionCatalogo(strings.TrimSpace(entidad))
		if err != nil {
			return err
		}
		if len(permisos) == 0 {
			fmt.Println("No hay permisos de edición registrados.")
			return nil
		}
		fmt.Printf("%-12s %-15s %-12s %-8s %-8s %-8s %-12s\n", "ENTIDAD", "ROL", "ALCANCE", "CREAR", "EDITAR", "ACTIVAR", "VERSIONAR")
		for _, p := range permisos {
			fmt.Printf("%-12s %-15s %-12s %-8s %-8s %-8s %-12s\n", p.Entidad, p.Rol, p.Alcance, siNo(p.PuedeCrear), siNo(p.PuedeEditar), siNo(p.PuedeActivar), siNo(p.PuedeVersionar))
		}
		return nil
	},
}

var permisosFijarCmd = &cobra.Command{
	Use:   "fijar",
	Short: "Define o actualiza permisos de edición del catálogo",
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := actorCatalogo(cmd)
		if err != nil {
			return err
		}
		entidad, _ := cmd.Flags().GetString("entidad")
		rol, _ := cmd.Flags().GetString("rol")
		alcance, _ := cmd.Flags().GetString("alcance")
		permiso := &db.PermisoEdicionCatalogo{
			Entidad:        strings.TrimSpace(entidad),
			Rol:            strings.TrimSpace(rol),
			Alcance:        strings.TrimSpace(alcance),
			PuedeCrear:     flagBool(cmd, "crear"),
			PuedeEditar:    flagBool(cmd, "editar"),
			PuedeActivar:   flagBool(cmd, "activar"),
			PuedeVersionar: flagBool(cmd, "versionar"),
		}
		if err := db.GuardarPermisoEdicionCatalogo(actor, permiso); err != nil {
			return err
		}
		fmt.Printf("✓ Permiso actualizado: %s/%s\n", permiso.Entidad, permiso.Rol)
		return nil
	},
}

func init() {
	reglasListarCmd.Flags().String("rol", "", "Filtrar por rol")
	reglasListarCmd.Flags().Bool("todas", false, "Incluir reglas inactivas")
	reglasCrearCmd.Flags().String("rol", "", "Rol objetivo: programador, documentador, admin")
	reglasCrearCmd.Flags().String("categoria", "", "Categoría de la regla")
	reglasCrearCmd.Flags().String("titulo", "", "Título de la regla")
	reglasCrearCmd.Flags().String("descripcion", "", "Descripción de la regla")
	reglasCrearCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	reglasEditarCmd.Flags().String("rol", "", "Nuevo rol")
	reglasEditarCmd.Flags().String("categoria", "", "Nueva categoría")
	reglasEditarCmd.Flags().String("titulo", "", "Nuevo título")
	reglasEditarCmd.Flags().String("descripcion", "", "Nueva descripción")
	reglasEditarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	reglasActivarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	reglasDesactivarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	reglasCmd.AddCommand(reglasListarCmd, reglasCrearCmd, reglasEditarCmd, reglasActivarCmd, reglasDesactivarCmd, reglasVersionesCmd)

	skillsListarCmd.Flags().String("rol", "", "Filtrar por rol")
	skillsListarCmd.Flags().Bool("todos", false, "Incluir skills inactivos")
	skillsCrearCmd.Flags().String("rol", "", "Rol objetivo: programador, documentador, admin")
	skillsCrearCmd.Flags().String("nombre", "", "Nombre del skill")
	skillsCrearCmd.Flags().String("descripcion", "", "Descripción del skill")
	skillsCrearCmd.Flags().String("cuando-usar", "", "Cuándo usar el skill")
	skillsCrearCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	skillsEditarCmd.Flags().String("rol", "", "Nuevo rol")
	skillsEditarCmd.Flags().String("nombre", "", "Nuevo nombre")
	skillsEditarCmd.Flags().String("descripcion", "", "Nueva descripción")
	skillsEditarCmd.Flags().String("cuando-usar", "", "Nuevo texto de cuándo usar")
	skillsEditarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	skillsActivarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	skillsDesactivarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	skillsCmd.AddCommand(skillsListarCmd, skillsCrearCmd, skillsEditarCmd, skillsActivarCmd, skillsDesactivarCmd, skillsVersionesCmd)

	workflowsListarCmd.Flags().String("rol", "", "Filtrar por rol")
	workflowsListarCmd.Flags().Bool("todos", false, "Incluir workflows inactivos")
	workflowsCrearCmd.Flags().String("rol", "", "Rol objetivo: programador, documentador, admin")
	workflowsCrearCmd.Flags().String("nombre", "", "Nombre del workflow")
	workflowsCrearCmd.Flags().String("descripcion", "", "Descripción del workflow")
	workflowsCrearCmd.Flags().StringArray("paso", nil, "Paso del workflow; repetir para varios")
	workflowsCrearCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	workflowsEditarCmd.Flags().String("rol", "", "Nuevo rol")
	workflowsEditarCmd.Flags().String("nombre", "", "Nuevo nombre")
	workflowsEditarCmd.Flags().String("descripcion", "", "Nueva descripción")
	workflowsEditarCmd.Flags().StringArray("paso", nil, "Reemplaza todos los pasos; repetir para varios")
	workflowsEditarCmd.Flags().Bool("vaciar-pasos", false, "Deja el workflow sin pasos")
	workflowsEditarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	workflowsActivarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	workflowsDesactivarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	workflowsCmd.AddCommand(workflowsListarCmd, workflowsVerCmd, workflowsCrearCmd, workflowsEditarCmd, workflowsActivarCmd, workflowsDesactivarCmd, workflowsVersionesCmd)
	permisosListarCmd.Flags().String("entidad", "", "Filtrar por entidad")
	permisosFijarCmd.Flags().String("entidad", "", "Entidad: reglas, skills o workflows")
	permisosFijarCmd.Flags().String("rol", "", "Rol: programador, documentador o admin")
	permisosFijarCmd.Flags().String("alcance", "mismo_rol", "Alcance: mismo_rol o todos")
	permisosFijarCmd.Flags().Bool("crear", false, "Permite crear")
	permisosFijarCmd.Flags().Bool("editar", false, "Permite editar")
	permisosFijarCmd.Flags().Bool("activar", false, "Permite activar/desactivar")
	permisosFijarCmd.Flags().Bool("versionar", false, "Permite ver y registrar versiones")
	permisosFijarCmd.Flags().String("por", "alberto", "Actor que ejecuta la operación")
	permisosCmd.AddCommand(permisosListarCmd)
	permisosCmd.AddCommand(permisosFijarCmd)

	rootCmd.AddCommand(reglasCmd, skillsCmd, workflowsCmd, permisosCmd)
}

func activarReglaCmd(activa bool) *cobra.Command {
	uso := "activar"
	short := "Activa una regla"
	if !activa {
		uso = "desactivar"
		short = "Desactiva una regla"
	}
	return &cobra.Command{
		Use:   uso + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actor, err := resolverValorFlag(cmd, "agente", "por")
			if err != nil {
				return err
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("id inválido")
			}
			if err := db.SetReglaActiva(actor, id, activa); err != nil {
				return err
			}
			fmt.Printf("✓ Regla #%d %s\n", id, estadoVerbo(activa))
			return nil
		},
	}
}

func activarSkillCmd(activo bool) *cobra.Command {
	uso := "activar"
	short := "Activa un skill"
	if !activo {
		uso = "desactivar"
		short = "Desactiva un skill"
	}
	return &cobra.Command{
		Use:   uso + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actor, err := resolverValorFlag(cmd, "agente", "por")
			if err != nil {
				return err
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("id inválido")
			}
			if err := db.SetSkillActivo(actor, id, activo); err != nil {
				return err
			}
			fmt.Printf("✓ Skill #%d %s\n", id, estadoVerbo(activo))
			return nil
		},
	}
}

func activarWorkflowCmd(activo bool) *cobra.Command {
	uso := "activar"
	short := "Activa un workflow"
	if !activo {
		uso = "desactivar"
		short = "Desactiva un workflow"
	}
	return &cobra.Command{
		Use:   uso + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actor, err := resolverValorFlag(cmd, "agente", "por")
			if err != nil {
				return err
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("id inválido")
			}
			if err := db.SetWorkflowActivo(actor, id, activo); err != nil {
				return err
			}
			fmt.Printf("✓ Workflow #%d %s\n", id, estadoVerbo(activo))
			return nil
		},
	}
}

func siNo(v bool) string {
	if v {
		return "sí"
	}
	return "no"
}

func estadoVerbo(v bool) string {
	if v {
		return "activado"
	}
	return "desactivado"
}

func truncarTexto(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func pasosWorkflowDesdeFlags(cmd *cobra.Command) (string, bool, error) {
	if vaciar, _ := cmd.Flags().GetBool("vaciar-pasos"); vaciar {
		return "[]", true, nil
	}
	pasos, _ := cmd.Flags().GetStringArray("paso")
	if len(pasos) == 0 {
		return "", false, nil
	}
	for i, paso := range pasos {
		pasos[i] = strings.TrimSpace(paso)
	}
	raw, err := json.Marshal(pasos)
	if err != nil {
		return "", false, err
	}
	return string(raw), true, nil
}

func decodificarPasos(raw string) []string {
	var pasos []string
	if err := json.Unmarshal([]byte(raw), &pasos); err != nil || len(pasos) == 0 {
		return []string{"(sin pasos o formato no legible)"}
	}
	return pasos
}

func contarPasos(raw string) int {
	var pasos []string
	if err := json.Unmarshal([]byte(raw), &pasos); err != nil {
		return 0
	}
	return len(pasos)
}

func actorCatalogo(cmd *cobra.Command) (string, error) {
	if cmd == nil {
		return "", fmt.Errorf("comando no disponible")
	}
	for _, nombre := range []string{"por", "agente"} {
		if flag := cmd.Flags().Lookup(nombre); flag != nil {
			valor, _ := cmd.Flags().GetString(nombre)
			valor = strings.TrimSpace(valor)
			if valor != "" {
				return valor, nil
			}
		}
	}
	return "alberto", nil
}

func flagBool(cmd *cobra.Command, nombre string) bool {
	if cmd == nil {
		return false
	}
	v, _ := cmd.Flags().GetBool(nombre)
	return v
}
