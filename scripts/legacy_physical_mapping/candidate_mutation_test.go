// Estas pruebas rechazan deriva, mezclas temporales y falsas ausencias.
package main

import (
	"bytes"
	"testing"
)

func TestRejectsBaseSubjectAndSealMutations(t *testing.T) {
	base := validCandidate(t)
	tests := map[string]func(*mappingCandidate){
		"falta": func(value *mappingCandidate) {
			value.Observations = value.Observations[:381]
		},
		"sobra": func(value *mappingCandidate) {
			value.Observations = append(value.Observations, value.Observations[0])
		},
		"duplicado": func(value *mappingCandidate) {
			value.Observations[1].Subject = value.Observations[0].Subject
		},
		"reordenado": func(value *mappingCandidate) {
			value.Observations[0], value.Observations[1] = value.Observations[1], value.Observations[0]
		},
		"histórico": func(value *mappingCandidate) {
			value.Observations[0].ExistsAtV3Observation = !value.Observations[0].ExistsAtV3Observation
		},
		"alias como ruta": func(value *mappingCandidate) {
			for index := range value.Observations {
				if value.Observations[index].Subject.Kind == "member" {
					value.Observations[index].Subject.PathAlias = "/ruta/prohibida"
					return
				}
			}
		},
		"V3":       func(value *mappingCandidate) { value.SourceV3BytesSHA256 = testDigest("otro-v3") },
		"universo": func(value *mappingCandidate) { value.UniverseBytesSHA256 = testDigest("otro-universo") },
		"referencias": func(value *mappingCandidate) {
			value.LogicalReferenceSetSHA256 = testDigest("otras-referencias")
		},
		"sujetos": func(value *mappingCandidate) { value.SubjectSetSHA256 = testDigest("otros-sujetos") },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			value := base
			value.Observations = append([]mappingObservation(nil), base.Observations...)
			mutate(&value)
			resealCandidate(t, &value)
			requireCandidateFailure(t, value)
		})
	}
}

func TestRejectsContextDriftAndMixedAttempts(t *testing.T) {
	for name, mutate := range map[string]func(*mappingCandidate){
		"vista":               func(value *mappingCandidate) { value.Context.ViewRef = opaqueValue("view", "otra") },
		"cercado":             func(value *mappingCandidate) { value.Context.FenceGeneration++ },
		"perdida":             func(value *mappingCandidate) { value.Context.FenceState = "declared_lost" },
		"mutable":             func(value *mappingCandidate) { value.Context.ViewState = "declared_mutable" },
		"política":            func(value *mappingCandidate) { value.Context.PolicySHA256 = testDigest("otra") },
		"referencia no opaca": func(value *mappingCandidate) { value.Context.ViewRef = "view_/ruta" },
		"huella mal formada":  func(value *mappingCandidate) { value.Context.PolicySHA256 = "sha256:ABC" },
		"ventana":             func(value *mappingCandidate) { value.Context.WindowEndedAt = value.Context.WindowStartedAt },
		"fuera":               func(value *mappingCandidate) { value.Observations[0].ObservedAt = "2026-07-30T04:11:00Z" },
	} {
		t.Run(name, func(t *testing.T) {
			value := validCandidate(t)
			mutate(&value)
			if name != "vista" && name != "cercado" && name != "política" {
				resealCandidate(t, &value)
			}
			requireCandidateFailure(t, value)
		})
	}
	value := validCandidate(t)
	raw := encodeCandidate(t, value)
	mixed := bytes.Replace(raw, []byte(`"subject":`),
		[]byte(`"view_ref":"view_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","subject":`), 1)
	requireRawFailure(t, mixed)
	mixed = bytes.Replace(raw, []byte(`"subject":`),
		[]byte(`"attempt_ref":"attempt_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","subject":`), 1)
	requireRawFailure(t, mixed)
}

func TestRejectsContradictoryPresenceAndPermissionAsAbsence(t *testing.T) {
	for name, mutate := range map[string]func(*mappingCandidate){
		"presente sin identidad": func(value *mappingCandidate) { value.Observations[0].IdentityRef = "" },
		"ausente con identidad": func(value *mappingCandidate) {
			makeAbsent(t, value, 0)
			value.Observations[0].IdentityRef = opaqueValue("identity", "prohibida")
		},
		"ausente sin evidencia": func(value *mappingCandidate) {
			makeAbsent(t, value, 0)
			value.Observations[0].AbsenceEvidenceRef = ""
		},
		"permiso como ausencia": func(value *mappingCandidate) {
			makeAbsent(t, value, 0)
			value.Observations[0].ObservationOutcome = "permission_denied"
		},
		"tipo inválido": func(value *mappingCandidate) { value.Observations[0].ObservedType = "symlink" },
	} {
		t.Run(name, func(t *testing.T) {
			value := validCandidate(t)
			mutate(&value)
			resealCandidate(t, &value)
			requireCandidateFailure(t, value)
		})
	}
}

func TestRejectsSemanticIntegrityAndPinnedBaseByteDrift(t *testing.T) {
	value := validCandidate(t)
	value.Observations[0].ObservedType = "regular"
	requireCandidateFailure(t, value)
	v3Raw, universeRaw, raw := testInputs(t, validCandidate(t))
	v3Raw = append([]byte(" "), v3Raw...)
	if _, err := validateMapping(v3Raw, universeRaw, raw); err == nil {
		t.Fatal("se aceptó V3 con bytes distintos")
	}
	v3Raw = readTestFile(t, v3Fixture)
	universeRaw = append(universeRaw, '\n')
	if _, err := validateMapping(v3Raw, universeRaw, raw); err == nil {
		t.Fatal("se aceptó universo con bytes distintos")
	}
}

func requireCandidateFailure(t *testing.T, value mappingCandidate) {
	t.Helper()
	requireRawFailure(t, encodeCandidate(t, value))
}

func requireRawFailure(t *testing.T, raw []byte) {
	t.Helper()
	if output, err := validateMapping(readTestFile(t, v3Fixture), readTestFile(t, universeFixture), raw); err == nil || len(output) != 0 {
		t.Fatalf("candidato inválido aceptado: error=%v bytes=%d", err, len(output))
	}
}
