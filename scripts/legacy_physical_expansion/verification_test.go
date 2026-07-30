// Estas pruebas exigen verificación semántica ante cambios de identidad y orden.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const universeFixture = "../../product/traceability/legacy_physical_subject_universe_2026-07-30.json"

func TestVerificationModeRegeneratesExactUniverse(t *testing.T) {
	var output bytes.Buffer
	err := execute([]string{
		"--verify", "--v3", sourceFixture, "--universe", universeFixture,
	}, &output)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile(universeFixture)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), committed) {
		t.Fatal("el verificador no devolvió el universo regenerado exacto")
	}
}

func TestVerificationRejectsIdentityAliasAndOrderWithPreservedDigest(t *testing.T) {
	original, err := os.ReadFile(universeFixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*expansionDocument)
	}{
		{name: "root_id", mutate: func(value *expansionDocument) {
			value.Subjects[0].RootID = "raiz-manipulada"
		}},
		{name: "path_alias", mutate: func(value *expansionDocument) {
			for index := range value.Subjects {
				if value.Subjects[index].Kind == "member" {
					value.Subjects[index].PathAlias = "alias-manipulado"
					return
				}
			}
		}},
		{name: "orden", mutate: func(value *expansionDocument) {
			value.Subjects[0], value.Subjects[1] = value.Subjects[1], value.Subjects[0]
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var value expansionDocument
			if err := json.Unmarshal(original, &value); err != nil {
				t.Fatal(err)
			}
			originalDigest := value.SubjectSetSHA256
			test.mutate(&value)
			if value.SubjectSetSHA256 != originalDigest {
				t.Fatal("la mutación no conservó el digest declarado")
			}
			mutated, err := encodeExpansion(value)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "universo.json")
			if err := os.WriteFile(path, mutated, 0o600); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			err = execute([]string{
				"--verify", "--v3", sourceFixture, "--universe", path,
			}, &output)
			if err == nil || errorCode(err) != "universo_no_verificado" || output.Len() != 0 {
				t.Fatalf("mutación aceptada o publicada: error=%v bytes=%d", err, output.Len())
			}
		})
	}
}
