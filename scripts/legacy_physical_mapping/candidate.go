// Este fichero valida contexto, presencia y evidencia declarada por cada sujeto.
package main

import "time"

type bindingGroup struct {
	identityRef, reopenDigest, observedType string
	subjects                                []int
}
type observationState struct {
	groups                         map[string]*bindingGroup
	bindingOrder, bindingBySubject []string
	subjectIndexes                 map[string]int
	subjects                       []subjectRef
	present                        []bool
	counts                         verdictCounts
}

func validateMapping(v3Raw, universeRaw, candidateRaw []byte) ([]byte, error) {
	facts, err := validateBases(v3Raw, universeRaw)
	if err != nil {
		return nil, err
	}
	candidate, err := decodeCanonicalCandidate(candidateRaw)
	if err != nil {
		return nil, err
	}
	start, end, err := validateCandidateHeader(candidate)
	if err != nil {
		return nil, err
	}
	state, err := validateObservations(candidate.Observations, facts, start, end)
	if err != nil {
		return nil, err
	}
	if err := validateOwnership(candidate.Ownership, &state); err != nil {
		return nil, err
	}
	unsigned := candidate
	unsigned.SemanticIntegritySHA256 = ""
	bodyDigest, err := domainDigest(candidateDigestDomain, unsigned)
	if err != nil || bodyDigest != candidate.SemanticIntegritySHA256 {
		return nil, contractFailure("integridad_semantica_invalida")
	}
	verdict := validationVerdict{
		DocumentKind: "orquesta_legacy_mapping_validation_verdict", SchemaVersion: 1,
		Decision: "candidate_semantically_coherent", Authority: "semantic_validator_only",
		SourceV3BytesSHA256: expectedV3SHA, UniverseBytesSHA256: expectedUniverseSHA,
		CandidateBytesSHA256:             rawSHA256(candidateRaw),
		CandidateSemanticIntegritySHA256: candidate.SemanticIntegritySHA256,
		ContextSHA256:                    candidate.ContextSHA256, Counts: state.counts,
	}
	encoded, err := canonicalJSON(verdict)
	if err != nil || len(encoded) > maxVerdictBytes {
		return nil, contractFailure("salida_invalida")
	}
	return encoded, nil
}

func validateCandidateHeader(value mappingCandidate) (time.Time, time.Time, error) {
	if value.DocumentKind != "orquesta_legacy_private_mapping_candidate" ||
		value.SchemaVersion != 1 ||
		value.SourceV3BytesSHA256 != expectedV3SHA ||
		value.UniverseBytesSHA256 != expectedUniverseSHA ||
		value.LogicalReferenceSetSHA256 != expectedReferences ||
		value.SubjectSetSHA256 != expectedSubjects ||
		!digestPattern.MatchString(value.ContextSHA256) ||
		!digestPattern.MatchString(value.SemanticIntegritySHA256) {
		return time.Time{}, time.Time{}, contractFailure("candidato_incompatible")
	}
	contextDigest, err := domainDigest(contextDigestDomain, value.Context)
	if err != nil || contextDigest != value.ContextSHA256 {
		return time.Time{}, time.Time{}, contractFailure("contexto_invalido")
	}
	context := value.Context
	if !opaqueRef(context.ViewRef, "view") ||
		!opaqueRef(context.FenceRef, "fence") ||
		!opaqueRef(context.PolicyRef, "policy") ||
		!opaqueRef(context.AttemptRef, "attempt") ||
		context.ViewState != "declared_stable" ||
		context.FenceState != "declared_valid" ||
		context.FenceGeneration == 0 ||
		!digestPattern.MatchString(context.ViewEvidenceSHA256) ||
		!digestPattern.MatchString(context.FenceEvidenceSHA256) ||
		!digestPattern.MatchString(context.PolicySHA256) ||
		!digestPattern.MatchString(context.ConfigurationSHA256) {
		return time.Time{}, time.Time{}, contractFailure("contexto_invalido")
	}
	start, validStart := canonicalTime(context.WindowStartedAt)
	end, validEnd := canonicalTime(context.WindowEndedAt)
	if !validStart || !validEnd || !start.Before(end) {
		return time.Time{}, time.Time{}, contractFailure("ventana_invalida")
	}
	return start, end, nil
}

func validateObservations(
	values []mappingObservation,
	facts baseFacts,
	start, end time.Time,
) (observationState, error) {
	state := newObservationState(facts)
	if values == nil || len(values) != 382 {
		return state, contractFailure("observaciones_incompletas")
	}
	identityBindings := make(map[string]string, 382)
	for index, value := range values {
		if value.Subject != facts.subjects[index] ||
			value.ExistsAtV3Observation != facts.historical[index] {
			return state, contractFailure("observacion_desalineada")
		}
		observedAt, valid := canonicalTime(value.ObservedAt)
		if !valid || observedAt.Before(start) || observedAt.After(end) {
			return state, contractFailure("observacion_fuera_de_ventana")
		}
		if value.PresentInStableView {
			if err := acceptPresentObservation(index, value, &state, identityBindings); err != nil {
				return state, err
			}
		} else if err := acceptAbsentObservation(value, &state); err != nil {
			return state, err
		}
	}
	return state, nil
}

func newObservationState(facts baseFacts) observationState {
	indexes := make(map[string]int, len(facts.subjects))
	for index, subject := range facts.subjects {
		indexes[subjectKey(subject)] = index
	}
	return observationState{
		groups:           make(map[string]*bindingGroup, 382),
		bindingBySubject: make([]string, 382),
		subjectIndexes:   indexes,
		subjects:         facts.subjects,
		present:          make([]bool, 382),
		counts: verdictCounts{
			LogicalReferences: 112, SimpleReferences: 97, Collections: 15,
			CollectionMembers: 285, Subjects: 382,
			HistoricalPresent: 364, HistoricalAbsent: 18,
		},
	}
}

func acceptPresentObservation(
	index int,
	value mappingObservation,
	state *observationState,
	identityBindings map[string]string,
) error {
	if value.ObservationOutcome != "present" ||
		(value.ObservedType != "regular" && value.ObservedType != "directory") ||
		!opaqueRef(value.IdentityRef, "identity") ||
		!opaqueRef(value.BindingRef, "binding") ||
		!digestPattern.MatchString(value.ReopenEvidenceSHA256) ||
		value.AbsenceEvidenceRef != "" || value.AbsenceEvidenceSHA256 != "" {
		return contractFailure("presencia_invalida")
	}
	if binding, exists := identityBindings[value.IdentityRef]; exists && binding != value.BindingRef {
		return contractFailure("propiedad_ambigua")
	}
	identityBindings[value.IdentityRef] = value.BindingRef
	group := state.groups[value.BindingRef]
	if group == nil {
		group = &bindingGroup{
			identityRef: value.IdentityRef, reopenDigest: value.ReopenEvidenceSHA256,
			observedType: value.ObservedType,
		}
		state.groups[value.BindingRef] = group
		state.bindingOrder = append(state.bindingOrder, value.BindingRef)
	} else if group.identityRef != value.IdentityRef ||
		group.reopenDigest != value.ReopenEvidenceSHA256 || group.observedType != value.ObservedType {
		return contractFailure("vinculacion_incoherente")
	}
	group.subjects = append(group.subjects, index)
	state.bindingBySubject[index] = value.BindingRef
	state.present[index] = true
	state.counts.StablePresent++
	if value.ObservedType == "regular" {
		state.counts.Regular++
	} else {
		state.counts.Directories++
	}
	return nil
}

func acceptAbsentObservation(value mappingObservation, state *observationState) error {
	if value.ObservationOutcome != "not_found" || value.ObservedType != "absent" ||
		value.IdentityRef != "" || value.ReopenEvidenceSHA256 != "" || value.BindingRef != "" ||
		!opaqueRef(value.AbsenceEvidenceRef, "absence") ||
		!digestPattern.MatchString(value.AbsenceEvidenceSHA256) {
		return contractFailure("ausencia_invalida")
	}
	state.counts.StableAbsent++
	return nil
}

func canonicalTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, err == nil && parsed.Location() == time.UTC &&
		parsed.Format(time.RFC3339Nano) == value
}
