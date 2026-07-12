package orquestadocumentextractionpdf_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	extraction "orquesta/modulos/orquesta-document-extraction"
	pdf "orquesta/modulos/orquesta-document-extraction-pdf"
)

// PDF real de un proceso selectivo: resolucion de la Diputacion de Granada con
// la lista definitiva de admitidos y excluidos. No es un fixture sintetico: es
// el documento que el operador necesita cargar. Un fake no acredita esta
// capacidad.
const pdfRealProcesoSelectivoV0 = "/home/alberto/Trabajo/Baremador_windows/DOC-20260519-WA0032..pdf"

func TestAdapterV0ExtraeTextoYAnclasDeUnPDFRealV0(t *testing.T) {
	requerirPDFRealV0(t)

	adapter := pdf.NewAdapterV0()
	ctx := context.Background()

	source, err := adapter.ResolveDocumentV0(ctx, pdfRealProcesoSelectivoV0)
	if err != nil {
		t.Fatalf("ResolveDocumentV0: %v", err)
	}
	if source.MediaKind != pdf.MediaKindV0 {
		t.Fatalf("media kind = %q, quiero %q", source.MediaKind, pdf.MediaKindV0)
	}
	if !strings.HasPrefix(source.ContentHash, "sha256:") {
		t.Fatalf("content hash sin algoritmo: %q", source.ContentHash)
	}

	normalized, err := adapter.NormalizeDocumentV0(ctx, source, extraction.DefaultDocumentExtractionPolicyV0())
	if err != nil {
		t.Fatalf("NormalizeDocumentV0: %v", err)
	}

	document, err := adapter.ParseDocumentV0(ctx, normalized)
	if err != nil {
		t.Fatalf("ParseDocumentV0: %v", err)
	}

	if document.IRVersion != extraction.DocumentExtractionIRSchemaVersionV0 {
		t.Fatalf("IR version = %q", document.IRVersion)
	}
	if document.PageCount != len(document.Pages) || document.PageCount == 0 {
		t.Fatalf("page count = %d, paginas = %d", document.PageCount, len(document.Pages))
	}
	if document.ContentHash != source.ContentHash {
		t.Fatalf("el hash del documento no conserva el de la fuente")
	}

	texto := textoCompletoV0(document)
	for _, esperado := range []string{
		"Diputación de Granada",
		"2025/PPT_01/000087",
		"lista definitiva",
	} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("el texto extraido no contiene %q; la extraccion no leyo el PDF de verdad", esperado)
		}
	}
}

// El nucleo rechaza toda evidencia sin pagina y sin ancla espacial. Si el
// adaptador devolviera spans sin bbox, la capacidad seria inutil aunque el
// texto saliera bien.
func TestAdapterV0DaAnclaEspacialACadaSpanV0(t *testing.T) {
	requerirPDFRealV0(t)

	adapter := pdf.NewAdapterV0()
	ctx := context.Background()
	source, err := adapter.ResolveDocumentV0(ctx, pdfRealProcesoSelectivoV0)
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

func TestAdapterV0FallaTipadoSiElDocumentoNoExisteV0(t *testing.T) {
	adapter := pdf.NewAdapterV0()
	if _, err := adapter.ResolveDocumentV0(context.Background(), "/no/existe.pdf"); err == nil {
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

func requerirPDFRealV0(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext (poppler) no disponible")
	}
	if _, err := os.Stat(pdfRealProcesoSelectivoV0); err != nil {
		t.Skip("PDF real del proceso selectivo no disponible")
	}
}
