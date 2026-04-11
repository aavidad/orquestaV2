/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package main

import (
	"os"

	"orquesta/cmd"
	"orquesta/internal/controlruntime"
)

func main() {
	if controlruntime.MaybeRunEmbeddedBroker(os.Args[1:]) {
		return
	}
	if controlruntime.MaybeRunEmbeddedTmuxMonitor(os.Args[1:]) {
		return
	}
	cmd.Execute()
}
