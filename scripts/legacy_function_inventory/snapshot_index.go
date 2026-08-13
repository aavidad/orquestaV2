// Este fichero genera el censo estructural reproducible desde el índice Git autorizado.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	snapshotIndexAlgorithm  = "legacy-go-function-snapshot-index.v1"
	snapshotIndexHashDomain = "orquesta.legacy-go-function-snapshot-index.v1"
	maximumSnapshotBlobSize = 4 << 20
)

type snapshotIndexEntry struct {
	Path string
	OID  string
	Mode string
}

type snapshotIndexManifest struct {
	DocumentKind            string `json:"document_kind"`
	SchemaVersion           int    `json:"schema_version"`
	Algorithm               string `json:"algorithm"`
	SourceRoot              string `json:"source_root"`
	CensusSHA256            string `json:"census_sha256"`
	GitIndexSnapshotSHA256  string `json:"git_index_snapshot_sha256"`
	JSONLSHA256             string `json:"jsonl_sha256"`
	JSONLBytes              int64  `json:"jsonl_bytes"`
	ProductionFileCount     int    `json:"production_file_count"`
	FunctionCount           int    `json:"function_count"`
	MethodCount             int    `json:"method_count"`
	DeclarationCount        int    `json:"declaration_count"`
	UniqueOccurrenceCount   int    `json:"unique_occurrence_count"`
	UniqueDeclarationCount  int    `json:"unique_declaration_count"`
	UniqueVariantCount      int    `json:"unique_variant_count"`
	ContainsBodies          bool   `json:"contains_bodies"`
	SemanticAssessmentState string `json:"semantic_assessment_state"`
	Authority               string `json:"authority"`
	CanonicalStateChange    bool   `json:"canonical_state_change"`
	ClaimsAccreditation     bool   `json:"claims_accreditation"`
}

func writeSnapshotFunctionIndex(repository, jsonlPath, manifestPath string) (runErr error) {
	repository, err := filepath.Abs(repository)
	if err != nil {
		return err
	}
	repository, err = filepath.EvalSymlinks(filepath.Clean(repository))
	if err != nil {
		return err
	}
	jsonlPath, err = normalizedOutput(repository, jsonlPath)
	if err != nil {
		return err
	}
	manifestPath, err = normalizedOutput(repository, manifestPath)
	if err != nil {
		return err
	}
	if jsonlPath == manifestPath {
		return errors.New("índice y manifiesto necesitan rutas distintas")
	}
	contract, err := loadLegacyGoCensusContract(repository)
	if err != nil {
		return err
	}
	entries, listing, err := readSnapshotIndexEntries(repository, contract)
	if err != nil {
		return err
	}
	if len(entries) != contract.Baseline.ProductionFileCount {
		return fmt.Errorf("ficheros productivos=%d, baseline=%d",
			len(entries), contract.Baseline.ProductionFileCount)
	}

	temporary, err := createTemporary(jsonlPath)
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if runErr != nil {
			_ = temporary.Close()
			_ = os.Remove(temporaryPath)
		}
	}()
	writer := bufio.NewWriterSize(temporary, 256*1024)
	hasher := sha256.New()
	multi := io.MultiWriter(writer, hasher)
	encoder := json.NewEncoder(multi)
	encoder.SetEscapeHTML(false)

	batch, err := newGitBatch(repository, nil)
	if err != nil {
		return err
	}
	cache := make(map[string]blobResult)
	occurrences := make(map[string]struct{})
	declarations := make(map[string]struct{})
	variants := make(map[string]struct{})
	functionCount := 0
	methodCount := 0
	declarationCount := 0
	for _, entry := range entries {
		parsed, ok := cache[entry.OID]
		if !ok {
			object, err := batch.get(entry.OID)
			if err != nil {
				_ = batch.close()
				return err
			}
			if object.kind != "blob" || object.oid != entry.OID ||
				len(object.content) > maximumSnapshotBlobSize {
				_ = batch.close()
				return fmt.Errorf("blob fuera de contrato: %s", entry.Path)
			}
			parsed, err = parseBlob(object.content, entry.Path)
			if err != nil || parsed.failure != nil {
				_ = batch.close()
				return fmt.Errorf("blob Go no parseable: %s", entry.Path)
			}
			cache[entry.OID] = parsed
		}
		for _, declaration := range parsed.records {
			record := buildIndexSymbolRecord(
				contract.SourceRoot, entry.Path, entry.OID, parsed, declaration,
			)
			if _, duplicate := occurrences[record.OccurrenceRef]; duplicate {
				_ = batch.close()
				return fmt.Errorf("aparición duplicada: %s", record.OccurrenceRef)
			}
			occurrences[record.OccurrenceRef] = struct{}{}
			declarations[record.DeclarationRef] = struct{}{}
			variants[record.VariantRef] = struct{}{}
			switch record.SymbolKind {
			case "func":
				functionCount++
			case "method":
				methodCount++
			default:
				_ = batch.close()
				return fmt.Errorf("clase de símbolo inesperada: %s", record.SymbolKind)
			}
			if err := encoder.Encode(record); err != nil {
				_ = batch.close()
				return err
			}
			declarationCount++
		}
	}
	if err := batch.close(); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	info, err := temporary.Stat()
	if err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	manifest := snapshotIndexManifest{
		DocumentKind:            "legacy_go_function_snapshot_index",
		SchemaVersion:           1,
		Algorithm:               snapshotIndexAlgorithm,
		SourceRoot:              contract.SourceRoot,
		CensusSHA256:            contract.Baseline.CensusSHA256,
		GitIndexSnapshotSHA256:  digest(snapshotIndexHashDomain, listing),
		JSONLSHA256:             "sha256:" + hex.EncodeToString(hasher.Sum(nil)),
		JSONLBytes:              info.Size(),
		ProductionFileCount:     len(entries),
		FunctionCount:           functionCount,
		MethodCount:             methodCount,
		DeclarationCount:        declarationCount,
		UniqueOccurrenceCount:   len(occurrences),
		UniqueDeclarationCount:  len(declarations),
		UniqueVariantCount:      len(variants),
		ContainsBodies:          false,
		SemanticAssessmentState: "structural_only_join_legacy_reuse_assessments",
		Authority:               "derived_advisory_read_model",
		CanonicalStateChange:    false,
		ClaimsAccreditation:     false,
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')
	manifestTemporary, err := writeTemporary(manifestPath, manifestBytes)
	if err != nil {
		return err
	}
	defer os.Remove(manifestTemporary)
	if err := publishPair(temporaryPath, jsonlPath, manifestTemporary, manifestPath); err != nil {
		return err
	}
	return nil
}

func readSnapshotIndexEntries(
	repository string,
	contract legacyGoCensusContract,
) ([]snapshotIndexEntry, []byte, error) {
	for _, arguments := range [][]string{
		{"diff", "--name-only", "-z", "--", contract.SourceRoot},
		{"ls-files", "--others", "--exclude-standard", "-z", "--", contract.SourceRoot},
	} {
		output, err := gitBytes(repository, nil, arguments...)
		if err != nil {
			return nil, nil, err
		}
		if len(output) != 0 {
			return nil, nil, errors.New("el source_root contiene overrides fuera del índice")
		}
	}
	listing, err := gitBytes(repository, nil, "ls-files", "--stage", "-z", "--", contract.SourceRoot)
	if err != nil {
		return nil, nil, err
	}
	var entries []snapshotIndexEntry
	seen := make(map[string]struct{})
	for _, raw := range bytes.Split(listing, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		header, rawPath, found := bytes.Cut(raw, []byte{'\t'})
		fields := strings.Fields(string(header))
		path := string(rawPath)
		if !found || len(fields) != 3 || !utf8.ValidString(path) ||
			(fields[0] != "100644" && fields[0] != "100755") || fields[2] != "0" {
			return nil, nil, fmt.Errorf("entrada de índice inválida: %q", raw)
		}
		if !validLookupSourcePath(path, contract) {
			continue
		}
		if _, duplicate := seen[path]; duplicate {
			return nil, nil, fmt.Errorf("ruta productiva duplicada: %s", path)
		}
		seen[path] = struct{}{}
		entries = append(entries, snapshotIndexEntry{Path: path, OID: fields[1], Mode: fields[0]})
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].Path < entries[right].Path })
	var acceptedListing bytes.Buffer
	for _, entry := range entries {
		acceptedListing.WriteString(entry.Mode)
		acceptedListing.WriteByte('\t')
		acceptedListing.WriteString(entry.OID)
		acceptedListing.WriteByte('\t')
		acceptedListing.WriteString(entry.Path)
		acceptedListing.WriteByte(0)
	}
	return entries, acceptedListing.Bytes(), nil
}
