// Este fichero expone una consulta acotada de símbolos sobre el índice Git.
// Reutiliza el parser V4 y nunca recorre historia ni publica cuerpos.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const occurrenceHashDomain = "orquesta.legacy-go-function-occurrence.v1"
const legacyGoCensusPath = "product/traceability/legacy_go.json"

type legacyGoCensusContract struct {
	DocumentKind  string `json:"document_kind"`
	SchemaVersion int    `json:"schema_version"`
	SourceRoot    string `json:"source_root"`
	SourcePolicy  struct {
		Include           string                   `json:"include"`
		ExcludeTestSuffix string                   `json:"exclude_test_suffix"`
		ExactExclusions   []legacyGoExactExclusion `json:"exact_exclusions"`
		SymbolKinds       []string                 `json:"symbol_kinds"`
	} `json:"source_policy"`
	Baseline struct {
		ModuleCount         int    `json:"module_count"`
		PackageCount        int    `json:"package_count"`
		ProductionFileCount int    `json:"production_file_count"`
		SymbolCount         int    `json:"symbol_count"`
		ModuleSetSHA256     string `json:"module_set_sha256"`
		CensusSHA256        string `json:"census_sha256"`
	} `json:"baseline"`
	ModuleRules []legacyGoModuleRule `json:"module_rules"`
}

type legacyGoExactExclusion struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

type legacyGoModuleRule struct {
	ID                       string   `json:"id"`
	Paths                    []string `json:"paths"`
	Disposition              string   `json:"disposition"`
	CapabilityIDs            []string `json:"capability_ids"`
	Reason                   string   `json:"reason"`
	CharacterizationRequired bool     `json:"characterization_required"`
}

type indexSymbolRecord struct {
	SchemaVersion  int    `json:"schema_version"`
	OccurrenceRef  string `json:"occurrence_ref"`
	DeclarationRef string `json:"declaration_ref"`
	VariantRef     string `json:"variant_ref"`
	SourceRoot     string `json:"source_root"`
	SourcePath     string `json:"source_path"`
	GitBlobOID     string `json:"git_blob_oid"`
	BlobDigest     string `json:"blob_digest"`
	PackagePath    string `json:"package_path"`
	PackageName    string `json:"package_name"`
	SymbolKind     string `json:"symbol_kind"`
	Name           string `json:"name"`
	Receiver       string `json:"receiver,omitempty"`
	Signature      string `json:"signature"`
	SourceSHA      string `json:"source_sha256"`
	ASTSHA         string `json:"ast_sha256"`
	BodySHA        string `json:"body_sha256,omitempty"`
	StartLine      int    `json:"start_line"`
	EndLine        int    `json:"end_line"`
}

func lookupIndexSymbols(repository, sourcePath, symbol string) ([]indexSymbolRecord, error) {
	repository, err := filepath.Abs(repository)
	if err != nil {
		return nil, err
	}
	repository, err = filepath.EvalSymlinks(filepath.Clean(repository))
	if err != nil {
		return nil, fmt.Errorf("resolver repositorio: %w", err)
	}
	contract, err := loadLegacyGoCensusContract(repository)
	if err != nil {
		return nil, err
	}
	sourcePath = filepath.ToSlash(filepath.Clean(sourcePath))
	if !validLookupSourcePath(sourcePath, contract) {
		return nil, fmt.Errorf("ruta Go productiva legacy inválida: %q", sourcePath)
	}
	if strings.TrimSpace(symbol) != symbol || strings.ContainsAny(symbol, "\r\n") {
		return nil, errors.New("símbolo de consulta inválido")
	}
	if err := rejectLookupOverrides(repository, sourcePath); err != nil {
		return nil, err
	}
	object, err := readIndexBlob(repository, sourcePath)
	if err != nil {
		return nil, err
	}
	parsed, err := parseBlob(object.content, sourcePath)
	if err != nil {
		return nil, err
	}
	if parsed.failure != nil {
		return nil, fmt.Errorf("blob Go no parseable para %s: %s", sourcePath, parsed.failure.ErrorCode)
	}
	result := make([]indexSymbolRecord, 0, len(parsed.records))
	for _, declaration := range parsed.records {
		if symbol != "" && declaration.Name != symbol {
			continue
		}
		result = append(result, buildIndexSymbolRecord(
			contract.SourceRoot, sourcePath, object.oid, parsed, declaration,
		))
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("ninguna declaración coincide con %q en %s", symbol, sourcePath)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Name != result[right].Name {
			return result[left].Name < result[right].Name
		}
		if result[left].Receiver != result[right].Receiver {
			return result[left].Receiver < result[right].Receiver
		}
		return result[left].StartLine < result[right].StartLine
	})
	return result, nil
}

func buildIndexSymbolRecord(
	sourceRoot, sourcePath, oid string,
	parsed blobResult,
	declaration record,
) indexSymbolRecord {
	declarationRef := digestStrings(
		"orquesta.legacy-go-declaration.v2",
		oid,
		strconv.Itoa(declaration.StartOffset),
		strconv.Itoa(declaration.EndOffset),
		declaration.VariantRef,
	)
	return indexSymbolRecord{
		SchemaVersion:  1,
		OccurrenceRef:  digestStrings(occurrenceHashDomain, sourceRoot, sourcePath, declarationRef),
		DeclarationRef: declarationRef,
		VariantRef:     declaration.VariantRef,
		SourceRoot:     sourceRoot,
		SourcePath:     sourcePath,
		GitBlobOID:     oid,
		BlobDigest:     parsed.sha,
		PackagePath:    filepath.ToSlash(filepath.Dir(sourcePath)),
		PackageName:    parsed.packageName,
		SymbolKind:     declaration.SymbolKind,
		Name:           declaration.Name,
		Receiver:       declaration.Receiver,
		Signature:      declaration.Signature,
		SourceSHA:      declaration.SourceSHA,
		ASTSHA:         declaration.ASTSHA,
		BodySHA:        declaration.BodySHA,
		StartLine:      declaration.StartLine,
		EndLine:        declaration.EndLine,
	}
}

func validLookupSourcePath(path string, contract legacyGoCensusContract) bool {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}
	if !strings.HasSuffix(path, ".go") ||
		strings.HasSuffix(path, contract.SourcePolicy.ExcludeTestSuffix) ||
		legacyGoPathExcluded(path, contract.SourcePolicy.ExactExclusions) {
		return false
	}
	for _, rule := range contract.ModuleRules {
		for _, modulePath := range rule.Paths {
			if strings.HasPrefix(path, modulePath+"/") {
				return true
			}
		}
	}
	return false
}

func legacyGoPathExcluded(path string, exclusions []legacyGoExactExclusion) bool {
	for _, exclusion := range exclusions {
		if exclusion.Kind == "file" && path == exclusion.Path {
			return true
		}
		if exclusion.Kind == "subtree" &&
			(path == exclusion.Path || strings.HasPrefix(path, exclusion.Path+"/")) {
			return true
		}
	}
	return false
}

func loadLegacyGoCensusContract(repository string) (legacyGoCensusContract, error) {
	object, err := readIndexBlob(repository, legacyGoCensusPath)
	if err != nil {
		return legacyGoCensusContract{}, err
	}
	var contract legacyGoCensusContract
	if err := decodeUniqueStrictJSON(object.content, &contract); err != nil {
		return legacyGoCensusContract{}, fmt.Errorf("contrato de censo inválido: %w", err)
	}
	if contract.DocumentKind != "legacy_go_census" || contract.SchemaVersion != 1 ||
		contract.SourceRoot == "" ||
		contract.SourcePolicy.Include != "all_descendant_go_files_of_exact_module_paths" ||
		contract.SourcePolicy.ExcludeTestSuffix != "_test.go" ||
		contract.Baseline.CensusSHA256 == "" || len(contract.ModuleRules) == 0 {
		return legacyGoCensusContract{}, errors.New("contrato de censo no soportado")
	}
	return contract, nil
}

func readIndexBlob(repository, path string) (gitObject, error) {
	if err := rejectLookupOverrides(repository, path); err != nil {
		return gitObject{}, err
	}
	listing, err := gitBytes(repository, nil, "ls-files", "--stage", "-z", "--", path)
	if err != nil {
		return gitObject{}, err
	}
	oid, err := exactIndexBlobOID(listing, path)
	if err != nil {
		return gitObject{}, err
	}
	batch, err := newGitBatch(repository, nil)
	if err != nil {
		return gitObject{}, err
	}
	object, readErr := batch.get(oid)
	closeErr := batch.close()
	if readErr != nil {
		return gitObject{}, readErr
	}
	if closeErr != nil {
		return gitObject{}, closeErr
	}
	if object.kind != "blob" || object.oid != oid {
		return gitObject{}, fmt.Errorf("objeto indexado inesperado para %s", path)
	}
	return object, nil
}

func decodeUniqueStrictJSON(raw []byte, destination any) error {
	unique := json.NewDecoder(bytes.NewReader(raw))
	if err := readUniqueJSONValue(unique); err != nil {
		return err
	}
	if token, err := unique.Token(); err != io.EOF {
		return fmt.Errorf("token JSON posterior: token=%v err=%v", token, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("contenido JSON posterior")
	}
	return nil
}

func readUniqueJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return fmt.Errorf("delimitador JSON inesperado: %q", delimiter)
	}
	seen := make(map[string]struct{})
	for decoder.More() {
		if delimiter == '{' {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("clave JSON no textual")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("clave JSON duplicada: %s", key)
			}
			seen[key] = struct{}{}
		}
		if err := readUniqueJSONValue(decoder); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	want := json.Delim('}')
	if delimiter == '[' {
		want = ']'
	}
	if closing != want {
		return fmt.Errorf("cierre JSON inesperado: %q", closing)
	}
	return nil
}

func rejectLookupOverrides(repository, sourcePath string) error {
	for _, args := range [][]string{
		{"diff", "--name-only", "-z", "--", sourcePath},
		{"ls-files", "--others", "--exclude-standard", "-z", "--", sourcePath},
	} {
		output, err := gitBytes(repository, nil, args...)
		if err != nil {
			return err
		}
		if len(output) != 0 {
			return fmt.Errorf("la ruta tiene overrides fuera del índice: %s", sourcePath)
		}
	}
	return nil
}

func exactIndexBlobOID(listing []byte, sourcePath string) (string, error) {
	records := bytes.Split(listing, []byte{0})
	var oid string
	for _, raw := range records {
		if len(raw) == 0 {
			continue
		}
		header, path, found := bytes.Cut(raw, []byte{'\t'})
		fields := strings.Fields(string(header))
		if !found || string(path) != sourcePath || len(fields) != 3 ||
			fields[0] != "100644" && fields[0] != "100755" || fields[2] != "0" {
			return "", fmt.Errorf("entrada de índice ambigua para %s", sourcePath)
		}
		if oid != "" {
			return "", fmt.Errorf("entrada de índice duplicada para %s", sourcePath)
		}
		oid = fields[1]
	}
	if oid == "" {
		return "", fmt.Errorf("ruta ausente del índice: %s", sourcePath)
	}
	return oid, nil
}
