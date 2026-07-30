// Estas pruebas rechazan JSON ambiguo, campos físicos y límites excedidos.
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRejectsNonCanonicalDuplicateAndPhysicalFields(t *testing.T) {
	raw := encodeCandidate(t, validCandidate(t))
	mutations := map[string][]byte{
		"espacio": append([]byte(" "), raw...),
		"BOM":     append([]byte{0xef, 0xbb, 0xbf}, raw...),
		"duplicada": bytes.Replace(raw, []byte(`"document_kind":`),
			[]byte(`"document_kind":"duplicado","document_kind":`), 1),
		"ruta": bytes.Replace(raw, []byte(`"subject":`),
			[]byte(`"path":"/secreto","subject":`), 1),
		"descriptor": bytes.Replace(raw, []byte(`"subject":`),
			[]byte(`"descriptor":7,"subject":`), 1),
		"dispositivo": bytes.Replace(raw, []byte(`"subject":`),
			[]byte(`"device":8,"inode":9,"uid":10,"user":"alguien","subject":`), 1),
		"recibo": bytes.Replace(raw, []byte(`"context":`),
			[]byte(`"accredited_receipt":true,"context":`), 1),
	}
	for name, mutated := range mutations {
		t.Run(name, func(t *testing.T) { requireRawFailure(t, mutated) })
	}
}

func TestJSONBoundsRejectDepthAndArrayNPlusOne(t *testing.T) {
	deep := []byte(strings.Repeat("[", maxJSONDepth+1) + "0" +
		strings.Repeat("]", maxJSONDepth+1))
	if err := validateJSONBounds(deep); err == nil {
		t.Fatal("se aceptó profundidad N+1")
	}
	var array strings.Builder
	array.WriteByte('[')
	for index := 0; index < maxArrayItems+1; index++ {
		if index > 0 {
			array.WriteByte(',')
		}
		array.WriteString("null")
	}
	array.WriteByte(']')
	if err := validateJSONBounds([]byte(array.String())); err == nil {
		t.Fatal("se aceptó matriz N+1")
	}
}

func TestJSONBoundsRejectTokenBudgetNPlusOne(t *testing.T) {
	innerValues := make([]string, 600)
	for index := range innerValues {
		innerValues[index] = "null"
	}
	inner := "[" + strings.Join(innerValues, ",") + "]"
	outerValues := make([]string, 100)
	for index := range outerValues {
		outerValues[index] = inner
	}
	if err := validateJSONBounds([]byte("[" + strings.Join(outerValues, ",") + "]")); err == nil {
		t.Fatal("se aceptó un documento por encima del máximo de elementos léxicos")
	}
}

func TestPortableJSONEncodingAndDigestFrame(t *testing.T) {
	fixture := struct {
		Kind     string `json:"kind"`
		Optional string `json:"optional,omitempty"`
		Text     string `json:"text"`
	}{Kind: "ejemplo", Text: "<á>&"}
	raw, err := canonicalJSON(fixture)
	if err != nil || string(raw) != "{\"kind\":\"ejemplo\",\"text\":\"<á>&\"}\n" {
		t.Fatalf("formato portable alterado: %q error=%v", raw, err)
	}
	digest, err := domainDigest("orquesta.fixture-portable.v1", fixture)
	if err != nil || digest != "sha256:cca6fade309497ab78643f9689f50e4a3c7897276fa33efeebf86228db3bfeb7" {
		t.Fatalf("encuadre de huella alterado: %s error=%v", digest, err)
	}
}

func TestRejectsOverlongOpaqueReferencesAndNonUTF8(t *testing.T) {
	raw := encodeCandidate(t, validCandidate(t))
	long := "view_" + strings.Repeat("a", maxStringBytes+1)
	mutated := bytes.Replace(raw, []byte(`"view_`), []byte(`"`+long), 1)
	requireRawFailure(t, mutated)
	invalidUTF8 := append([]byte(nil), raw...)
	invalidUTF8[len(invalidUTF8)/2] = 0xff
	requireRawFailure(t, invalidUTF8)
}
