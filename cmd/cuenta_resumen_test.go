package cmd

import "testing"

func TestResumenCuentaCanonicaPrefiereIdentidadCanonica(t *testing.T) {
	got := resumenCuentaCanonica("acc-codex-12345678", "shared@example.com", "Codex7")
	want := "shared@example.com (Codex7) [acc-code]"
	if got != want {
		t.Fatalf("resumen inesperado: got=%q want=%q", got, want)
	}
}

func TestResumenCuentaCanonicaSinCorreoUsaAccountID(t *testing.T) {
	got := resumenCuentaCanonica("acc-codex-1234", "", "")
	if got != "cuenta acc-codex-1234" {
		t.Fatalf("resumen inesperado: %q", got)
	}
}
