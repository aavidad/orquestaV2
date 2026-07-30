// Este fichero decide qué blobs pueden leerse y emite sus hechos una sola vez.
// Los subárboles vendorizados se enumeran, pero no autorizan lectura de contenido.
package main

import (
	"sort"
	"strings"
)

func collectReadableBlobs(
	rootTrees, directBlobs []string,
	cache map[string][]treeEntry,
) map[string]struct{} {
	result := map[string]struct{}{}
	for _, blobID := range directBlobs {
		result[blobID] = struct{}{}
	}
	visited := map[string]struct{}{}
	var walk func(string)
	walk = func(treeID string) {
		if _, found := visited[treeID]; found {
			return
		}
		visited[treeID] = struct{}{}
		for _, entry := range cache[treeID] {
			childType := treeEntryType(entry.mode)
			if exclusionCause(entry.name, childType) != "" {
				continue
			}
			switch childType {
			case "tree":
				walk(entry.oid)
			case "blob":
				result[entry.oid] = struct{}{}
			}
		}
	}
	for _, root := range rootTrees {
		walk(root)
	}
	return result
}

func readBlobAggregates(
	batch *gitBatch,
	readable map[string]struct{},
	output *inventoryWriter,
) (map[string]*blobAggregate, error) {
	blobIDs := make([]string, 0, len(readable))
	for blobID := range readable {
		blobIDs = append(blobIDs, blobID)
	}
	sort.Strings(blobIDs)
	aggregates := make(map[string]*blobAggregate, len(blobIDs))
	for _, blobID := range blobIDs {
		facts, err := readBlobFacts(batch, blobID)
		if err != nil {
			if writeErr := output.write(blobFailureRecord(blobID, err)); writeErr != nil {
				return nil, writeErr
			}
			continue
		}
		aggregates[blobID] = newBlobAggregate(facts)
	}
	return aggregates, nil
}

func emitBlobAggregates(output *inventoryWriter, aggregates map[string]*blobAggregate) error {
	blobIDs := make([]string, 0, len(aggregates))
	for blobID := range aggregates {
		blobIDs = append(blobIDs, blobID)
	}
	sort.Strings(blobIDs)
	for _, blobID := range blobIDs {
		aggregate := aggregates[blobID]
		families := sortedFamilies(aggregate.families)
		detectedTypes := sortedSet(aggregate.detectedTypes)
		fileModes := sortedSet(aggregate.fileModes)
		summaries := sortedSet(aggregate.summaries)
		classificationRef := stableRef(
			"clasificacion-superficie",
			blobID+"\x00"+strings.Join(detectedTypes, "\x00")+"\x00"+strings.Join(fileModes, "\x00")+
				"\x00"+string(mustJSON(families)),
		)
		if err := output.write(record{
			RecordKind: "hechos_blob", BlobID: blobID,
			BlobSize: aggregate.facts.size, BlobSHA: aggregate.facts.sha256,
			ClassificationRef: classificationRef, Families: families,
			DetectedTypes: detectedTypes, FileModes: fileModes,
			Encoding: aggregate.facts.encoding, LineCount: aggregate.facts.lineCount,
			Summaries: summaries, ClassificationCount: len(aggregate.classifications),
		}); err != nil {
			return err
		}
	}
	return nil
}
