package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"orquesta/db"
	"orquesta/fabricaapp"
)

type projectAppFactory interface {
	Generate(spec fabricaapp.AppSpec) (fabricaapp.GenerationResult, error)
}

var newProjectAppFactory = func() projectAppFactory {
	return fabricaapp.NewService()
}

var proyectoFabricarAppCmd = &cobra.Command{
	Use:   "fabricar-app <slug|id>",
	Short: "Genera backlog base de una app completa para un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nombre, _ := cmd.Flags().GetString("nombre")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		tipo, _ := cmd.Flags().GetString("tipo")
		frontend, _ := cmd.Flags().GetBool("frontend")
		apiEnabled, _ := cmd.Flags().GetBool("api")
		auth, _ := cmd.Flags().GetBool("auth")
		database, _ := cmd.Flags().GetBool("db")
		docker, _ := cmd.Flags().GetBool("docker")
		i18nEnabled, _ := cmd.Flags().GetBool("i18n")
		idiomasRaw, _ := cmd.Flags().GetString("idiomas")
		actor, _ := cmd.Flags().GetString("por")
		if strings.TrimSpace(tipo) == "" {
			return fmt.Errorf("--tipo es obligatorio")
		}

		if resp, ok, err := fabricarAppProyectoPorAPI(args[0], apiProyectoFabricarAppRequest{
			Nombre:      strings.TrimSpace(nombre),
			Descripcion: strings.TrimSpace(descripcion),
			Tipo:        strings.TrimSpace(tipo),
			Frontend:    frontend,
			API:         apiEnabled,
			Auth:        auth,
			Database:    database,
			Docker:      docker,
			I18n:        i18nEnabled,
			Idiomas:     splitCSV(idiomasRaw),
			Por:         strings.TrimSpace(actor),
		}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Backlog de app generado para %s (%s)\n", resp.Slug, resp.Tipo)
			fmt.Printf("  Tareas creadas: %d\n", resp.Created)
			fmt.Printf("  En backlog:     %d\n", resp.Backlog)
			fmt.Printf("  Primeras libres: %d\n", resp.Created-resp.Backlog)
			return nil
		}

		if err := ensureLocalDB(); err != nil {
			return err
		}

		proyecto, err := db.GetProyecto(args[0])
		if err != nil {
			return err
		}

		if strings.TrimSpace(nombre) == "" {
			nombre = strings.TrimSpace(proyecto.Nombre)
		}
		if strings.TrimSpace(descripcion) == "" {
			descripcion = "Backlog inicial de " + nombre + " generado por la fabrica de apps de Orquesta."
		}
		if strings.TrimSpace(actor) == "" {
			actor = "alberto"
		}

		svc := newProjectAppFactory()
		plan, err := svc.Generate(fabricaapp.AppSpec{
			Nombre:      nombre,
			Descripcion: descripcion,
			Tipo:        tipo,
			Frontend:    frontend,
			API:         apiEnabled,
			Auth:        auth,
			Database:    database,
			Docker:      docker,
			I18n:        i18nEnabled,
			Idiomas:     splitCSV(idiomasRaw),
		})
		if err != nil {
			return err
		}

		materialized, err := fabricaapp.Materialize(fabricaapp.DBStore{}, proyecto.ID, actor, plan)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Backlog de app generado para %s (%s)\n", proyecto.Slug, tipo)
		fmt.Printf("  Tareas creadas: %d\n", materialized.Created)
		fmt.Printf("  En backlog:     %d\n", materialized.Backlog)
		fmt.Printf("  Primeras libres: %d\n", materialized.Created-materialized.Backlog)
		for _, item := range plan.Tasks {
			fmt.Printf("  [%s] %s\n", item.Key, item.Titulo)
		}
		return nil
	},
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func init() {
	proyectoFabricarAppCmd.Flags().String("tipo", "", "Tipo de app: web, api, web_api o cli")
	proyectoFabricarAppCmd.Flags().String("nombre", "", "Nombre funcional de la app")
	proyectoFabricarAppCmd.Flags().String("descripcion", "", "Resumen funcional para construir el backlog")
	proyectoFabricarAppCmd.Flags().Bool("frontend", false, "Forzar tarea de frontend")
	proyectoFabricarAppCmd.Flags().Bool("api", false, "Forzar tarea de API")
	proyectoFabricarAppCmd.Flags().Bool("auth", false, "Incluir autenticacion")
	proyectoFabricarAppCmd.Flags().Bool("db", false, "Incluir persistencia/base de datos")
	proyectoFabricarAppCmd.Flags().Bool("docker", false, "Incluir dockerizacion/despliegue")
	proyectoFabricarAppCmd.Flags().Bool("i18n", true, "Incluir i18n y paquete inicial de idiomas")
	proyectoFabricarAppCmd.Flags().String("idiomas", "es,en", "Lista CSV de idiomas iniciales")
	proyectoFabricarAppCmd.Flags().String("por", "alberto", "Actor que genera el backlog")
	proyectoCmd.AddCommand(proyectoFabricarAppCmd)
}
