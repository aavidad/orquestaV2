package candidateseal

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildAndVerifyBindEveryGateASubject(t *testing.T) {
	input := candidateInput("one")
	manifest, err := Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := Verify(manifest, input); err != nil {
		t.Fatalf("Verify built manifest: %v", err)
	}
	if manifest.Gate != "A" {
		t.Fatalf("gate=%q, want A", manifest.Gate)
	}
	for name, digest := range map[string]string{
		"source tree":      manifest.SourceTree.SHA256,
		"binary":           manifest.Binary.SHA256,
		"effective config": manifest.EffectiveConfig.SHA256,
		"candidate":        manifest.CandidateSHA256,
		"manifest":         manifest.ManifestSHA256,
	} {
		if !isSHA256ForTest(digest) {
			t.Errorf("%s digest=%q", name, digest)
		}
	}

	t.Run("source order is canonical", func(t *testing.T) {
		reordered := candidateInput("one")
		reordered.SourceFiles[0], reordered.SourceFiles[1] = reordered.SourceFiles[1], reordered.SourceFiles[0]
		got, err := Build(reordered)
		if err != nil {
			t.Fatalf("Build reordered: %v", err)
		}
		if got != manifest {
			t.Fatalf("source enumeration changed manifest:\n got %#v\nwant %#v", got, manifest)
		}
	})
}

func TestVerifyRejectsSubjectDriftAndTamper(t *testing.T) {
	input := candidateInput("one")
	manifest, err := Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tests := map[string]func(*Input, *Manifest){
		"source content":            func(in *Input, _ *Manifest) { in.SourceFiles[0].Content[0] ^= 1 },
		"source path":               func(in *Input, _ *Manifest) { in.SourceFiles[0].Path = "internal/other.go" },
		"source mode":               func(in *Input, _ *Manifest) { in.SourceFiles[0].Mode ^= 0111 },
		"binary":                    func(in *Input, _ *Manifest) { in.Binary[0] ^= 1 },
		"effective config":          func(in *Input, _ *Manifest) { in.EffectiveConfig[0] ^= 1 },
		"manifest source digest":    func(_ *Input, got *Manifest) { got.SourceTree.SHA256 = testDigest('0') },
		"manifest binary digest":    func(_ *Input, got *Manifest) { got.Binary.SHA256 = testDigest('0') },
		"manifest config digest":    func(_ *Input, got *Manifest) { got.EffectiveConfig.SHA256 = testDigest('0') },
		"manifest candidate digest": func(_ *Input, got *Manifest) { got.CandidateSHA256 = testDigest('0') },
		"manifest seal digest":      func(_ *Input, got *Manifest) { got.ManifestSHA256 = testDigest('0') },
		"gate attribution":          func(_ *Input, got *Manifest) { got.Gate = "B" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changed := cloneInput(input)
			changedManifest := manifest
			mutate(&changed, &changedManifest)
			if err := Verify(changedManifest, changed); err == nil {
				t.Fatal("Verify accepted drift or tamper")
			}
		})
	}
}

func TestVerifyRejectsMissingMalformedAndCrossedDigests(t *testing.T) {
	one := candidateInput("one")
	two := candidateInput("two")
	oneManifest, err := Build(one)
	if err != nil {
		t.Fatalf("Build one: %v", err)
	}
	twoManifest, err := Build(two)
	if err != nil {
		t.Fatalf("Build two: %v", err)
	}

	for name, digest := range map[string]string{"missing": "", "short": "abcd", "non-hex": "sha256:" + strings.Repeat("z", 64)} {
		t.Run(name, func(t *testing.T) {
			for field := 0; field < 5; field++ {
				changed := oneManifest
				switch field {
				case 0:
					changed.SourceTree.SHA256 = digest
				case 1:
					changed.Binary.SHA256 = digest
				case 2:
					changed.EffectiveConfig.SHA256 = digest
				case 3:
					changed.CandidateSHA256 = digest
				case 4:
					changed.ManifestSHA256 = digest
				}
				if err := Verify(changed, one); err == nil {
					t.Fatalf("field %d accepted digest %q", field, digest)
				}
			}
		})
	}

	crossed := oneManifest
	crossed.Binary = twoManifest.Binary
	crossed.CandidateSHA256 = twoManifest.CandidateSHA256
	if err := Verify(crossed, one); err == nil {
		t.Fatal("Verify accepted binary and candidate digest from a different subject")
	}
	if err := Verify(oneManifest, two); err == nil {
		t.Fatal("Verify accepted a manifest against a different candidate")
	}
}

func TestManifestCannotAttributeGateBKVMCOrPhysicalExecution(t *testing.T) {
	typeOfManifest := reflect.TypeOf(Manifest{})
	for i := 0; i < typeOfManifest.NumField(); i++ {
		name := strings.ToLower(typeOfManifest.Field(i).Name)
		for _, forbidden := range []string{"gateb", "gatec", "kvm", "firecracker", "microvm", "physical", "wave", "receipt"} {
			if strings.Contains(name, forbidden) {
				t.Fatalf("Manifest field %q can over-attribute %q", typeOfManifest.Field(i).Name, forbidden)
			}
		}
	}
}

func candidateInput(suffix string) Input {
	return Input{
		SourceFiles: []SourceFile{
			{Path: "internal/a.go", Mode: 0644, Content: []byte("package internal // " + suffix)},
			{Path: "cmd/orquesta/main.go", Mode: 0755, Content: []byte("package main // " + suffix)},
		},
		Binary:          []byte("orquesta-binary-" + suffix),
		EffectiveConfig: []byte(`{"document_type":"orquesta.effective_config","schema_version":1,"registry_revision":"r1-` + suffix + `","snapshot_hash":"sha256:` + strings.Repeat("a", 64) + `","entries":[{"key":"runtime.secret","value":"[REDACTED]","source":"file","type":"credential_ref","semantic_ref":"orquesta.config.runtime.secret","sensitive":true,"scope":"runtime","restart_required":true,"env_alias":"ORQUESTA_PRIVATE_API_KEY","validator_ids":["credential_ref"]}]}`),
	}
}

func cloneInput(input Input) Input {
	cloned := Input{
		SourceFiles:     make([]SourceFile, len(input.SourceFiles)),
		Binary:          append([]byte(nil), input.Binary...),
		EffectiveConfig: append([]byte(nil), input.EffectiveConfig...),
	}
	for i, source := range input.SourceFiles {
		cloned.SourceFiles[i] = source
		cloned.SourceFiles[i].Content = append([]byte(nil), source.Content...)
	}
	return cloned
}

func isSHA256ForTest(digest string) bool {
	if len(digest) != 71 || !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	for _, char := range strings.TrimPrefix(digest, "sha256:") {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return false
		}
	}
	return true
}

func testDigest(char byte) string {
	return "sha256:" + strings.Repeat(string(char), 64)
}
