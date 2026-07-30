// Este contrato liga las revisiones bootstrap del validador a sus bytes
// exactos; no las convierte en un recibo productivo de Orquesta.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

const physicalMappingReview = "docs/reconstruccion/revision_validador_mapeo_historico_2026-07-30.md"

var forbiddenPhysicalMappingPromotions = []string{
	"GOV-16 queda acreditada.",
	"Este registro sí es un receipt productivo.",
	"este registro acredita GOV-16",
	"el mapeo queda acreditado",
	"la vista estable queda acreditada",
	"los recibos del censo quedan acreditados",
	"la compuerta queda cerrada",
}

type physicalMappingReviewRecord struct {
	Schema            string `json:"schema"`
	SubjectSHA256     string `json:"subject_sha256"`
	ProductiveReceipt bool   `json:"productive_receipt"`
	Reviews           []struct {
		Reviewer string `json:"reviewer"`
		Verdict  string `json:"verdict"`
		P0       int    `json:"p0"`
		P1       int    `json:"p1"`
		P2       int    `json:"p2"`
	} `json:"reviews"`
}

func TestPhysicalMappingIndependentReviewRecord(t *testing.T) {
	got, err := physicalMappingAggregate(physicalMappingApp)
	if err != nil {
		t.Fatal(err)
	}
	if got != physicalMappingSubject {
		t.Fatalf("sujeto agregado=%s, esperado=%s", got, physicalMappingSubject)
	}
	document := readPhysicalMappingReview(t)
	if _, err := validatePhysicalMappingReview(document); err != nil {
		t.Fatal(err)
	}
	requirePhysicalMappingCaveat(t, document)
}

func TestPhysicalMappingReviewRejectsAmbiguityAndPromotion(t *testing.T) {
	document := readPhysicalMappingReview(t)
	mutations := []struct {
		name, old, new string
	}{
		{"clave duplicada", `"schema":`, `"schema":"duplicada","schema":`},
		{"recibo productivo", `"productive_receipt":false`, `"productive_receipt":true`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(document, []byte(mutation.old), []byte(mutation.new), 1)
			if bytes.Equal(changed, document) {
				t.Fatal("la mutación no cambió el documento")
			}
			if _, err := validatePhysicalMappingReview(changed); err == nil {
				t.Fatal("la revisión ambigua o productiva fue aceptada")
			}
		})
	}
	for _, promotion := range forbiddenPhysicalMappingPromotions {
		t.Run(promotion, func(t *testing.T) {
			changed := append(append([]byte(nil), document...), []byte("\n"+promotion+"\n")...)
			if _, err := validatePhysicalMappingReview(changed); err == nil {
				t.Fatal("la promoción falsa fue aceptada")
			}
		})
	}
}

func validatePhysicalMappingReview(document []byte) (physicalMappingReviewRecord, error) {
	record, err := decodePhysicalMappingReview(document)
	if err != nil {
		return record, err
	}
	if record.Schema != "orquesta.bootstrap-independent-review-record.v1" ||
		record.SubjectSHA256 != physicalMappingSubject || record.ProductiveReceipt {
		return record, fmt.Errorf("cabecera de revisión inválida: %+v", record)
	}
	want := map[string]bool{"revisor_mapeo_semantico": false, "revisor_bloqueo_app13": false}
	if len(record.Reviews) != len(want) {
		return record, fmt.Errorf("revisiones=%d", len(record.Reviews))
	}
	for _, review := range record.Reviews {
		seen, expected := want[review.Reviewer]
		if !expected || seen || review.Verdict != "ACEPTAR" ||
			review.P0 != 0 || review.P1 != 0 || review.P2 != 0 {
			return record, fmt.Errorf("revisión inválida: %+v", review)
		}
		want[review.Reviewer] = true
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(string(document)), " "))
	for _, promotion := range forbiddenPhysicalMappingPromotions {
		if strings.Contains(normalized, strings.ToLower(promotion)) {
			return record, fmt.Errorf("promoción productiva prohibida: %q", promotion)
		}
	}
	return record, nil
}

func decodePhysicalMappingReview(document []byte) (physicalMappingReviewRecord, error) {
	var record physicalMappingReviewRecord
	const opening = "```json\n"
	if bytes.Count(document, []byte(opening)) != 1 {
		return record, errors.New("se requiere un único registro JSON")
	}
	start := bytes.Index(document, []byte(opening)) + len(opening)
	tail := document[start:]
	end := bytes.Index(tail, []byte("\n```"))
	if end < 0 {
		return record, errors.New("cierre del registro JSON ausente")
	}
	body := tail[:end]
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return record, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return record, errors.New("valores posteriores en el registro JSON")
	}
	canonical, err := json.Marshal(record)
	if err != nil {
		return record, err
	}
	if !bytes.Equal(body, canonical) {
		return record, errors.New("el registro JSON no es canónico")
	}
	return record, nil
}

func readPhysicalMappingReview(t *testing.T) []byte {
	t.Helper()
	document, err := os.ReadFile(physicalMappingReview)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func requirePhysicalMappingCaveat(t *testing.T, document []byte) {
	t.Helper()
	normalized := strings.Join(strings.Fields(string(document)), " ")
	required := []string{
		"registro bootstrap",
		"no es un receipt productivo",
		"no acredita el mapeo, la vista estable, el cercado, la presencia o ausencia real",
		"los recibos del censo ni ninguna compuerta",
		"commit: `08063318512682b7c6e5212affc1986afa363da3`",
		"código productivo Go: 847 líneas",
		"pruebas Go: 847 líneas",
		"fichero mayor: 203 líneas",
		"función mayor: 59 líneas",
	}
	for _, fragment := range required {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("falta el límite o la métrica %q", fragment)
		}
	}
}
