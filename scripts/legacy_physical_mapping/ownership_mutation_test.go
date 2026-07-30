// Estas pruebas cortan dueños ambiguos, podas huérfanas y ciclos declarados.
package main

import "testing"

func TestRejectsUnreconciledPhysicalAliasAndAmbiguousIdentity(t *testing.T) {
	value := validCandidate(t)
	value.Observations[1].IdentityRef = value.Observations[0].IdentityRef
	value.Observations[1].BindingRef = value.Observations[0].BindingRef
	value.Observations[1].ReopenEvidenceSHA256 = value.Observations[0].ReopenEvidenceSHA256
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)

	value = validCandidate(t)
	value.Observations[1].IdentityRef = value.Observations[0].IdentityRef
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}

func TestRejectsDifferentTypeForSamePhysicalBinding(t *testing.T) {
	value := validCandidate(t)
	first, second := &value.Observations[0], &value.Observations[1]
	second.IdentityRef, second.BindingRef = first.IdentityRef, first.BindingRef
	second.ReopenEvidenceSHA256 = first.ReopenEvidenceSHA256
	value.Ownership[0].AliasSubjects = []subjectRef{second.Subject}
	value.Ownership = append(value.Ownership[:1], value.Ownership[2:]...)
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}

func TestRejectsNoncanonicalAliasOwner(t *testing.T) {
	value := validCandidate(t)
	first, second := &value.Observations[0], &value.Observations[1]
	second.IdentityRef, second.BindingRef = first.IdentityRef, first.BindingRef
	second.ReopenEvidenceSHA256, second.ObservedType = first.ReopenEvidenceSHA256, first.ObservedType
	value.Ownership[0].OwnerSubject = second.Subject
	value.Ownership[0].AliasSubjects = []subjectRef{first.Subject}
	value.Ownership = append(value.Ownership[:1], value.Ownership[2:]...)
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}

func TestRejectsAbsentOwnerOrOrphanPrune(t *testing.T) {
	value := validCandidate(t)
	makeAbsent(t, &value, 0)
	value.Ownership[0].OwnerSubject = value.Observations[0].Subject
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)

	value = validCandidate(t)
	orphan := subjectRef{Kind: "single", RootID: "sujeto_que_no_existe"}
	value.Ownership[0].OverlapSubjects = []subjectRef{orphan}
	value.Ownership[0].PrunedFromSubjects = []subjectRef{orphan}
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}

func TestRejectsOverlapWithoutPruneAndPruneWithoutOverlap(t *testing.T) {
	for name, mutate := range map[string]func(*mappingCandidate){
		"solapamiento": func(value *mappingCandidate) {
			value.Ownership[1].OverlapSubjects = []subjectRef{value.Observations[0].Subject}
		},
		"poda": func(value *mappingCandidate) {
			value.Ownership[1].PrunedFromSubjects = []subjectRef{value.Observations[0].Subject}
		},
	} {
		t.Run(name, func(t *testing.T) {
			value := validCandidate(t)
			mutate(&value)
			resealCandidate(t, &value)
			requireCandidateFailure(t, value)
		})
	}
}

func TestRejectsOwnershipCycleAndUnorderedPrunes(t *testing.T) {
	value := validCandidate(t)
	value.Ownership[0].OverlapSubjects = []subjectRef{value.Observations[1].Subject}
	value.Ownership[0].PrunedFromSubjects = []subjectRef{value.Observations[1].Subject}
	value.Ownership[1].OverlapSubjects = []subjectRef{value.Observations[0].Subject}
	value.Ownership[1].PrunedFromSubjects = []subjectRef{value.Observations[0].Subject}
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)

	value = validCandidate(t)
	value.Ownership[2].OverlapSubjects = []subjectRef{
		value.Observations[1].Subject, value.Observations[0].Subject,
	}
	value.Ownership[2].PrunedFromSubjects = append(
		[]subjectRef(nil), value.Ownership[2].OverlapSubjects...,
	)
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}

func TestRejectsBindingCycleHiddenBehindCrossedAliases(t *testing.T) {
	value := validCandidate(t)
	for _, pair := range [][2]int{{0, 2}, {1, 3}} {
		owner, alias := &value.Observations[pair[0]], &value.Observations[pair[1]]
		alias.IdentityRef, alias.BindingRef = owner.IdentityRef, owner.BindingRef
		alias.ReopenEvidenceSHA256, alias.ObservedType = owner.ReopenEvidenceSHA256, owner.ObservedType
		value.Ownership[pair[0]].AliasSubjects = []subjectRef{alias.Subject}
	}
	value.Ownership = append(value.Ownership[:3], value.Ownership[4:]...)
	value.Ownership = append(value.Ownership[:2], value.Ownership[3:]...)
	value.Ownership[0].OverlapSubjects = []subjectRef{value.Observations[3].Subject}
	value.Ownership[0].PrunedFromSubjects = []subjectRef{value.Observations[3].Subject}
	value.Ownership[1].OverlapSubjects = []subjectRef{value.Observations[2].Subject}
	value.Ownership[1].PrunedFromSubjects = []subjectRef{value.Observations[2].Subject}
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}

func TestRejectsAliasAsSourceOfPrune(t *testing.T) {
	value := validCandidate(t)
	owner, alias := &value.Observations[0], &value.Observations[2]
	alias.IdentityRef, alias.BindingRef = owner.IdentityRef, owner.BindingRef
	alias.ReopenEvidenceSHA256, alias.ObservedType = owner.ReopenEvidenceSHA256, owner.ObservedType
	value.Ownership[0].AliasSubjects = []subjectRef{alias.Subject}
	value.Ownership = append(value.Ownership[:2], value.Ownership[3:]...)
	value.Ownership[1].OverlapSubjects = []subjectRef{alias.Subject}
	value.Ownership[1].PrunedFromSubjects = []subjectRef{alias.Subject}
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}

func TestRejectsNullOwnershipArrays(t *testing.T) {
	value := validCandidate(t)
	value.Ownership[0].AliasSubjects = nil
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}

func TestRejectsReorderedOrDuplicatedOwnership(t *testing.T) {
	value := validCandidate(t)
	value.Ownership[0], value.Ownership[1] = value.Ownership[1], value.Ownership[0]
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)

	value = validCandidate(t)
	value.Ownership[1] = value.Ownership[0]
	resealCandidate(t, &value)
	requireCandidateFailure(t, value)
}
