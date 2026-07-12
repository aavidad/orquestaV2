package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// Smoke real de la capacidad de extraccion documental: servidor arrancado con la
// composicion canonica, PDF de verdad en el inbox de ingesta, y llamada por
// JSON-RPC a POST /mcp. Nada de fakes: si el adaptador no lee el PDF, esto se
// pone rojo. Un test con fake no acredita una capacidad.
func TestMCPDocumentTextExtractLeeUnPDFRealPorTransporteV0(t *testing.T) {
	stack := buildCanonicalMCPBootstrapStackForTestV0(t)
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)

	documentRef := ponerPDFEnElInboxParaTestV0(t, os.Getenv(envServerStateDirV0))

	var raw json.RawMessage
	if err := callMCPJSONRPCBootstrapV0(server.URL, "tools/call", map[string]any{
		"name": orquestamcp.MCPDocumentTextExtractToolNameV0,
		"arguments": map[string]any{
			"document_ref": documentRef,
			"page_from":    1,
			"page_limit":   2,
		},
	}, &raw); err != nil {
		t.Fatalf("tools/call %s: %v", orquestamcp.MCPDocumentTextExtractToolNameV0, err)
	}

	payload := string(raw)
	if reason := bootstrapMissingPortReasonV0(payload); reason != "" {
		t.Fatalf("la tool respondio con puerto sin cablear (%s): %s", reason, payload)
	}

	result := decodeDocumentTextExtractResultV0(t, raw)
	if result.Estado != orquestamcp.MCPDocumentTextExtractEstadoOKV0 {
		t.Fatalf("estado = %q, errores = %+v", result.Estado, result.ErroresPublicos)
	}
	if result.PageCount == 0 || len(result.Pages) == 0 {
		t.Fatalf("la tool no devolvio paginas: %+v", result)
	}
	if result.SpanCount == 0 {
		t.Fatal("la tool no devolvio ni una linea de texto")
	}
	if result.AdapterRef == "" || result.ContentHash == "" {
		t.Fatalf("resultado sin procedencia verificable: adapter=%q hash=%q", result.AdapterRef, result.ContentHash)
	}

	texto := strings.Join(result.Pages[0].Lines, "\n")
	for _, esperado := range []string{"RESOLUCION DE PROCESO SELECTIVO", "2025/PPT_01/000087"} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("el texto devuelto por MCP no contiene %q: la tool no leyo el PDF de verdad", esperado)
		}
	}

	// El confinamiento a la raiz de ingesta tiene que sostenerse tambien a traves
	// del transporte: una ruta absoluta del host no puede leerse por MCP.
	var escape json.RawMessage
	if err := callMCPJSONRPCBootstrapV0(server.URL, "tools/call", map[string]any{
		"name":      orquestamcp.MCPDocumentTextExtractToolNameV0,
		"arguments": map[string]any{"document_ref": "/etc/passwd"},
	}, &escape); err != nil {
		t.Fatalf("tools/call escape: %v", err)
	}
	escapeResult := decodeDocumentTextExtractResultV0(t, escape)
	if escapeResult.Estado != orquestamcp.MCPDocumentTextExtractEstadoErrorV0 {
		t.Fatalf("una ruta absoluta del host debe rechazarse por MCP: %+v", escapeResult)
	}
}

func decodeDocumentTextExtractResultV0(
	t *testing.T,
	raw json.RawMessage,
) orquestamcp.MCPDocumentTextExtractToolResultV0 {
	t.Helper()
	var envelope struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Content) != 1 {
		t.Fatalf("envelope inesperado: %v raw=%s", err, string(raw))
	}
	var result orquestamcp.MCPDocumentTextExtractToolResultV0
	if err := json.Unmarshal([]byte(envelope.Content[0].Text), &result); err != nil {
		t.Fatalf("decodificando payload: %v text=%s", err, envelope.Content[0].Text)
	}
	return result
}

func ponerPDFEnElInboxParaTestV0(t *testing.T, stateDir string) string {
	t.Helper()
	origen := filepath.Join(
		"..", "..", "modulos", "orquesta-document-extraction-pdf", "testdata",
		"proceso_selectivo_fixture_v0.pdf",
	)
	bytes, err := os.ReadFile(origen)
	if err != nil {
		t.Fatalf("leyendo la fixtura PDF: %v", err)
	}
	inbox := filepath.Join(stateDir, "document-inbox")
	if err := os.MkdirAll(inbox, 0o700); err != nil {
		t.Fatalf("creando el inbox: %v", err)
	}
	destino := filepath.Join(inbox, "proceso_selectivo_v0.pdf")
	if err := os.WriteFile(destino, bytes, 0o600); err != nil {
		t.Fatalf("copiando el PDF al inbox: %v", err)
	}
	return "proceso_selectivo_v0.pdf"
}
