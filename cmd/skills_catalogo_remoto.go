package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"orquesta/db"
)

var skillsRemotasCmd = &cobra.Command{
	Use:   "remotas",
	Short: "Lista skills disponibles en skills.sh",
	RunE: func(cmd *cobra.Command, args []string) error {
		filtro, _ := cmd.Flags().GetString("q")
		limite, _ := cmd.Flags().GetInt("limit")
		items, ok, err := listarSkillsRemotasPorAPI(filtro, limite)
		if !ok {
			items, err = skillsCatalogoFetcher.Listar(context.Background(), filtro, limite)
		}
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No se encontraron skills remotas.")
			return nil
		}
		fmt.Printf("%-24s %-28s %s\n", "SKILL", "REPO", "URL")
		for _, item := range items {
			fmt.Printf("%-24s %-28s %s\n", item.Skill, item.Repo, item.URLCanonica)
		}
		return nil
	},
}

var skillsBorrarCmd = &cobra.Command{
	Use:   "borrar <id>",
	Short: "Borra una skill del catálogo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, err := resolverValorFlag(cmd, "agente", "por")
		if err != nil {
			return err
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		if ok, err := borrarSkillPorAPI(id, actor); err != nil {
			return err
		} else if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			if err := db.EliminarSkill(strings.TrimSpace(actor), id); err != nil {
				return err
			}
		}
		fmt.Printf("✓ Skill #%d borrada\n", id)
		return nil
	},
}
