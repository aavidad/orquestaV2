// Este fichero simula cortes persistentes antes y después del punto de confirmación.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInjectedCutsPreserveEvidenceAndOnlyManifestConfirms(t *testing.T) {
	for _, step := range []string{"journal", "jsonl", "manifest"} {
		t.Run(step, func(t *testing.T) {
			opts := testOptions(t, t.TempDir(), modeMetadata)
			publication := preparedPublication(t, opts)
			publication.cut = func(current string) error {
				if current == step {
					return errors.New("cut_injected")
				}
				return nil
			}
			if err := publication.publish(); err == nil {
				t.Fatal("el corte inyectado devolvió éxito")
			}
			mustSucceed(t, publication.abort())
			_, jsonErr := os.Stat(opts.jsonlPath)
			_, manifestErr := os.Stat(opts.manifestPath)
			if step == "manifest" {
				if jsonErr != nil || manifestErr != nil {
					t.Fatalf("el commit ya visible quedó incompleto: %v / %v", jsonErr, manifestErr)
				}
			} else if !errors.Is(manifestErr, os.ErrNotExist) {
				t.Fatalf("un corte previo confirmó la generación: %v", manifestErr)
			} else if step == "jsonl" && jsonErr != nil {
				t.Fatalf("el JSONL no confirmado no se preservó: %v", jsonErr)
			}
		})
	}
}
func TestNextStartPreservesCrashAfterJSONLRename(t *testing.T) {
	opts := testOptions(t, t.TempDir(), modeMetadata)
	publication := preparedPublication(t, opts)
	journal := testJournal(t, publication)
	mustSucceed(t, publication.writeJournal(journal))
	promoteStages(t, publication, false)
	mustSucceed(t, simulateCrash(publication))
	if _, err := newPublicationForTest(opts); !errors.Is(err, errRecoveryConflict) {
		t.Fatalf("la reanudación borró o aceptó evidencia ambigua: %v", err)
	}
	if _, err := os.Stat(opts.jsonlPath); err != nil {
		t.Fatalf("el JSONL ambiguo no fue preservado: %v", err)
	}
}
func TestNextStartRecognizesCrashAfterManifestCommit(t *testing.T) {
	opts := testOptions(t, t.TempDir(), modeMetadata)
	publication := preparedPublication(t, opts)
	mustSucceed(t, publication.writeJournal(testJournal(t, publication)))
	promoteStages(t, publication, true)
	mustSucceed(t, simulateCrash(publication))
	if _, err := newPublicationForTest(opts); !errors.Is(err, errOutputNameExists) {
		t.Fatalf("no se reconoció el commit previo: %v", err)
	}
	journalPath := filepath.Join(filepath.Dir(opts.jsonlPath), "."+pairIDForTest(opts)+".journal")
	if _, err := os.Stat(journalPath); err != nil {
		t.Fatalf("el diario confirmado no se preservó: %v", err)
	}
}
func TestCommittedFinalsMustMatchJournalIdentitiesAndDigest(t *testing.T) {
	cases := []struct {
		name   string
		change func(*journalRecord)
	}{
		{"identidad_jsonl", func(value *journalRecord) { value.JSONLIdentity = "otra" }},
		{"identidad_manifiesto", func(value *journalRecord) { value.ManifestIdentity = "otra" }},
		{"resumen_jsonl", func(value *journalRecord) { value.JSONLSHA256 = "otro" }},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			opts := testOptions(t, t.TempDir(), modeMetadata)
			publication := preparedPublication(t, opts)
			journal := testJournal(t, publication)
			item.change(&journal)
			mustSucceed(t, publication.writeJournal(journal))
			promoteStages(t, publication, true)
			mustSucceed(t, simulateCrash(publication))
			if _, err := newPublicationForTest(opts); !errors.Is(err, errRecoveryConflict) {
				t.Fatalf("el diario incoherente fue aceptado: %v", err)
			}
		})
	}
}
func TestUnconfirmedStageWithoutJournalIsPreserved(t *testing.T) {
	opts := testOptions(t, t.TempDir(), modeMetadata)
	stage := filepath.Join(
		filepath.Dir(opts.jsonlPath), "."+pairIDForTest(opts)+".jsonl.stage",
	)
	mustWrite(t, stage, "artefacto-ajeno", 0o600)
	if _, err := newPublicationForTest(opts); !errors.Is(err, errRecoveryConflict) {
		t.Fatalf("la etapa ambigua fue aceptada o retirada: %v", err)
	}
	content, err := os.ReadFile(stage)
	if err != nil || string(content) != "artefacto-ajeno" {
		t.Fatalf("la etapa ambigua no se preservó: %q / %v", content, err)
	}
}
func preparedPublication(t *testing.T, opts options) *publication {
	t.Helper()
	publication, err := newPublicationForTest(opts)
	mustSucceed(t, err)
	_, err = publication.jsonl.WriteString("{\"prueba\":true}\n")
	mustSucceed(t, err)
	digest, err := digestOpenFile(publication.jsonl)
	mustSucceed(t, err)
	stat, _ := publication.jsonl.Stat()
	value := manifest{
		Schema: schemaVersion, Algorithm: "sha256", Complete: true,
		JSONLSHA256: digest, JSONLBytes: stat.Size(), JSONLName: publication.jsonlFinal,
		ManifestDigestDomain: manifestDigestDomain,
	}
	mustSucceed(t, publication.prepareManifest(value))
	return publication
}
func testJournal(t *testing.T, publication *publication) journalRecord {
	t.Helper()
	mustSucceed(t, errors.Join(publication.jsonl.Sync(), publication.manifest.Sync()))
	digest, err := digestOpenFile(publication.jsonl)
	mustSucceed(t, err)
	return journalRecord{
		Schema: "legacy_physical_inventory_journal/v1", PairID: publication.anchor.pairID,
		JSONLStage: publication.jsonlStage, ManifestStage: publication.manifestStage,
		JSONLFinal: publication.jsonlFinal, ManifestFinal: publication.manifestFinal,
		JSONLIdentity: publication.jsonlIdentity, ManifestIdentity: publication.manifestIdentity,
		JSONLSHA256: digest,
	}
}
func promoteStages(t *testing.T, publication *publication, manifest bool) {
	t.Helper()
	pairs := [][2]string{{publication.jsonlStage, publication.jsonlFinal}}
	if manifest {
		pairs = append(pairs, [2]string{publication.manifestStage, publication.manifestFinal})
	}
	for _, pair := range pairs {
		mustSucceed(t, renameNoReplace(publication.anchor.fd(), pair[0], pair[1]))
	}
	mustSucceed(t, publication.anchor.sync())
}
func pairIDForTest(opts options) string {
	anchor, err := openOutputAnchor(opts.jsonlPath, opts.manifestPath)
	if err != nil {
		return ""
	}
	defer anchor.close()
	return anchor.pairID
}
func simulateCrash(publication *publication) error {
	return publication.abort()
}
