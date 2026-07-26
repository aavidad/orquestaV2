package gaps

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"testing"

	"orquesta/internal/intake"
)

func TestEvaluatorV1SourceDigestGolden(t *testing.T) {
	hasher := sha256.New()
	for _, name := range []string{
		"../../intake/answer_text.go",
		"../catalog/builtin.go",
		"../catalog/catalog.go",
		"../catalog/errors.go",
		"../catalog/model.go",
		"../catalog/semantic.go",
		"../catalog/validation.go",
		"catalog.go",
		"engine.go",
		"errors.go",
		"input.go",
		"model.go",
		"rules.go",
	} {
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(hasher, name)
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write(content)
		_, _ = hasher.Write([]byte{0})
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	if got != evaluatorV1SourceDigestGolden {
		t.Fatalf(
			"evaluator V1 source digest=%q want=%q",
			got,
			evaluatorV1SourceDigestGolden,
		)
	}
}

func TestEvaluatorV1SemanticDigestGolden(t *testing.T) {
	if got := evaluatorV1SemanticDigest(); got != evaluatorV1DigestGolden {
		t.Fatalf(
			"evaluator V1 semantic digest=%q want=%q",
			got,
			evaluatorV1DigestGolden,
		)
	}
	identity := EvaluatorV1Identity()
	if identity.Schema != evaluatorSchema ||
		identity.Version != evaluatorV1Version ||
		identity.SemanticDigest != evaluatorV1DigestGolden {
		t.Fatalf("identity=%+v", identity)
	}
	if _, err := BuiltInEvaluatorRegistry().Resolve(identity); err != nil {
		t.Fatal(err)
	}
}

func TestBuiltInEvaluatorRegistryRejectsIdentityRebinding(t *testing.T) {
	identity := EvaluatorV1Identity()
	for name, mutated := range map[string]func(*intake.DerivationIdentity){
		"schema": func(value *intake.DerivationIdentity) {
			value.Schema = "orquesta.wizard.gaps.other"
		},
		"version": func(value *intake.DerivationIdentity) {
			value.Version = "v2"
		},
		"digest": func(value *intake.DerivationIdentity) {
			value.SemanticDigest = strings.Repeat("f", 64)
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := identity
			mutated(&changed)
			if _, err := BuiltInEvaluatorRegistry().Resolve(changed); !IsUnsupportedEvaluator(err) {
				t.Fatalf("mutated identity error=%v", err)
			}
		})
	}
}
