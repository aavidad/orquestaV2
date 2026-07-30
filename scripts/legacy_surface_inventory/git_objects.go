// Este fichero emite árboles, entradas y blobs una sola vez por objeto Git.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

func emitObjectGraph(
	batch *gitBatch,
	rootTrees []string,
	objectBytes int,
	output *inventoryWriter,
) (map[string][]treeEntry, error) {
	cache := map[string][]treeEntry{}
	queued := map[string]struct{}{}
	pending := make([]string, 0, len(rootTrees))
	for _, tree := range rootTrees {
		if _, exists := queued[tree]; !exists {
			queued[tree] = struct{}{}
			pending = append(pending, tree)
		}
	}
	for index := 0; index < len(pending); index++ {
		treeID := pending[index]
		entries, err := readTree(batch, treeID, objectBytes, cache)
		if err != nil {
			if writeErr := output.write(treeFailureRecord(treeID, err)); writeErr != nil {
				return nil, writeErr
			}
			continue
		}
		if err := output.write(record{RecordKind: "objeto_arbol", TreeID: treeID}); err != nil {
			return nil, err
		}
		for _, entry := range entries {
			encoding, segment, encoded := displayGitPath(entry.name)
			childType := treeEntryType(entry.mode)
			cause := exclusionCause(entry.name, childType)
			obligation := ""
			if cause != "" {
				obligation = vendoredReviewDuty
			}
			if err := output.write(record{
				RecordKind: "entrada_arbol", TreeID: treeID,
				PathEncoding: encoding, PathSegment: segment, PathSegmentBase64: encoded,
				FileMode: entry.mode, ChildType: childType, ChildObjectID: entry.oid,
				ExclusionCause: cause, ReviewObligation: obligation,
			}); err != nil {
				return nil, err
			}
			if childType == "tree" {
				if _, exists := queued[entry.oid]; !exists {
					queued[entry.oid] = struct{}{}
					pending = append(pending, entry.oid)
				}
				continue
			}
		}
	}
	return cache, nil
}

func readTree(batch *gitBatch, oid string, objectBytes int, cache map[string][]treeEntry) ([]treeEntry, error) {
	if cached, found := cache[oid]; found {
		return cached, nil
	}
	header, err := batch.header(oid)
	if err != nil {
		return nil, err
	}
	if header.missing {
		return nil, fmt.Errorf("objeto %s ausente", oid)
	}
	if header.kind != "tree" {
		if err := batch.discard(header.size); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("objeto %s es %s, no tree", oid, header.kind)
	}
	if header.size > int64(^uint(0)>>1) {
		return nil, errors.New("árbol demasiado grande")
	}
	content := make([]byte, int(header.size))
	if err := batch.readContent(content); err != nil {
		return nil, err
	}
	var result []treeEntry
	for cursor := 0; cursor < len(content); {
		space := bytes.IndexByte(content[cursor:], ' ')
		if space < 0 {
			return nil, errors.New("entrada de árbol sin modo")
		}
		space += cursor
		zero := bytes.IndexByte(content[space+1:], 0)
		if zero < 0 {
			return nil, errors.New("entrada de árbol sin terminador")
		}
		zero += space + 1
		oidStart := zero + 1
		oidEnd := oidStart + objectBytes
		if oidEnd > len(content) {
			return nil, errors.New("entrada de árbol con identificador truncado")
		}
		result = append(result, treeEntry{
			mode: string(content[cursor:space]),
			name: bytes.Clone(content[space+1 : zero]),
			oid:  hex.EncodeToString(content[oidStart:oidEnd]),
		})
		cursor = oidEnd
	}
	cache[oid] = result
	return result, nil
}

func readBlobFacts(batch *gitBatch, oid string) (blobFacts, error) {
	header, err := batch.header(oid)
	if err != nil {
		return blobFacts{}, err
	}
	if header.missing {
		return blobFacts{}, fmt.Errorf("objeto %s ausente", oid)
	}
	if header.kind != "blob" {
		if err := batch.discard(header.size); err != nil {
			return blobFacts{}, err
		}
		return blobFacts{}, fmt.Errorf("objeto %s es %s, no blob", oid, header.kind)
	}
	hasher := sha256.New()
	validator := utf8Validator{valid: true}
	facts := blobFacts{size: header.size}
	remaining := header.size
	buffer := make([]byte, 64*1024)
	var containsZero bool
	for remaining > 0 {
		toRead := int64(len(buffer))
		if remaining < toRead {
			toRead = remaining
		}
		part := buffer[:toRead]
		if _, err := io.ReadFull(batch.output, part); err != nil {
			return blobFacts{}, err
		}
		_, _ = hasher.Write(part)
		validator.write(part)
		containsZero = containsZero || bytes.IndexByte(part, 0) >= 0
		facts.lineCount += int64(bytes.Count(part, []byte{'\n'}))
		facts.trailingByte = part[len(part)-1]
		if len(facts.prefix) < blobPrefixLimit {
			needed := min(blobPrefixLimit-len(facts.prefix), len(part))
			facts.prefix = append(facts.prefix, part[:needed]...)
		}
		remaining -= toRead
	}
	if err := batch.consumeDelimiter(); err != nil {
		return blobFacts{}, err
	}
	if header.size > 0 && facts.trailingByte != '\n' {
		facts.lineCount++
	}
	if containsZero || !validator.complete() {
		facts.encoding = "binario"
		facts.lineCount = 0
	} else {
		facts.encoding = "utf-8"
	}
	facts.sha256 = hex.EncodeToString(hasher.Sum(nil))
	return facts, nil
}

func (validator *utf8Validator) write(part []byte) {
	if !validator.valid {
		return
	}
	data := append(append([]byte{}, validator.carry...), part...)
	validator.carry = validator.carry[:0]
	for len(data) > 0 {
		if !utf8.FullRune(data) {
			validator.carry = append(validator.carry, data...)
			return
		}
		runeValue, size := utf8.DecodeRune(data)
		if runeValue == utf8.RuneError && size == 1 {
			validator.valid = false
			return
		}
		data = data[size:]
	}
}

func (validator *utf8Validator) complete() bool {
	return validator.valid && len(validator.carry) == 0
}
