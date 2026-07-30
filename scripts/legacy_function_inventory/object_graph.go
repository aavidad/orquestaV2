// Este fichero emite árboles, entradas, blobs y declaraciones una sola vez
// por identidad de objeto, conservando aristas suficientes para reconstrucción.
package main

import (
	"bytes"
	"fmt"
	"strconv"
)

func emitObjectGraph(
	batch *gitBatch,
	commits []string,
	history map[string]historyEntry,
	objectBytes int,
	stream *inventoryStream,
) error {
	pendingTrees := make([]string, 0, len(commits))
	queuedTrees := make(map[string]struct{})
	for _, commit := range commits {
		tree := history[commit].tree
		if _, exists := queuedTrees[tree]; !exists {
			queuedTrees[tree] = struct{}{}
			pendingTrees = append(pendingTrees, tree)
		}
	}
	seenGoBlobs := make(map[string]struct{})
	variants := make(map[string]string)
	for index := 0; index < len(pendingTrees); index++ {
		treeID := pendingTrees[index]
		entries, err := readTreeObject(batch, treeID, objectBytes)
		if err != nil {
			return fmt.Errorf("árbol %s: %w", treeID, err)
		}
		if err := stream.emit(record{RecordKind: "tree_object", TreeID: treeID}); err != nil {
			return err
		}
		for _, entry := range entries {
			encoding, text, encoded := encodePathSegment(entry.Path)
			if err := stream.emit(record{
				RecordKind: "tree_entry", TreeID: treeID,
				PathEncoding: encoding, PathSegment: text, PathSegmentBase64: encoded,
				FileMode: entry.Mode, ChildType: entry.Type, ChildObjectID: entry.OID,
			}); err != nil {
				return err
			}
			if entry.Type == "tree" {
				if _, exists := queuedTrees[entry.OID]; !exists {
					queuedTrees[entry.OID] = struct{}{}
					pendingTrees = append(pendingTrees, entry.OID)
				}
				continue
			}
			if entry.Type != "blob" || !bytes.HasSuffix(entry.Path, []byte(".go")) {
				continue
			}
			if _, exists := seenGoBlobs[entry.OID]; exists {
				continue
			}
			seenGoBlobs[entry.OID] = struct{}{}
			if err := emitGoBlob(batch, entry.OID, stream, variants); err != nil {
				return err
			}
		}
	}
	return nil
}

func emitGoBlob(
	batch *gitBatch,
	blobID string,
	stream *inventoryStream,
	variants map[string]string,
) error {
	object, err := batch.get(blobID)
	if err != nil {
		return err
	}
	if object.kind != "blob" {
		return fmt.Errorf("objeto Go %s no es un blob", blobID)
	}
	result, err := parseBlob(object.content, "historical.go")
	if err != nil {
		return err
	}
	if err := stream.emit(record{
		RecordKind: "go_blob", BlobID: blobID, BlobSize: result.size, BlobSHA: result.sha,
		Package: result.packageName, BuildConstraints: result.buildConstraints,
		Generated: result.generated,
	}); err != nil {
		return err
	}
	if result.failure != nil {
		failure := *result.failure
		failure.BlobID = blobID
		return stream.emit(failure)
	}
	for _, parsed := range result.records {
		declaration := parsed
		declaration.RecordKind = "go_declaration"
		declaration.BlobID = blobID
		declaration.DeclarationRef = digestStrings(
			"orquesta.legacy-go-declaration.v2",
			blobID, strconv.Itoa(parsed.StartOffset), strconv.Itoa(parsed.EndOffset), parsed.VariantRef,
		)
		canonical := declaration.CanonicalSource
		declaration.CanonicalSource = ""
		if err := stream.emit(declaration); err != nil {
			return err
		}
		if previous, exists := variants[parsed.VariantRef]; exists {
			if previous != canonical {
				return fmt.Errorf("colisión de variante %s", parsed.VariantRef)
			}
			continue
		}
		variants[parsed.VariantRef] = canonical
		variant := parsed
		variant.RecordKind = "function_variant"
		variant.BlobID = blobID
		variant.DeclarationRef = declaration.DeclarationRef
		if err := stream.emit(variant); err != nil {
			return err
		}
	}
	return nil
}
