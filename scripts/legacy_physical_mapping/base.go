// Este fichero deriva el orden histórico desde dos bases fijadas por bytes.
package main

import "encoding/json"

type baseFacts struct {
	subjects   []subjectRef
	historical []bool
}

func validateBases(v3Raw, universeRaw []byte) (baseFacts, error) {
	var source sourceDocument
	if err := decodePinned(v3Raw, expectedV3SHA, &source); err != nil {
		return baseFacts{}, err
	}
	var universe universeDocument
	if err := decodePinned(universeRaw, expectedUniverseSHA, &universe); err != nil {
		return baseFacts{}, err
	}
	if len(source.Roots) != 122 || len(source.BlockingPhysicalCensusRootIDs) != 112 ||
		len(source.Collections) != 15 || len(universe.Subjects) != 382 {
		return baseFacts{}, contractFailure("base_incompatible")
	}
	roots := make(map[string]sourceRoot, len(source.Roots))
	for _, root := range source.Roots {
		roots[root.ID] = root
	}
	collections := make(map[string]sourceCollection, len(source.Collections))
	for _, collection := range source.Collections {
		collections[collection.RootID] = collection
	}
	facts, counts, decisions, err := deriveBase(source.BlockingPhysicalCensusRootIDs, roots, collections)
	if err != nil {
		return baseFacts{}, err
	}
	if counts != [6]int{97, 15, 285, 382, 364, 18} ||
		decisions != [3]int{36, 49, 27} {
		return baseFacts{}, contractFailure("cardinalidades_invalidas")
	}
	for index := range facts.subjects {
		if facts.subjects[index] != universe.Subjects[index] {
			return baseFacts{}, contractFailure("orden_sujetos_invalido")
		}
	}
	return facts, nil
}

func deriveBase(
	references []string,
	roots map[string]sourceRoot,
	collections map[string]sourceCollection,
) (baseFacts, [6]int, [3]int, error) {
	facts := baseFacts{subjects: make([]subjectRef, 0, 382), historical: make([]bool, 0, 382)}
	var counts [6]int
	var decisions [3]int
	seen := make(map[string]struct{}, 112)
	for _, rootID := range references {
		root, exists := roots[rootID]
		if !exists {
			return facts, counts, decisions, contractFailure("referencia_invalida")
		}
		if _, duplicate := seen[rootID]; duplicate {
			return facts, counts, decisions, contractFailure("referencia_duplicada")
		}
		seen[rootID] = struct{}{}
		decisionIndex := map[string]int{"incluir": 0, "excluir": 1, "pendiente": 2}
		index, known := decisionIndex[root.SourceDecision]
		if !known {
			return facts, counts, decisions, contractFailure("decision_invalida")
		}
		decisions[index]++
		switch root.ScopeKind {
		case "single":
			counts[0]++
			facts.subjects = append(facts.subjects, subjectRef{Kind: "single", RootID: root.ID})
			facts.historical = append(facts.historical, root.ExistsAtObservation)
		case "collection":
			counts[1]++
			collection, found := collections[root.ID]
			if !found || len(collection.Members) > maxArrayItems {
				return facts, counts, decisions, contractFailure("membresia_invalida")
			}
			for _, member := range collection.Members {
				if member.PathAlias == "" {
					return facts, counts, decisions, contractFailure("membresia_invalida")
				}
				facts.subjects = append(facts.subjects, subjectRef{
					Kind: "member", CollectionRootID: collection.RootID, PathAlias: member.PathAlias,
				})
				facts.historical = append(facts.historical, member.ExistsAtObservation)
			}
			counts[2] += len(collection.Members)
		default:
			return facts, counts, decisions, contractFailure("alcance_invalido")
		}
	}
	counts[3] = len(facts.subjects)
	for _, present := range facts.historical {
		if present {
			counts[4]++
		} else {
			counts[5]++
		}
	}
	return facts, counts, decisions, nil
}

func subjectKey(value subjectRef) string { raw, _ := json.Marshal(value); return string(raw) }
