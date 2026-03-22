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
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/forge"
)

var conectorCmd = &cobra.Command{
	Use:   "conector",
	Short: "Conectores desacoplados para forges externos",
}

var conectorListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista conectores forge registrados",
	RunE: func(cmd *cobra.Command, args []string) error {
		tipo, _ := cmd.Flags().GetString("tipo")
		incluirInactivos, _ := cmd.Flags().GetBool("all")
		lista, err := db.ListarConectoresForge(strings.TrimSpace(tipo), incluirInactivos)
		if err != nil {
			return err
		}
		if len(lista) == 0 {
			fmt.Println("No hay conectores registrados.")
			return nil
		}
		fmt.Printf("%-16s %-10s %-12s %-8s %-8s %s\n", "SLUG", "TIPO", "OWNER", "SCOPE", "ACTIVO", "API")
		for _, item := range lista {
			fmt.Printf("%-16s %-10s %-12s %-8s %-8t %s\n", item.Slug, item.Tipo, item.Owner, item.OwnerKind, item.Activo, item.APIBaseURL)
		}
		return nil
	},
}

var conectorGithubCmd = &cobra.Command{
	Use:   "github",
	Short: "Operaciones GitHub sobre el contrato genérico de forge",
}

var conectorGithubRegistrarCmd = &cobra.Command{
	Use:   "registrar <slug>",
	Short: "Registra o actualiza un conector GitHub",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, _ := cmd.Flags().GetString("owner")
		ownerKind, _ := cmd.Flags().GetString("owner-kind")
		apiBaseURL, _ := cmd.Flags().GetString("api-base-url")
		tokenEnv, _ := cmd.Flags().GetString("token-env")

		id, err := db.RegistrarConectorForge(&db.ConectorForge{
			Slug:       strings.TrimSpace(args[0]),
			Tipo:       "github",
			Owner:      strings.TrimSpace(owner),
			OwnerKind:  strings.TrimSpace(ownerKind),
			APIBaseURL: strings.TrimSpace(apiBaseURL),
			TokenEnv:   strings.TrimSpace(tokenEnv),
			Activo:     true,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Conector GitHub registrado (id: %d)\n", id)
		return nil
	},
}

var conectorGithubAltaSubproyectoCmd = &cobra.Command{
	Use:   "alta-subproyecto <proyecto> <repo>",
	Short: "Crea un repositorio remoto en GitHub y registra el subproyecto",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, _ := cmd.Flags().GetString("conector")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		visibility, _ := cmd.Flags().GetString("visibility")
		homepage, _ := cmd.Flags().GetString("homepage")
		registradoPor, _ := cmd.Flags().GetString("por")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		conector, err := db.GetConectorForge(strings.TrimSpace(slug))
		if err != nil {
			return err
		}
		if !conector.Activo {
			return fmt.Errorf("el conector %s no está activo", conector.Slug)
		}
		if conector.Tipo != "github" {
			return fmt.Errorf("el conector %s no es de tipo github", conector.Slug)
		}

		proyecto := strings.TrimSpace(args[0])
		repo := strings.TrimSpace(args[1])
		descripcion = strings.TrimSpace(descripcion)
		visibility = strings.TrimSpace(visibility)
		homepage = strings.TrimSpace(homepage)
		registradoPor = strings.TrimSpace(registradoPor)
		if visibility == "" {
			visibility = "private"
		}
		if registradoPor == "" {
			registradoPor = "orquesta"
		}

		if dryRun {
			fmt.Printf("~ Dry run GitHub: %s/%s via %s\n", conector.Owner, repo, conector.Slug)
			fmt.Printf("  proyecto=%s visibility=%s\n", proyecto, visibility)
			return nil
		}

		token := ""
		if conector.TokenEnv != "" {
			token = strings.TrimSpace(os.Getenv(conector.TokenEnv))
		}
		info, err := (&forge.GitHubClient{}).CreateRepo(forge.GitHubConfig{
			APIBaseURL: conector.APIBaseURL,
			Owner:      conector.Owner,
			OwnerKind:  conector.OwnerKind,
			Token:      token,
		}, forge.GitHubCreateRepoRequest{
			Name:        repo,
			Description: descripcion,
			Visibility:  visibility,
			Homepage:    homepage,
		})
		if err != nil {
			return err
		}

		id, err := db.RegistrarSubproyectoRemoto(&db.SubproyectoRemoto{
			Proyecto:      proyecto,
			ConectorSlug:  conector.Slug,
			ForgeTipo:     "github",
			Owner:         info.Owner,
			RepoName:      info.Name,
			RepoFullName:  info.FullName,
			Visibility:    info.Visibility,
			Descripcion:   descripcion,
			HTMLURL:       info.HTMLURL,
			CloneURL:      info.CloneURL,
			SSHURL:        info.SSHURL,
			DefaultBranch: info.DefaultBranch,
			Estado:        "creado",
			MetadataJSON:  info.RawJSON,
			RegistradoPor: registradoPor,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Subproyecto remoto registrado (id: %d)\n", id)
		fmt.Printf("  %s -> %s\n", proyecto, info.FullName)
		fmt.Printf("  %s\n", info.HTMLURL)
		return nil
	},
}

var conectorGithubSubproyectosCmd = &cobra.Command{
	Use:   "subproyectos",
	Short: "Lista subproyectos remotos registrados",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")
		lista, err := db.ListarSubproyectosRemotos(strings.TrimSpace(proyecto))
		if err != nil {
			return err
		}
		if len(lista) == 0 {
			fmt.Println("No hay subproyectos remotos registrados.")
			return nil
		}
		fmt.Printf("%-14s %-16s %-28s %-8s %s\n", "PROYECTO", "CONECTOR", "REPO", "VISIB.", "URL")
		for _, item := range lista {
			fmt.Printf("%-14s %-16s %-28s %-8s %s\n", item.Proyecto, item.ConectorSlug, item.RepoFullName, item.Visibility, item.HTMLURL)
		}
		return nil
	},
}

func init() {
	conectorListarCmd.Flags().String("tipo", "", "Filtrar por tipo")
	conectorListarCmd.Flags().Bool("all", false, "Incluir conectores inactivos")

	conectorGithubRegistrarCmd.Flags().String("owner", "", "Usuario u organización destino")
	conectorGithubRegistrarCmd.Flags().String("owner-kind", "org", "Scope del owner: org o user")
	conectorGithubRegistrarCmd.Flags().String("api-base-url", "https://api.github.com", "Base URL de API GitHub")
	conectorGithubRegistrarCmd.Flags().String("token-env", "GITHUB_TOKEN", "Variable de entorno con el token")
	_ = conectorGithubRegistrarCmd.MarkFlagRequired("owner")

	conectorGithubAltaSubproyectoCmd.Flags().String("conector", "", "Slug del conector GitHub")
	conectorGithubAltaSubproyectoCmd.Flags().String("descripcion", "", "Descripción del repositorio")
	conectorGithubAltaSubproyectoCmd.Flags().String("visibility", "private", "Visibilidad: public o private")
	conectorGithubAltaSubproyectoCmd.Flags().String("homepage", "", "Homepage del repositorio")
	conectorGithubAltaSubproyectoCmd.Flags().String("por", "orquesta", "Agente que registra el subproyecto")
	conectorGithubAltaSubproyectoCmd.Flags().Bool("dry-run", false, "No ejecuta llamada remota; solo muestra el plan")
	_ = conectorGithubAltaSubproyectoCmd.MarkFlagRequired("conector")

	conectorGithubSubproyectosCmd.Flags().String("proyecto", "", "Filtrar por proyecto")

	conectorGithubCmd.AddCommand(conectorGithubRegistrarCmd, conectorGithubAltaSubproyectoCmd, conectorGithubSubproyectosCmd)
	conectorCmd.AddCommand(conectorListarCmd, conectorGithubCmd)
	rootCmd.AddCommand(conectorCmd)
}
