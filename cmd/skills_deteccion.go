package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var skillsDetectarCarenciaCmd = &cobra.Command{
	Use:   "detectar-carencia",
	Short: "Detecta si falta una skill y prepara un borrador revisable",
	RunE: func(cmd *cobra.Command, args []string) error {
		rol, _ := cmd.Flags().GetString("rol")
		nombre, _ := cmd.Flags().GetString("nombre")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		cuandoUsar, _ := cmd.Flags().GetString("cuando-usar")
		escenario, _ := cmd.Flags().GetString("escenario")
		aliasesJSON, _, err := listaJSONDesdeFlags(cmd, "alias", "")
		if err != nil {
			return err
		}
		herramientasJSON, _, err := listaJSONDesdeFlags(cmd, "herramienta", "")
		if err != nil {
			return err
		}
		req := apiSkillDeteccionRequest{
			TipoAgente:       strings.TrimSpace(rol),
			Nombre:           strings.TrimSpace(nombre),
			Descripcion:      strings.TrimSpace(descripcion),
			CuandoUsar:       strings.TrimSpace(cuandoUsar),
			Escenario:        strings.TrimSpace(escenario),
			AliasesJSON:      aliasesJSON,
			HerramientasJSON: herramientasJSON,
		}
		if resp, ok, err := detectarCarenciaSkillPorAPI(req); err != nil {
			return err
		} else if ok {
			return emitirDeteccionSkill(cmd, resp.Resultado)
		}
		return serverFirstCommandError("skills detectar-carencia")
	},
}

func emitirDeteccionSkill(cmd *cobra.Command, resultado any) error {
	if resultado == nil {
		return fmt.Errorf("resultado de deteccion vacio")
	}
	data, err := json.MarshalIndent(resultado, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}

func init() {
	skillsDetectarCarenciaCmd.Flags().String("rol", "", "Rol objetivo")
	skillsDetectarCarenciaCmd.Flags().String("nombre", "", "Nombre tentativo de la skill")
	skillsDetectarCarenciaCmd.Flags().String("descripcion", "", "Descripcion funcional")
	skillsDetectarCarenciaCmd.Flags().String("cuando-usar", "", "Criterio funcional de activacion")
	skillsDetectarCarenciaCmd.Flags().String("escenario", "", "Escenario principal")
	skillsDetectarCarenciaCmd.Flags().StringArray("alias", nil, "Alias funcional; repetir para varios")
	skillsDetectarCarenciaCmd.Flags().StringArray("herramienta", nil, "Herramienta asociada; repetir para varias")
	skillsCmd.AddCommand(skillsDetectarCarenciaCmd)
}
