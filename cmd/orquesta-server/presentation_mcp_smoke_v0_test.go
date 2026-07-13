package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// Smoke real de la extraccion de presentaciones: un PPTX de VERDAD (un zip
// OpenXML valido, no un fake) en el inbox, leido por POST /mcp con la misma tool
// que los PDF. Las presentaciones proyectan al mismo documento intermedio, asi que
// no hacia falta superficie nueva.
func TestMCPExtraeTextoDeUnPPTXRealPorTransporteV0(t *testing.T) {
	stack := buildCanonicalMCPBootstrapStackForTestV0(t)
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)

	ref := ponerPPTXEnElInboxParaTestV0(t, os.Getenv(envServerStateDirV0))

	var raw json.RawMessage
	if err := callMCPJSONRPCBootstrapV0(server.URL, "tools/call", map[string]any{
		"name":      orquestamcp.MCPDocumentTextExtractToolNameV0,
		"arguments": map[string]any{"document_ref": ref},
	}, &raw); err != nil {
		t.Fatalf("tools/call: %v", err)
	}
	if reason := bootstrapMissingPortReasonV0(string(raw)); reason != "" {
		t.Fatalf("puerto sin cablear (%s): %s", reason, string(raw))
	}
	result := decodeDocumentTextExtractResultV0(t, raw)
	if result.Estado != orquestamcp.MCPDocumentTextExtractEstadoOKV0 {
		t.Fatalf("estado=%q errores=%+v", result.Estado, result.ErroresPublicos)
	}
	if result.PageCount == 0 {
		t.Fatal("la presentacion no devolvio ni una diapositiva")
	}

	var texto strings.Builder
	for _, page := range result.Pages {
		texto.WriteString(strings.Join(page.Lines, "\n"))
		texto.WriteString("\n")
	}
	for _, esperado := range []string{"presupuesto", "conclusiones"} {
		if !strings.Contains(texto.String(), esperado) {
			t.Fatalf("el texto extraido no contiene %q: no leyo el PPTX de verdad. Texto=%q", esperado, texto.String())
		}
	}
}

// Construye un PPTX autentico: un paquete OpenXML con sus relaciones y dos
// diapositivas. Si el adaptador no supiera abrir zips ni seguir relaciones, esto
// se pondria rojo.
func ponerPPTXEnElInboxParaTestV0(t *testing.T, stateDir string) string {
	t.Helper()
	const (
		nsPresentation = "http://schemas.openxmlformats.org/presentationml/2006/main"
		nsRelationship = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
		nsPackageRels  = "http://schemas.openxmlformats.org/package/2006/relationships"
		tipoSlide      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide"
		nsDrawing      = "http://schemas.openxmlformats.org/drawingml/2006/main"
	)
	var buffer bytes.Buffer
	z := zip.NewWriter(&buffer)
	ficheros := []struct{ nombre, contenido string }{
		{"ppt/presentation.xml", `<p:presentation xmlns:p="` + nsPresentation + `" xmlns:r="` + nsRelationship + `"><p:sldSz cx="9144000" cy="5143500"/><p:sldIdLst><p:sldId id="256" r:id="rId1"/><p:sldId id="257" r:id="rId2"/></p:sldIdLst></p:presentation>`},
		{"ppt/_rels/presentation.xml.rels", `<Relationships xmlns="` + nsPackageRels + `"><Relationship Id="rId1" Type="` + tipoSlide + `" Target="slides/slide1.xml"/><Relationship Id="rId2" Type="` + tipoSlide + `" Target="slides/slide2.xml"/></Relationships>`},
		{"ppt/slides/slide1.xml", `<p:sld xmlns:p="` + nsPresentation + `" xmlns:a="` + nsDrawing + `"><a:t>presupuesto 2026</a:t></p:sld>`},
		{"ppt/slides/slide2.xml", `<p:sld xmlns:p="` + nsPresentation + `" xmlns:a="` + nsDrawing + `"><a:t>conclusiones</a:t></p:sld>`},
	}
	for _, fichero := range ficheros {
		w, err := z.Create(fichero.nombre)
		if err != nil {
			t.Fatalf("creando %s: %v", fichero.nombre, err)
		}
		if _, err := w.Write([]byte(fichero.contenido)); err != nil {
			t.Fatalf("escribiendo %s: %v", fichero.nombre, err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatalf("cerrando el zip: %v", err)
	}

	inbox := filepath.Join(stateDir, "document-inbox")
	if err := os.MkdirAll(inbox, 0o700); err != nil {
		t.Fatalf("creando el inbox: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inbox, "informe_v0.pptx"), buffer.Bytes(), 0o600); err != nil {
		t.Fatalf("escribiendo el PPTX: %v", err)
	}
	return "informe_v0.pptx"
}
