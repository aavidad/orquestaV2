/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import "fmt"

const defaultServePort = 16543

var defaultServerURL = fmt.Sprintf("http://127.0.0.1:%d", defaultServePort)
