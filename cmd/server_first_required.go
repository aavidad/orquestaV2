/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import "fmt"

func serverFirstCommandError(nombre string) error {
	return fmt.Errorf("%s requiere el servidor de Orquesta activo; arranca 'orquesta serve'. El modo local queda solo para recuperación puntual", nombre)
}
