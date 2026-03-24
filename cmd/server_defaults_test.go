/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import "testing"

func TestDefaultServerURLDerivaDelPuertoDeServe(t *testing.T) {
	want := "http://127.0.0.1:16543"
	if defaultServerURL != want {
		t.Fatalf("defaultServerURL = %q, want %q", defaultServerURL, want)
	}
}
