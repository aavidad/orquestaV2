package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGateACLICreatesAndVerifiesExternalCandidateArtifact(t *testing.T) {
	fixture := newGateACandidate(t)
	manifest := filepath.Join(t.TempDir(), "gate-a.json")
	arguments := gateAArguments(fixture, manifest)

	var created, stderr bytes.Buffer
	if code := runGateA(append([]string{"create"}, arguments...), &created, &stderr); code != 0 {
		t.Fatalf("create code=%d stderr=%q", code, stderr.String())
	}
	info, err := os.Stat(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o400 {
		t.Fatalf("manifest mode=%o, want 400", info.Mode().Perm())
	}
	artifact, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"gate": "B"`, `"gate": "C"`, "kvm", "firecracker", "microvm", "physical", "wave", "receipt"} {
		if strings.Contains(strings.ToLower(string(artifact)), forbidden) {
			t.Fatalf("Gate A artifact attributes forbidden subject %q: %s", forbidden, artifact)
		}
	}

	var verified bytes.Buffer
	stderr.Reset()
	if code := runGateA(append([]string{"verify"}, arguments...), &verified, &stderr); code != 0 {
		t.Fatalf("verify code=%d stderr=%q", code, stderr.String())
	}
	if created.String() != verified.String() || !strings.HasPrefix(created.String(), "sha256:") {
		t.Fatalf("candidate identity create=%q verify=%q", created.String(), verified.String())
	}

	if err := os.WriteFile(fixture.binary, []byte("drift"), 0o700); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if code := runGateA(append([]string{"verify"}, arguments...), &bytes.Buffer{}, &stderr); code == 0 || !strings.Contains(stderr.String(), "verification_failed") {
		t.Fatalf("drift verify code=%d stderr=%q", code, stderr.String())
	}
}

func TestGateACLIRejectsMissingInputsAndManifestInsideSourceSubject(t *testing.T) {
	fixture := newGateACandidate(t)
	externalManifest := filepath.Join(t.TempDir(), "gate-a.json")
	base := gateAArguments(fixture, externalManifest)
	for name, index := range map[string]int{"source": 1, "binary": 3, "config": 5} {
		t.Run(name, func(t *testing.T) {
			arguments := append([]string(nil), base...)
			arguments[index] = filepath.Join(t.TempDir(), "missing")
			var stderr bytes.Buffer
			if code := runGateA(append([]string{"create"}, arguments...), &bytes.Buffer{}, &stderr); code == 0 || !strings.Contains(stderr.String(), "input_invalid") {
				t.Fatalf("missing %s code=%d stderr=%q", name, code, stderr.String())
			}
		})
	}

	inside := filepath.Join(fixture.root, "gate-a.json")
	var stderr bytes.Buffer
	if code := runGateA(append([]string{"create"}, gateAArguments(fixture, inside)...), &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatalf("manifest inside source subject accepted; stderr=%q", stderr.String())
	}
}

func TestGateACLIRejectsExternalAliasBackIntoSourceTree(t *testing.T) {
	fixture := newGateACandidate(t)
	alias := filepath.Join(t.TempDir(), "source-alias")
	if err := os.Symlink(fixture.root, alias); err != nil {
		t.Fatal(err)
	}
	externalManifest := filepath.Join(t.TempDir(), "gate-a.json")

	t.Run("manifest", func(t *testing.T) {
		var stderr bytes.Buffer
		if code := runGateA(append([]string{"create"}, gateAArguments(fixture, filepath.Join(alias, "gate-a.json"))...), &bytes.Buffer{}, &stderr); code == 0 {
			t.Fatalf("manifest alias back into source accepted; stderr=%q", stderr.String())
		}
	})

	for name, path := range map[string]string{
		"binary": filepath.Join(fixture.root, "candidate-binary"),
		"config": filepath.Join(fixture.root, "candidate-effective.json"),
	} {
		t.Run(name, func(t *testing.T) {
			content, mode := []byte("binary"), os.FileMode(0o700)
			if name == "config" {
				content, mode = []byte(gateAEffectiveConfig()), 0o400
			}
			if err := os.WriteFile(path, content, mode); err != nil {
				t.Fatal(err)
			}
			changed := fixture
			if name == "binary" {
				changed.binary = filepath.Join(alias, filepath.Base(path))
			} else {
				changed.config = filepath.Join(alias, filepath.Base(path))
			}
			var stderr bytes.Buffer
			if code := runGateA(append([]string{"create"}, gateAArguments(changed, externalManifest)...), &bytes.Buffer{}, &stderr); code == 0 {
				t.Fatalf("%s alias back into source accepted; stderr=%q", name, stderr.String())
			}
		})
	}
}

func TestGateACLIRejectsUnmarkedAndUnredactedEffectiveConfig(t *testing.T) {
	fixture := newGateACandidate(t)
	manifest := filepath.Join(t.TempDir(), "gate-a.json")
	tests := map[string]string{
		"unmarked":    `{"schema_version":1,"registry_revision":"r1","snapshot_hash":"sha256:` + strings.Repeat("a", 64) + `","entries":[{}]}`,
		"secret leak": strings.Replace(gateAEffectiveConfig(), `"value":"[REDACTED]"`, `"value":"secret"`, 1),
		"unknown":     strings.TrimSuffix(gateAEffectiveConfig(), "}") + `,"gate_b":true}`,
		"trailing":    gateAEffectiveConfig() + `{}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			if err := os.Chmod(fixture.config, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fixture.config, []byte(content), 0o400); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(fixture.config, 0o400); err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			if code := runGateA(append([]string{"create"}, gateAArguments(fixture, manifest)...), &bytes.Buffer{}, &stderr); code == 0 || !strings.Contains(stderr.String(), "build_failed") {
				t.Fatalf("unsafe config accepted code=%d stderr=%q", code, stderr.String())
			}
		})
	}
}

type gateACandidateFixture struct{ root, binary, config string }

func newGateACandidate(t *testing.T) gateACandidateFixture {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.go"), []byte("package candidate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "-C", root, "init", "-q")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	binary := filepath.Join(t.TempDir(), "orquesta")
	config := filepath.Join(t.TempDir(), "effective.json")
	if err := os.WriteFile(binary, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte(gateAEffectiveConfig()), 0o400); err != nil {
		t.Fatal(err)
	}
	return gateACandidateFixture{root: root, binary: binary, config: config}
}

func gateAArguments(fixture gateACandidateFixture, manifest string) []string {
	return []string{"--source-root", fixture.root, "--binary", fixture.binary, "--effective-config", fixture.config, "--manifest", manifest}
}

func gateAEffectiveConfig() string {
	return `{"document_type":"orquesta.effective_config","schema_version":1,"registry_revision":"r1","snapshot_hash":"sha256:` + strings.Repeat("a", 64) + `","entries":[{"key":"runtime.secret","value":"[REDACTED]","source":"file","type":"credential_ref","semantic_ref":"orquesta.config.runtime.secret","sensitive":true,"scope":"runtime","restart_required":true,"env_alias":"ORQUESTA_PRIVATE_API_KEY","validator_ids":["credential_ref"]}]}`
}
