package orquestadocumentextractionpdf_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extraction "orquesta/modulos/orquesta-document-extraction"
	pdf "orquesta/modulos/orquesta-document-extraction-pdf"
)

// PDF real de un proceso selectivo, commiteado en testdata. No es un fake: lo
// genera poppler y lo lee poppler. El test NO hace Skip: un skip es un verde que
// esconde la ausencia de la capacidad, y esa es justo la trampa que perseguimos.
const (
	rootFixturesV0    = "testdata"
	pdfProcesoRefV0   = "proceso_selectivo_fixture_v0.pdf"
	expedienteFixtura = "2025/PPT_01/000087"
)

func adaptadorFixturaV0(t *testing.T) *pdf.AdapterV0 {
	t.Helper()
	adapter, err := pdf.NewAdapterV0(pdf.ConfigV0{RootDir: rootFixturesV0})
	if err != nil {
		t.Fatalf("NewAdapterV0: %v", err)
	}
	return adapter
}

func documentoFixturaV0(t *testing.T, adapter *pdf.AdapterV0) extraction.DocumentV0 {
	t.Helper()
	ctx := context.Background()
	source, err := adapter.ResolveDocumentV0(ctx, pdfProcesoRefV0)
	if err != nil {
		t.Fatalf("ResolveDocumentV0: %v", err)
	}
	normalized, err := adapter.NormalizeDocumentV0(ctx, source, extraction.DefaultDocumentExtractionPolicyV0())
	if err != nil {
		t.Fatalf("NormalizeDocumentV0: %v", err)
	}
	document, err := adapter.ParseDocumentV0(ctx, normalized)
	if err != nil {
		t.Fatalf("ParseDocumentV0: %v", err)
	}
	return document
}

func TestAdapterV0ExtraeTextoRealDeUnPDFV0(t *testing.T) {
	adapter := adaptadorFixturaV0(t)
	document := documentoFixturaV0(t, adapter)

	if document.IRVersion != extraction.DocumentExtractionIRSchemaVersionV0 {
		t.Fatalf("IR version = %q", document.IRVersion)
	}
	if document.PageCount != len(document.Pages) || document.PageCount == 0 {
		t.Fatalf("page count = %d, paginas = %d", document.PageCount, len(document.Pages))
	}
	texto := textoCompletoV0(document)
	for _, esperado := range []string{"RESOLUCION DE PROCESO SELECTIVO", expedienteFixtura, "GARCIA LOPEZ"} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("el texto extraido no contiene %q: la extraccion no leyo el PDF de verdad", esperado)
		}
	}
}

// El nucleo rechaza toda evidencia sin pagina y sin ancla espacial. Un adaptador
// que devuelva texto sin bbox deja la capacidad inservible aunque el texto salga.
func TestAdapterV0DaAnclaEspacialACadaSpanV0(t *testing.T) {
	adapter := adaptadorFixturaV0(t)
	document := documentoFixturaV0(t, adapter)

	spans := 0
	for _, page := range document.Pages {
		if page.Width <= 0 || page.Height <= 0 {
			t.Fatalf("pagina %s sin dimensiones", page.PageRef)
		}
		for _, block := range page.Blocks {
			if block.BoundingBox == nil {
				t.Fatalf("bloque %s sin bbox", block.BlockRef)
			}
			for _, span := range block.Spans {
				if span.BoundingBox == nil {
					t.Fatalf("span %s sin bbox: el nucleo rechazaria su evidencia", span.SpanRef)
				}
				if span.Provenance.AdapterRef != pdf.AdapterRefV0 {
					t.Fatalf("span %s sin procedencia del adaptador", span.SpanRef)
				}
				spans++
			}
		}
	}
	if spans == 0 {
		t.Fatal("no se extrajo ni un span con ancla espacial")
	}
}

// El servidor corre en contenedor: una ruta absoluta del host no existe dentro y
// aceptarla abriria lectura arbitraria de ficheros. El document_ref es siempre
// relativo a la raiz de ingesta.
func TestAdapterV0ConfinaElDocumentRefALaRaizV0(t *testing.T) {
	adapter := adaptadorFixturaV0(t)
	ctx := context.Background()

	fuera := []string{
		"/etc/passwd",
		"../../etc/passwd",
		"../adapter_v0.go",
		filepath.Join("..", "..", "..", "etc", "hosts"),
	}
	for _, ref := range fuera {
		if _, err := adapter.ResolveDocumentV0(ctx, ref); err == nil {
			t.Fatalf("document_ref %q escapo de la raiz: lectura arbitraria de ficheros", ref)
		}
	}

	enlace := filepath.Join(rootFixturesV0, "escape_v0.pdf")
	if err := os.Symlink("/etc/passwd", enlace); err == nil {
		defer os.Remove(enlace)
		if _, err := adapter.ResolveDocumentV0(ctx, "escape_v0.pdf"); err == nil {
			t.Fatal("un symlink fuera de la raiz escapo el confinamiento")
		}
	}
}

func TestAdapterV0ExigeRaizYAplicaLimitesV0(t *testing.T) {
	if _, err := pdf.NewAdapterV0(pdf.ConfigV0{}); err == nil {
		t.Fatal("un adaptador sin raiz de ingesta debe fallar al construirse")
	}

	limitado, err := pdf.NewAdapterV0(pdf.ConfigV0{RootDir: rootFixturesV0, MaxBytes: 1024})
	if err != nil {
		t.Fatalf("NewAdapterV0: %v", err)
	}
	if _, err := limitado.ResolveDocumentV0(context.Background(), pdfProcesoRefV0); err == nil {
		t.Fatal("un documento por encima del limite de bytes debe rechazarse")
	}

	porPaginas, err := pdf.NewAdapterV0(pdf.ConfigV0{RootDir: rootFixturesV0, MaxPages: 0})
	if err != nil {
		t.Fatalf("NewAdapterV0: %v", err)
	}
	if _, err := porPaginas.ResolveDocumentV0(context.Background(), pdfProcesoRefV0); err != nil {
		t.Fatalf("el limite de paginas por defecto no debe rechazar la fixtura: %v", err)
	}
}

func TestAdapterV0FallaTipadoSiElDocumentoNoExisteV0(t *testing.T) {
	adapter := adaptadorFixturaV0(t)
	if _, err := adapter.ResolveDocumentV0(context.Background(), "no_existe.pdf"); err == nil {
		t.Fatal("un documento inexistente debe fallar, no devolver material vacio")
	}
	if _, err := adapter.ResolveDocumentV0(context.Background(), "  "); err == nil {
		t.Fatal("un document_ref vacio debe fallar")
	}
}

func textoCompletoV0(document extraction.DocumentV0) string {
	var builder strings.Builder
	for _, page := range document.Pages {
		for _, block := range page.Blocks {
			for _, span := range block.Spans {
				builder.WriteString(span.TextRaw)
				builder.WriteString("\n")
			}
		}
	}
	return builder.String()
}
