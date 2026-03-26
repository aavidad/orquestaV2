/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import "fmt"

func serverFirstCommandError(nombre string) error {
	return fmt.Errorf("%s requiere el servidor de Orquesta activo; arranca 'orquesta serve'. Usa ORQUESTA_FORCE_LOCAL_DB=1 solo en recuperación explícita", nombre)
}
