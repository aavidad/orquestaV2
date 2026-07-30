// Este fichero normaliza agregados, tipos y fallos del grafo de objetos.
package main

import (
	"encoding/base64"
	"sort"
)

func newBlobAggregate(facts blobFacts) *blobAggregate {
	return &blobAggregate{
		facts: facts, families: map[surfaceFamily]struct{}{},
		detectedTypes: map[string]struct{}{}, fileModes: map[string]struct{}{},
		summaries: map[string]struct{}{}, classifications: map[string]struct{}{},
	}
}

func treeEntryType(mode string) string {
	switch mode {
	case "40000", "040000":
		return "tree"
	case "160000":
		return "commit"
	default:
		return "blob"
	}
}

func sortedFamilies(values map[surfaceFamily]struct{}) []surfaceFamily {
	result := make([]surfaceFamily, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Family != result[right].Family {
			return result[left].Family < result[right].Family
		}
		return result[left].Subtype < result[right].Subtype
	})
	return result
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func classificationKey(families []surfaceFamily, detectedType, mode string) string {
	return base64.StdEncoding.EncodeToString(mustJSON(families)) +
		"\x00" + detectedType + "\x00" + mode
}

func treeFailureRecord(treeID string, err error) record {
	return record{
		RecordKind: "fallo", TreeID: treeID,
		ErrorCode: "arbol_no_legible", ErrorDetail: boundedDetail(err.Error()),
	}
}

func blobFailureRecord(blobID string, err error) record {
	return record{
		RecordKind: "fallo", BlobID: blobID,
		ErrorCode: "blob_no_legible", ErrorDetail: boundedDetail(err.Error()),
	}
}
