package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// Smoke real de la ingesta de datos: servidor con la composicion canonica, CSV de
// verdad en el inbox, y llamada por JSON-RPC a POST /mcp. Si el adaptador no lee
// el fichero, esto se pone rojo. Un fake no acredita una capacidad.
func TestMCPDataProfileLeeUnCSVRealPorTransporteV0(t *testing.T) {
	stack := buildCanonicalMCPBootstrapStackForTestV0(t)
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)

	datasetRef := ponerCSVEnElInboxParaTestV0(t, os.Getenv(envServerStateDirV0))

	catalogo := llamarDataProfileV0(t, server.URL, map[string]any{})
	if catalogo.Estado != orquestamcp.MCPDataProfileEstadoOKV0 {
		t.Fatalf("listar catalogo: estado=%q errores=%+v", catalogo.Estado, catalogo.ErroresPublicos)
	}
	if len(catalogo.Datasets) == 0 {
		t.Fatal("el catalogo no descubrio el dataset del inbox")
	}

	perfil := llamarDataProfileV0(t, server.URL, map[string]any{"dataset_ref": datasetRef})
	if perfil.Estado != orquestamcp.MCPDataProfileEstadoOKV0 {
		t.Fatalf("perfilar: estado=%q errores=%+v", perfil.Estado, perfil.ErroresPublicos)
	}
	if perfil.RowCount == 0 {
		t.Fatal("el perfil no conto ni una fila: no leyo el CSV de verdad")
	}
	if perfil.ContentHash == "" || perfil.AdapterRef == "" {
		t.Fatalf("perfil sin procedencia verificable: hash=%q adapter=%q", perfil.ContentHash, perfil.AdapterRef)
	}
	columnas := make([]string, 0, len(perfil.Columns))
	for _, column := range perfil.Columns {
		columnas = append(columnas, column.Name)
	}
	for _, esperada := range []string{"dni", "apellidos", "puntuacion"} {
		if !contieneColumnaV0(columnas, esperada) {
			t.Fatalf("el perfil no descubrio la columna %q; columnas = %v", esperada, columnas)
		}
	}

	// Un fichero fuera del inbox no es alcanzable ni nombrandolo: el adaptador
	// solo conoce datasets catalogados.
	fuera := llamarDataProfileV0(t, server.URL, map[string]any{"dataset_ref": "../../etc/passwd"})
	if fuera.Estado != orquestamcp.MCPDataProfileEstadoErrorV0 {
		t.Fatalf("una ruta fuera del inbox debe rechazarse: %+v", fuera)
	}
}

func llamarDataProfileV0(
	t *testing.T,
	baseURL string,
	arguments map[string]any,
) orquestamcp.MCPDataProfileToolResultV0 {
	t.Helper()
	var raw json.RawMessage
	if err := callMCPJSONRPCBootstrapV0(baseURL, "tools/call", map[string]any{
		"name":      orquestamcp.MCPDataProfileToolNameV0,
		"arguments": arguments,
	}, &raw); err != nil {
		t.Fatalf("tools/call %s: %v", orquestamcp.MCPDataProfileToolNameV0, err)
	}
	if reason := bootstrapMissingPortReasonV0(string(raw)); reason != "" {
		t.Fatalf("la tool respondio con puerto sin cablear (%s): %s", reason, string(raw))
	}
	var envelope struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Content) != 1 {
		t.Fatalf("envelope inesperado: %v raw=%s", err, string(raw))
	}
	var result orquestamcp.MCPDataProfileToolResultV0
	if err := json.Unmarshal([]byte(envelope.Content[0].Text), &result); err != nil {
		t.Fatalf("decodificando payload: %v text=%s", err, envelope.Content[0].Text)
	}
	return result
}

func ponerCSVEnElInboxParaTestV0(t *testing.T, stateDir string) string {
	t.Helper()
	inbox := filepath.Join(stateDir, "data-inbox")
	if err := os.MkdirAll(inbox, 0o700); err != nil {
		t.Fatalf("creando el inbox de datos: %v", err)
	}
	csv := strings.Join([]string{
		"dni,apellidos,nombre,puntuacion",
		"***1234**,GARCIA LOPEZ,ANA,8.50",
		"***5678**,MARTIN RUIZ,JUAN,7.25",
		"***9012**,SANZ DIAZ,LUIS,6.00",
	}, "\n") + "\n"
	destino := filepath.Join(inbox, "baremo_v0.csv")
	if err := os.WriteFile(destino, []byte(csv), 0o600); err != nil {
		t.Fatalf("escribiendo el CSV: %v", err)
	}
	return "baremo_v0.csv"
}

func contieneColumnaV0(columnas []string, esperada string) bool {
	for _, columna := range columnas {
		if strings.EqualFold(strings.TrimSpace(columna), esperada) {
			return true
		}
	}
	return false
}
