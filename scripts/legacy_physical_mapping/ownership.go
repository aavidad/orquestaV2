// Este fichero reconcilia alias, dueños, solapamientos y podas sin usar rutas.
package main

import "slices"

func validateOwnership(values []ownershipDeclaration, state *observationState) error {
	if values == nil || len(values) != len(state.bindingOrder) {
		return contractFailure("propiedad_incompleta")
	}
	graph := make([][]int, len(state.bindingOrder))
	bindingIndexes := make(map[string]int, len(state.bindingOrder))
	for index, binding := range state.bindingOrder {
		bindingIndexes[binding] = index
	}
	pairs := make(map[[2]int]struct{})
	for index, value := range values {
		if value.BindingRef != state.bindingOrder[index] ||
			value.AliasSubjects == nil || value.OverlapSubjects == nil ||
			value.PrunedFromSubjects == nil {
			return contractFailure("propiedad_desalineada")
		}
		group := state.groups[value.BindingRef]
		ownerIndex, exists := state.subjectIndexes[subjectKey(value.OwnerSubject)]
		if !exists || !state.present[ownerIndex] ||
			state.bindingBySubject[ownerIndex] != value.BindingRef ||
			ownerIndex != group.subjects[0] {
			return contractFailure("dueno_invalido")
		}
		expectedAliases := aliasesForGroup(group, ownerIndex, state)
		if !slices.Equal(value.AliasSubjects, expectedAliases) {
			return contractFailure("alias_no_reconciliado")
		}
		state.counts.Aliases += len(expectedAliases)
		if err := validatePrunes(value, index, ownerIndex, state, bindingIndexes, graph, pairs); err != nil {
			return err
		}
	}
	if hasCycle(graph) {
		return contractFailure("ciclo_de_propiedad")
	}
	state.counts.Bindings = len(values)
	return nil
}

func aliasesForGroup(
	group *bindingGroup,
	ownerIndex int,
	state *observationState,
) []subjectRef {
	result := make([]subjectRef, 0, len(group.subjects)-1)
	for _, subjectIndex := range group.subjects {
		if subjectIndex != ownerIndex {
			result = append(result, state.subjects[subjectIndex])
		}
	}
	return result
}

func validatePrunes(
	value ownershipDeclaration,
	ownerBindingIndex int,
	ownerIndex int,
	state *observationState,
	bindingIndexes map[string]int,
	graph [][]int,
	pairs map[[2]int]struct{},
) error {
	if !slices.Equal(value.OverlapSubjects, value.PrunedFromSubjects) {
		return contractFailure("solapamiento_no_reconciliado")
	}
	previous := -1
	for _, subject := range value.OverlapSubjects {
		subjectIndex, exists := state.subjectIndexes[subjectKey(subject)]
		if !exists || !state.present[subjectIndex] || subjectIndex == ownerIndex ||
			state.bindingBySubject[subjectIndex] == value.BindingRef ||
			subjectIndex != state.groups[state.bindingBySubject[subjectIndex]].subjects[0] ||
			subjectIndex <= previous {
			return contractFailure("poda_invalida")
		}
		sourceBindingIndex := bindingIndexes[state.bindingBySubject[subjectIndex]]
		pair := [2]int{sourceBindingIndex, ownerBindingIndex}
		if _, duplicate := pairs[pair]; duplicate {
			return contractFailure("propiedad_ambigua")
		}
		pairs[pair] = struct{}{}
		graph[sourceBindingIndex] = append(graph[sourceBindingIndex], ownerBindingIndex)
		previous = subjectIndex
		state.counts.Overlaps++
		state.counts.Prunes++
	}
	return nil
}

func hasCycle(graph [][]int) bool {
	state := make([]uint8, len(graph))
	var visit func(int) bool
	visit = func(node int) bool {
		if state[node] == 1 {
			return true
		}
		if state[node] == 2 {
			return false
		}
		state[node] = 1
		for _, next := range graph[node] {
			if visit(next) {
				return true
			}
		}
		state[node] = 2
		return false
	}
	for node := range graph {
		if state[node] == 0 && visit(node) {
			return true
		}
	}
	return false
}
