// Este contrato reejecuta la expansión sellada y la compara sin abrir fuentes físicas.
package orquesta_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

const (
	legacyPhysicalSubjectUniversePath = "product/traceability/legacy_physical_subject_universe_2026-07-30.json"
	legacyPhysicalSubjectUniverseSHA  = "b81478cef265fbb3970925cd09d450b23cd42196858e271c7b26a229fa106f1e"
)

func TestLegacyPhysicalSubjectUniverseReexecutesWithoutPhysicalRoots(t *testing.T) {
	command := exec.Command(
		"go", "run", "./scripts/legacy_physical_expansion",
		"--verify",
		"--v3", "product/traceability/legacy_source_roots_2026-07-30.json",
		"--universe", legacyPhysicalSubjectUniversePath,
	)
	generated, err := command.Output()
	if err != nil {
		t.Fatalf("reejecutar expansión lógica: %v", err)
	}
	committed, err := os.ReadFile(legacyPhysicalSubjectUniversePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, committed) {
		t.Fatal("el artefacto versionado no coincide byte a byte con la aplicación")
	}
	sum := sha256.Sum256(committed)
	if got := hex.EncodeToString(sum[:]); got != legacyPhysicalSubjectUniverseSHA {
		t.Fatalf("huella del artefacto = %s; se esperaba %s", got, legacyPhysicalSubjectUniverseSHA)
	}
	var document struct {
		DocumentKind              string `json:"document_kind"`
		SchemaVersion             int    `json:"schema_version"`
		LogicalReferenceSetSHA256 string `json:"logical_reference_set_sha256"`
		SubjectSetSHA256          string `json:"subject_set_sha256"`
		Counts                    struct {
			SourceRoots          int `json:"source_roots"`
			LogicalReferences    int `json:"logical_references"`
			SimpleReferences     int `json:"simple_references"`
			CollectionReferences int `json:"collection_references"`
			CollectionMembers    int `json:"collection_members"`
			PhysicalSubjects     int `json:"physical_subjects"`
			HistoricalPresent    int `json:"historical_present"`
			HistoricalAbsent     int `json:"historical_absent"`
		} `json:"counts"`
		Decisions struct {
			Include int `json:"incluir"`
			Exclude int `json:"excluir"`
			Pending int `json:"pendiente"`
		} `json:"decisions"`
		MembershipSeals []json.RawMessage `json:"membership_seals"`
		Subjects        []json.RawMessage `json:"subjects"`
	}
	if err := json.Unmarshal(committed, &document); err != nil {
		t.Fatal(err)
	}
	counts := document.Counts
	if document.DocumentKind != "orquesta_legacy_physical_subject_universe" ||
		document.SchemaVersion != 1 ||
		document.LogicalReferenceSetSHA256 != "sha256:2d2b81b27bfccad5a0b3a6dfabdec299a516dc3b31400f2cb2ecd6948c1e7834" ||
		document.SubjectSetSHA256 != "sha256:405f5b68f0e886a40bb69751fbb62f60567a15d6879393e9f42c4d1b5a458cc4" ||
		counts.SourceRoots != 122 || counts.LogicalReferences != 112 ||
		counts.SimpleReferences != 97 || counts.CollectionReferences != 15 ||
		counts.CollectionMembers != 285 || counts.PhysicalSubjects != 382 ||
		counts.HistoricalPresent != 364 || counts.HistoricalAbsent != 18 ||
		document.Decisions.Include != 36 || document.Decisions.Exclude != 49 ||
		document.Decisions.Pending != 27 || len(document.MembershipSeals) != 15 ||
		len(document.Subjects) != 382 {
		t.Fatalf("contrato de expansión incompleto: %#v", document)
	}
	for _, forbidden := range [][]byte{
		[]byte("present_in_stable_view"), []byte("physical_path"),
		[]byte("relative_path"), []byte("/home/"), []byte("Codex"),
	} {
		if bytes.Contains(committed, forbidden) {
			t.Fatalf("el artefacto publica el campo o dato prohibido %q", forbidden)
		}
	}
}
