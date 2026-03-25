package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/skillsapp"
)

var skillsImportarCmd = &cobra.Command{
	Use:   "importar",
	Short: "Importa una skill externa desde skills.sh o GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		rol, _ := cmd.Flags().GetString("rol")
		url, _ := cmd.Flags().GetString("url")
		repo, _ := cmd.Flags().GetString("repo")
		skill, _ := cmd.Flags().GetString("skill")

		req := apiSkillImportarRequest{
			Actor:      actor,
			TipoAgente: strings.TrimSpace(rol),
			URL:        strings.TrimSpace(url),
			Repo:       strings.TrimSpace(repo),
			Skill:      strings.TrimSpace(skill),
		}
		if resp, ok, err := importarSkillPorAPI(req); err != nil {
			return err
		} else if ok {
			if resp.Existente {
				fmt.Printf("✓ Skill #%d ya existia en el catalogo\n", resp.ID)
			} else {
				fmt.Printf("✓ Skill #%d importado\n", resp.ID)
			}
			return nil
		}

		result, err := skillsImportService.ImportFromWeb(context.Background(), skillsapp.ImportInput{
			Actor:      req.Actor,
			TipoAgente: req.TipoAgente,
			URL:        req.URL,
			Repo:       req.Repo,
			Skill:      req.Skill,
		})
		if err != nil {
			return err
		}
		if result.Existente {
			fmt.Printf("✓ Skill #%d ya existia en el catalogo\n", result.ID)
		} else {
			fmt.Printf("✓ Skill #%d importado\n", result.ID)
		}
		return nil
	},
}

func init() {
	skillsImportarCmd.Flags().String("rol", "", "Rol objetivo")
	skillsImportarCmd.Flags().String("url", "", "URL de skills.sh o GitHub")
	skillsImportarCmd.Flags().String("repo", "", "Repositorio owner/repo")
	skillsImportarCmd.Flags().String("skill", "", "Nombre de la skill dentro del repositorio")
	skillsImportarCmd.Flags().String("por", "alberto", "Actor que ejecuta la importacion")
	skillsImportarCmd.Flags().String("agente", "", "Alias de --por para compatibilidad con el briefing")
	skillsCmd.AddCommand(skillsImportarCmd)
}
