package orquestadocumentextractioncsv_test

import (
	"context"
	"strings"
	"testing"

	orquestadocumentextraction "orquesta/modulos/orquesta-document-extraction"
	orquestadocumentextractioncsv "orquesta/modulos/orquesta-document-extraction-csv"
)

func TestCSVDocumentExporterV0IncludesEvidenceAndDoesNotReplaceRaw(t *testing.T) {
	exporter := orquestadocumentextractioncsv.CSVDocumentExporterV0{Identity: orquestadocumentextraction.DocumentAdapterIdentityV0{AdapterRef: "csv", Version: "1"}, DestinationRef: "destination:csv"}
	artifact, err := exporter.ExportDocumentFieldsV0(context.Background(), orquestadocumentextraction.DocumentExportRequestV0{
		DocumentRef: "document:1",
		Fields: []orquestadocumentextraction.DocumentFieldCandidateV0{
			{
				FieldRef:        "field:1",
				OccurrenceRef:   "occurrence:1",
				ValueRaw:        "01/02/2026",
				ValueNormalized: &orquestadocumentextraction.DocumentNormalizedValueV0{Value: "2026-02-01"},
				ValueState:      orquestadocumentextraction.DocumentValueStatePresentV0,
				EvidenceRefs:    []string{"evidence:1"},
			},
		},
	})
	if err != nil || artifact.ContentHash == "" {
		t.Fatalf("export = %#v, %v", artifact, err)
	}
	output := string(artifact.Bytes)
	if !strings.Contains(output, "value_raw") || !strings.Contains(output, "01/02/2026") || !strings.Contains(output, "2026-02-01") || !strings.Contains(output, "evidence:1") {
		t.Fatalf("csv = %q", output)
	}
}
