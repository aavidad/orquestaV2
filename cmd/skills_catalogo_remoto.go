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
)

func skillsCatalogoModoRecuperacionLocalExplicito() bool {
	return strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1"
}

func skillsCatalogoErrorServerFirst() error {
	return fmt.Errorf("este comando exige servidor/daemon de Orquesta; usa --local solo en recuperacion explicita o exporta ORQUESTA_FORCE_LOCAL_DB=1")
}

var skillsRemotasCmd = &cobra.Command{
	Use:   "remotas",
	Short: "Lista skills disponibles en skills.sh",
	RunE: func(cmd *cobra.Command, args []string) error {
		filtro, _ := cmd.Flags().GetString("q")
		limite, _ := cmd.Flags().GetInt("limit")
		items, ok, err := listarSkillsRemotasPorAPI(filtro, limite)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("skills remotas")
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
			return serverFirstCommandError("skills borrar")
		}
		fmt.Printf("✓ Skill #%d borrada\n", id)
		return nil
	},
}
