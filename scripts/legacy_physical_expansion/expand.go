// Este fichero valida las cardinalidades V3 y expande identidades lógicas tipadas.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

func buildExpansion(document sourceDocument, sourceDigest string) (expansionDocument, error) {
	if document.DocumentKind != "orquesta_legacy_source_roots" ||
		document.SchemaVersion != 3 || document.Closed ||
		len(document.Roots) != 122 || document.RootIDsSHA256 != expectedRootIDsSHA256 {
		return expansionDocument{}, contractFailure("v3_incompatible")
	}
	if digestSortedRootIDs(document.Roots) != expectedRootIDsSHA256 {
		return expansionDocument{}, contractFailure("raices_invalidas")
	}
	roots, err := indexRoots(document.Roots)
	if err != nil {
		return expansionDocument{}, err
	}
	collections, err := indexCollections(document.Collections)
	if err != nil {
		return expansionDocument{}, err
	}
	if len(document.BlockingPhysicalCensusRootIDs) != 112 ||
		orderedNULDigest(document.BlockingPhysicalCensusRootIDs) != expectedPendingSHA256 {
		return expansionDocument{}, contractFailure("referencias_invalidas")
	}
	result := newExpansion(sourceDigest)
	seenReferences := make(map[string]struct{}, 112)
	seenSubjects := make(map[string]struct{}, 382)
	for _, rootID := range document.BlockingPhysicalCensusRootIDs {
		root, exists := roots[rootID]
		if !exists || !root.PhysicalCensusRequired || root.PhysicalCensusStatus != "pendiente" {
			return expansionDocument{}, contractFailure("referencias_invalidas")
		}
		if _, duplicate := seenReferences[rootID]; duplicate {
			return expansionDocument{}, contractFailure("referencia_duplicada")
		}
		seenReferences[rootID] = struct{}{}
		countDecision(&result.Decisions, root.SourceDecision)
		if err := expandRoot(root, collections, &result, seenSubjects); err != nil {
			return expansionDocument{}, err
		}
	}
	if err := validateTotals(result, collections); err != nil {
		return expansionDocument{}, err
	}
	result.SubjectSetSHA256, err = domainSeparatedDigest(subjectDigestDomain, result.Subjects)
	return result, err
}

func newExpansion(sourceDigest string) expansionDocument {
	sha256Contract := func(domain string, included bool) digestContract {
		return digestContract{Algorithm: "sha256", Domain: domain, DomainInDigest: included}
	}
	return expansionDocument{
		DocumentKind:  "orquesta_legacy_physical_subject_universe",
		SchemaVersion: 1,
		SourceV3: sourceSeal{
			DocumentKind:  "orquesta_legacy_source_roots",
			SchemaVersion: 3,
			BytesSHA256:   sourceDigest,
		},
		DigestContracts: digestContracts{
			SourceV3Bytes:     sha256Contract(sourceDigestDomain, false),
			LogicalReferences: sha256Contract(pendingDigestDomain, false),
			CollectionMembers: sha256Contract(membersDigestDomain, false),
			Subjects:          sha256Contract(subjectDigestDomain, true),
		},
		LogicalReferenceSetSHA256: expectedPendingSHA256,
		Counts:                    expansionCounts{SourceRoots: 122, LogicalReferences: 112},
		MembershipSeals:           make([]membershipSeal, 0, 15),
		Subjects:                  make([]subjectRef, 0, 382),
	}
}

func indexRoots(values []sourceRoot) (map[string]sourceRoot, error) {
	index := make(map[string]sourceRoot, len(values))
	for _, value := range values {
		if value.ID == "" {
			return nil, contractFailure("raices_invalidas")
		}
		if _, duplicate := index[value.ID]; duplicate {
			return nil, contractFailure("raiz_duplicada")
		}
		index[value.ID] = value
	}
	return index, nil
}

func indexCollections(values []sourceCollection) (map[string]sourceCollection, error) {
	index := make(map[string]sourceCollection, len(values))
	for _, value := range values {
		if value.RootID == "" {
			return nil, contractFailure("colecciones_invalidas")
		}
		if _, duplicate := index[value.RootID]; duplicate {
			return nil, contractFailure("coleccion_duplicada")
		}
		index[value.RootID] = value
	}
	return index, nil
}

func expandRoot(root sourceRoot, collections map[string]sourceCollection, result *expansionDocument, seen map[string]struct{}) error {
	switch root.ScopeKind {
	case "single":
		result.Counts.SimpleReferences++
		if root.ExistsAtObservation {
			result.Counts.HistoricalPresent++
		} else {
			result.Counts.HistoricalAbsent++
		}
		return appendSubject(result, seen, subjectRef{Kind: "single", RootID: root.ID})
	case "collection":
		collection, exists := collections[root.ID]
		if !exists {
			return contractFailure("coleccion_ausente")
		}
		result.Counts.CollectionReferences++
		return expandCollection(collection, result, seen)
	default:
		return contractFailure("alcance_invalido")
	}
}

func expandCollection(collection sourceCollection, result *expansionDocument, seen map[string]struct{}) error {
	digest, err := canonicalLFDigest(collection.Members)
	if err != nil || digest != collection.MembersSHA256 {
		return contractFailure("miembros_corruptos")
	}
	aliases := make(map[string]struct{}, len(collection.Members))
	for _, member := range collection.Members {
		if member.PathAlias == "" {
			return contractFailure("alias_invalido")
		}
		if _, duplicate := aliases[member.PathAlias]; duplicate {
			return contractFailure("alias_duplicado")
		}
		aliases[member.PathAlias] = struct{}{}
		if member.ExistsAtObservation {
			result.Counts.HistoricalPresent++
		} else {
			result.Counts.HistoricalAbsent++
		}
		subject := subjectRef{Kind: "member", CollectionRootID: collection.RootID, PathAlias: member.PathAlias}
		if err := appendSubject(result, seen, subject); err != nil {
			return err
		}
	}
	result.Counts.CollectionMembers += len(collection.Members)
	result.MembershipSeals = append(result.MembershipSeals, membershipSeal{
		RootID: collection.RootID, MemberCount: len(collection.Members), MembersSHA256: digest,
	})
	return nil
}

func appendSubject(result *expansionDocument, seen map[string]struct{}, subject subjectRef) error {
	key := subject.Kind + "\x00" + subject.RootID + "\x00" + subject.CollectionRootID + "\x00" + subject.PathAlias
	if _, duplicate := seen[key]; duplicate {
		return contractFailure("sujeto_duplicado")
	}
	seen[key] = struct{}{}
	result.Subjects = append(result.Subjects, subject)
	result.Counts.PhysicalSubjects++
	return nil
}

func validateTotals(result expansionDocument, collections map[string]sourceCollection) error {
	counts, decisions := result.Counts, result.Decisions
	if len(collections) != 15 || len(result.MembershipSeals) != 15 ||
		counts.SimpleReferences != 97 || counts.CollectionReferences != 15 ||
		counts.CollectionMembers != 285 || counts.PhysicalSubjects != 382 ||
		counts.HistoricalPresent != 364 || counts.HistoricalAbsent != 18 ||
		decisions.Include != 36 || decisions.Exclude != 49 || decisions.Pending != 27 {
		return contractFailure("cardinalidades_invalidas")
	}
	return nil
}

func countDecision(counts *decisionCounts, decision string) {
	switch decision {
	case "incluir":
		counts.Include++
	case "excluir":
		counts.Exclude++
	case "pendiente":
		counts.Pending++
	}
}

func digestSortedRootIDs(roots []sourceRoot) string {
	ids := make([]string, 0, len(roots))
	for _, root := range roots {
		ids = append(ids, root.ID)
	}
	sort.Strings(ids)
	hash := sha256.New()
	for _, id := range ids {
		_, _ = hash.Write([]byte(id))
		_, _ = hash.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func encodeExpansion(value expansionDocument) ([]byte, error) {
	return canonicalJSON(value)
}
