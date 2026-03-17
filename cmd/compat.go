/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func resolverValorFlag(cmd *cobra.Command, nombres ...string) (string, error) {
	for _, nombre := range nombres {
		valor, err := cmd.Flags().GetString(nombre)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(valor) != "" {
			return strings.TrimSpace(valor), nil
		}
	}
	return "", nil
}

func resolverTextoFlagOPosicional(cmd *cobra.Command, args []string, desde int, flag string) (string, error) {
	if len(args) > desde {
		return strings.TrimSpace(strings.Join(args[desde:], " ")), nil
	}
	valor, err := cmd.Flags().GetString(flag)
	if err != nil {
		return "", err
	}
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return "", fmt.Errorf("debe indicar el texto posicional o --%s", flag)
	}
	return valor, nil
}
