// Este fichero clasifica rutas no excluidas con un estado finito por árbol.
package main

type classificationVisit struct {
	treeID  string
	context classificationContext
}

func classifyObjectGraph(
	rootTrees []string,
	cache map[string][]treeEntry,
	aggregates map[string]*blobAggregate,
) int {
	visited := map[classificationVisit]struct{}{}
	for _, root := range rootTrees {
		classifyTree(root, 0, cache, aggregates, visited)
	}
	return len(visited)
}

func classifyTree(
	treeID string,
	context classificationContext,
	cache map[string][]treeEntry,
	aggregates map[string]*blobAggregate,
	visited map[classificationVisit]struct{},
) {
	key := classificationVisit{treeID: treeID, context: context}
	if _, found := visited[key]; found {
		return
	}
	visited[key] = struct{}{}
	for _, entry := range cache[treeID] {
		childType := treeEntryType(entry.mode)
		if exclusionCause(entry.name, childType) != "" {
			continue
		}
		childContext := extendClassificationContext(context, entry.name, childType == "tree")
		if childType == "tree" {
			classifyTree(entry.oid, childContext, cache, aggregates, visited)
			continue
		}
		aggregate := aggregates[entry.oid]
		if aggregate == nil {
			continue
		}
		classified := classifyWithContext(childContext, entry.name, aggregate.facts.prefix)
		detectedType := detectType(entry.name, entry.mode, aggregate.facts)
		addClassification(aggregate, classified, detectedType, entry.mode)
	}
}

func classifyDirectBlobs(blobIDs []string, aggregates map[string]*blobAggregate) {
	for _, blobID := range blobIDs {
		aggregate := aggregates[blobID]
		if aggregate == nil || len(aggregate.classifications) > 0 {
			continue
		}
		classified := classification{families: []surfaceFamily{{
			Family: "desconocido", Subtype: "referencia_git_sin_ruta",
		}}}
		addClassification(aggregate, classified, "objeto_git_sin_ruta", "")
	}
}

func addClassification(
	aggregate *blobAggregate,
	classified classification,
	detectedType, mode string,
) {
	summary := structuralSummary(mode, aggregate.facts, detectedType)
	for _, family := range classified.families {
		aggregate.families[family] = struct{}{}
	}
	aggregate.detectedTypes[detectedType] = struct{}{}
	if mode != "" {
		aggregate.fileModes[mode] = struct{}{}
	}
	aggregate.summaries[summary] = struct{}{}
	aggregate.classifications[classificationKey(classified.families, detectedType, mode)] = struct{}{}
}
