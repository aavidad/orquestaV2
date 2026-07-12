package orquestadocumentextractionpdf_test

import (
	"context"
	"fmt"
	"testing"

	extraction "orquesta/modulos/orquesta-document-extraction"
	pdf "orquesta/modulos/orquesta-document-extraction-pdf"
)

// Prueba de uso, no de test: imprime lo que el adaptador saca del PDF real del
// operador para que el revisor lo lea con sus ojos.
func TestEvidenciaDeUsoRealV0(t *testing.T) {
	requerirPDFRealV0(t)
	a := pdf.NewAdapterV0()
	ctx := context.Background()
	src, err := a.ResolveDocumentV0(ctx, pdfRealProcesoSelectivoV0)
	if err != nil {
		t.Fatal(err)
	}
	nm, err := a.NormalizeDocumentV0(ctx, src, extraction.DefaultDocumentExtractionPolicyV0())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := a.ParseDocumentV0(ctx, nm)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("EVIDENCIA paginas=%d hash=%s\n", doc.PageCount, src.ContentHash[:24])
	n := 0
	for _, p := range doc.Pages {
		for _, b := range p.Blocks {
			for _, s := range b.Spans {
				if n < 7 {
					fmt.Printf("EVIDENCIA [%s x=%.0f y=%.0f] %s\n", s.SpanRef, s.BoundingBox.X, s.BoundingBox.Y, s.TextRaw)
				}
				n++
			}
		}
	}
	fmt.Printf("EVIDENCIA spans con ancla espacial = %d\n", n)
}
