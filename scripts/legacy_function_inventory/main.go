// El inventario histórico recorre objetos Git sin materializar árboles de trabajo.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const schemaVersion = 1

var errGitObjectMissing = errors.New("objeto Git ausente")

type options struct {
	repository string
	jsonl      string
	manifest   string

	gitProcessStarted func()
}

type record struct {
	RecordKind string `json:"record_kind"`

	RefName  string   `json:"ref_name,omitempty"`
	ObjectID string   `json:"object_id,omitempty"`
	CommitID string   `json:"commit_id,omitempty"`
	TreeID   string   `json:"tree_id,omitempty"`
	Parents  []string `json:"parents,omitempty"`

	Path     string `json:"path,omitempty"`
	FileMode string `json:"file_mode,omitempty"`
	BlobID   string `json:"blob_id,omitempty"`
	BlobSize int    `json:"blob_size,omitempty"`
	BlobSHA  string `json:"blob_sha256,omitempty"`

	Package          string   `json:"package,omitempty"`
	BuildConstraints []string `json:"build_constraints,omitempty"`
	TestFile         bool     `json:"test_file,omitempty"`
	Generated        bool     `json:"generated,omitempty"`

	SymbolKind      string `json:"symbol_kind,omitempty"`
	Name            string `json:"name,omitempty"`
	Exported        bool   `json:"exported,omitempty"`
	Receiver        string `json:"receiver,omitempty"`
	Signature       string `json:"signature,omitempty"`
	StartOffset     int    `json:"start_offset,omitempty"`
	EndOffset       int    `json:"end_offset,omitempty"`
	StartLine       int    `json:"start_line,omitempty"`
	EndLine         int    `json:"end_line,omitempty"`
	SourceSHA       string `json:"source_sha256,omitempty"`
	DocSHA          string `json:"doc_sha256,omitempty"`
	ASTCanonical    string `json:"ast_canonical_version,omitempty"`
	ASTSHA          string `json:"ast_sha256,omitempty"`
	BodySHA         string `json:"body_sha256,omitempty"`
	VariantRef      string `json:"variant_ref,omitempty"`
	OccurrenceRef   string `json:"occurrence_ref,omitempty"`
	CanonicalSource string `json:"canonical_source,omitempty"`

	ErrorCode   string `json:"error_code,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

type manifest struct {
	SchemaVersion     int            `json:"schema_version"`
	Algorithm         string         `json:"algorithm"`
	Repository        string         `json:"repository"`
	GitObjectFormat   string         `json:"git_object_format"`
	GoVersion         string         `json:"go_version"`
	InventorySHA256   string         `json:"inventory_sha256"`
	Counts            map[string]int `json:"counts"`
	RefSnapshotSHA256 string         `json:"ref_snapshot_sha256"`
}

type refInfo struct {
	Name   string
	Object string
	Commit string
}

type treeEntry struct {
	Mode string
	Type string
	OID  string
	Path string
}

type blobResult struct {
	size    int
	sha     string
	records []record
	failure *record
}

type historyEntry struct {
	tree    string
	parents []string
}

type gitBatch struct {
	command *exec.Cmd
	input   io.WriteCloser
	output  *bufio.Reader
	stderr  bytes.Buffer
	closed  bool
}

type gitObject struct {
	oid     string
	kind    string
	content []byte
}

func main() {
	var options options
	flag.StringVar(&options.repository, "repo", ".", "repositorio Git que se censará")
	flag.StringVar(&options.jsonl, "jsonl", "", "salida JSONL obligatoria")
	flag.StringVar(&options.manifest, "manifest", "", "manifiesto JSON obligatorio")
	flag.Parse()
	if options.jsonl == "" || options.manifest == "" {
		fatal(errors.New("debes indicar --jsonl y --manifest"))
	}
	if err := run(options); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "legacy_function_inventory:", err)
	os.Exit(1)
}

func run(options options) error {
	repository, err := filepath.Abs(options.repository)
	if err != nil {
		return err
	}
	refsBefore, err := readRefs(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	refDigest := digestJSON("orquesta.legacy-function-inventory.refs.v1", refsBefore)
	history, err := readHistory(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	if len(history) == 0 {
		return errors.New("el repositorio no tiene confirmaciones alcanzables")
	}
	commits := make([]string, 0, len(history))
	for commit := range history {
		commits = append(commits, commit)
	}
	sort.Strings(commits)

	records := make([]record, 0, len(commits)*8)
	counts := map[string]int{}
	for _, ref := range refsBefore {
		records = append(records, record{
			RecordKind: "ref", RefName: ref.Name, ObjectID: ref.Object, CommitID: ref.Commit,
		})
		reachable, err := reachableCommits(ref.Commit, history)
		if err != nil {
			return err
		}
		for _, commit := range reachable {
			records = append(records, record{
				RecordKind: "commit_ref_reachability", RefName: ref.Name, CommitID: commit,
			})
		}
	}

	objectFormat, err := gitText(
		repository,
		options.gitProcessStarted,
		"rev-parse",
		"--show-object-format",
	)
	if err != nil {
		return err
	}
	objectBytes, err := objectIDBytes(objectFormat)
	if err != nil {
		return err
	}
	batch, err := newGitBatch(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	defer batch.close()

	seenTrees := map[string]struct{}{}
	treeCache := map[string][]treeEntry{}
	seenBlobs := map[string]blobResult{}
	variants := map[string]record{}
	for _, commit := range commits {
		header := history[commit]
		records = append(records, record{
			RecordKind: "commit", CommitID: commit, TreeID: header.tree, Parents: header.parents,
		})
		if _, found := seenTrees[header.tree]; !found {
			seenTrees[header.tree] = struct{}{}
			records = append(records, record{RecordKind: "tree", TreeID: header.tree})
		}
		entries, err := readTree(batch, header.tree, objectBytes, treeCache)
		if err != nil {
			return fmt.Errorf("árbol %s: %w", header.tree, err)
		}
		for _, entry := range entries {
			if entry.Type == "tree" {
				if _, found := seenTrees[entry.OID]; !found {
					seenTrees[entry.OID] = struct{}{}
					records = append(records, record{RecordKind: "tree", TreeID: entry.OID})
				}
				continue
			}
			if entry.Type != "blob" || !strings.HasSuffix(entry.Path, ".go") {
				continue
			}
			result, found := seenBlobs[entry.OID]
			if !found {
				object, err := batch.get(entry.OID)
				if err != nil {
					return err
				}
				if object.kind != "blob" {
					return fmt.Errorf("objeto %s de %s no es un blob", entry.OID, entry.Path)
				}
				result, err = parseBlob(object.content, entry.Path)
				if err != nil {
					return err
				}
				seenBlobs[entry.OID] = result
				blobRecord := record{
					RecordKind: "go_blob", BlobID: entry.OID, BlobSize: result.size,
					BlobSHA: result.sha,
				}
				records = append(records, blobRecord)
				if result.failure != nil {
					failure := *result.failure
					failure.BlobID = entry.OID
					records = append(records, failure)
				}
				for _, symbol := range result.records {
					if previous, exists := variants[symbol.VariantRef]; exists &&
						previous.CanonicalSource != symbol.CanonicalSource {
						return fmt.Errorf("colisión de variante %s", symbol.VariantRef)
					}
					if _, exists := variants[symbol.VariantRef]; !exists {
						variant := symbol
						variant.RecordKind = "function_variant"
						variant.Path = ""
						variant.TestFile = false
						variant.StartOffset, variant.EndOffset = 0, 0
						variant.StartLine, variant.EndLine = 0, 0
						variants[symbol.VariantRef] = variant
					}
				}
			}
			if result.failure != nil {
				records = append(records, record{
					RecordKind: "parse_failure_occurrence", CommitID: commit, TreeID: header.tree,
					Path: entry.Path, FileMode: entry.Mode, BlobID: entry.OID,
					ErrorCode: result.failure.ErrorCode,
				})
				continue
			}
			for _, symbol := range result.records {
				occurrence := symbol
				occurrence.RecordKind = "function_occurrence"
				occurrence.CommitID = commit
				occurrence.TreeID = header.tree
				occurrence.Path = entry.Path
				occurrence.TestFile = strings.HasSuffix(entry.Path, "_test.go")
				occurrence.FileMode = entry.Mode
				occurrence.BlobID = entry.OID
				occurrence.BlobSize = result.size
				occurrence.BlobSHA = result.sha
				occurrence.OccurrenceRef = digestStrings(
					"orquesta.legacy-go-function-occurrence.v1",
					commit, entry.Path, strconv.Itoa(symbol.StartOffset), symbol.VariantRef,
				)
				occurrence.CanonicalSource = ""
				records = append(records, occurrence)
			}
		}
	}
	var variantKeys []string
	for key := range variants {
		variantKeys = append(variantKeys, key)
	}
	sort.Strings(variantKeys)
	for _, key := range variantKeys {
		records = append(records, variants[key])
	}
	sortRecords(records)

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	for _, item := range records {
		if err := encoder.Encode(item); err != nil {
			return err
		}
		counts[item.RecordKind]++
	}
	if err := batch.close(); err != nil {
		return err
	}
	refsAfter, err := readRefs(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	if !equalRefs(refsBefore, refsAfter) {
		return errors.New("las referencias Git cambiaron durante el censo")
	}
	inventorySHA := digest("orquesta.legacy-function-inventory.jsonl.v1", output.Bytes())
	manifestBytes, err := json.MarshalIndent(manifest{
		SchemaVersion:     schemaVersion,
		Algorithm:         "git-objects-go-ast-function-declarations.v1",
		Repository:        repository,
		GitObjectFormat:   objectFormat,
		GoVersion:         strings.TrimSpace(mustCommand("go", "version")),
		InventorySHA256:   inventorySHA,
		Counts:            counts,
		RefSnapshotSHA256: refDigest,
	}, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := writeAtomic(options.jsonl, output.Bytes()); err != nil {
		return err
	}
	return writeAtomic(options.manifest, manifestBytes)
}

func readHistory(repository string, processStarted func()) (map[string]historyEntry, error) {
	content, err := gitBytes(
		repository,
		processStarted,
		"rev-list",
		"--all",
		"--format=%H%x00%T%x00%P",
		"--no-commit-header",
	)
	if err != nil {
		return nil, err
	}
	result := make(map[string]historyEntry)
	for _, line := range bytes.Split(bytes.TrimSpace(content), []byte{'\n'}) {
		fields := bytes.Split(line, []byte{0})
		if len(fields) != 3 {
			return nil, errors.New("entrada inválida en el grafo histórico")
		}
		commit := string(fields[0])
		tree := string(fields[1])
		if commit == "" || tree == "" {
			return nil, errors.New("confirmación histórica sin identificador o árbol")
		}
		var parents []string
		if len(fields[2]) > 0 {
			parents = strings.Fields(string(fields[2]))
		}
		if previous, found := result[commit]; found &&
			(previous.tree != tree || !equalStrings(previous.parents, parents)) {
			return nil, fmt.Errorf("datos contradictorios para la confirmación %s", commit)
		}
		result[commit] = historyEntry{tree: tree, parents: parents}
	}
	return result, nil
}

func reachableCommits(root string, history map[string]historyEntry) ([]string, error) {
	if _, found := history[root]; !found {
		return nil, fmt.Errorf("la referencia apunta a una confirmación no censada: %s", root)
	}
	seen := make(map[string]struct{})
	pending := []string{root}
	for len(pending) > 0 {
		last := len(pending) - 1
		commit := pending[last]
		pending = pending[:last]
		if _, found := seen[commit]; found {
			continue
		}
		entry, found := history[commit]
		if !found {
			return nil, fmt.Errorf("falta el ancestro %s de la confirmación %s", commit, root)
		}
		seen[commit] = struct{}{}
		pending = append(pending, entry.parents...)
	}
	result := make([]string, 0, len(seen))
	for commit := range seen {
		result = append(result, commit)
	}
	sort.Strings(result)
	return result, nil
}

func readTree(
	batch *gitBatch,
	root string,
	objectBytes int,
	cache map[string][]treeEntry,
) ([]treeEntry, error) {
	var result []treeEntry
	var walk func(string, string) error
	walk = func(tree, prefix string) error {
		entries, found := cache[tree]
		if !found {
			object, err := batch.get(tree)
			if err != nil {
				return err
			}
			if object.kind != "tree" {
				return fmt.Errorf("objeto %s no es un árbol", tree)
			}
			entries, err = parseTree(object.content, objectBytes)
			if err != nil {
				return fmt.Errorf("objeto %s: %w", tree, err)
			}
			cache[tree] = entries
		}
		for _, entry := range entries {
			withPath := entry
			if prefix != "" {
				withPath.Path = prefix + "/" + entry.Path
			}
			result = append(result, withPath)
			if entry.Type == "tree" {
				if err := walk(entry.OID, withPath.Path); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].OID < result[j].OID
	})
	return result, nil
}

func parseTree(content []byte, objectBytes int) ([]treeEntry, error) {
	var result []treeEntry
	for len(content) > 0 {
		modeEnd := bytes.IndexByte(content, ' ')
		if modeEnd <= 0 {
			return nil, errors.New("modo ausente en entrada de árbol")
		}
		nameEnd := bytes.IndexByte(content[modeEnd+1:], 0)
		if nameEnd < 0 {
			return nil, errors.New("nombre sin terminador en entrada de árbol")
		}
		nameEnd += modeEnd + 1
		objectStart := nameEnd + 1
		objectEnd := objectStart + objectBytes
		if objectEnd > len(content) {
			return nil, errors.New("identificador truncado en entrada de árbol")
		}
		mode := string(content[:modeEnd])
		kind := "blob"
		switch mode {
		case "40000":
			mode = "040000"
			kind = "tree"
		case "160000":
			kind = "commit"
		}
		result = append(result, treeEntry{
			Mode: mode,
			Type: kind,
			OID:  hex.EncodeToString(content[objectStart:objectEnd]),
			Path: string(content[modeEnd+1 : nameEnd]),
		})
		content = content[objectEnd:]
	}
	return result, nil
}

func objectIDBytes(objectFormat string) (int, error) {
	switch strings.TrimSpace(objectFormat) {
	case "sha1":
		return sha1.Size, nil
	case "sha256":
		return sha256.Size, nil
	default:
		return 0, fmt.Errorf("formato de objetos Git no admitido: %q", objectFormat)
	}
}

func parseBlob(content []byte, diagnosticPath string) (blobResult, error) {
	fset := token.NewFileSet()
	file, parseErr := parser.ParseFile(
		fset, diagnosticPath, content,
		parser.AllErrors|parser.ParseComments|parser.SkipObjectResolution,
	)
	if parseErr != nil {
		return blobResult{
			size: len(content),
			sha:  digest("orquesta.legacy-go-blob.v1", content),
			failure: &record{
				RecordKind: "parse_failure", ErrorCode: "go_parse_failed",
				ErrorDetail: parseErr.Error(),
			},
		}, nil
	}
	constraints, generated := sourceMetadata(content)
	result := blobResult{
		size: len(content),
		sha:  digest("orquesta.legacy-go-blob.v1", content),
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		start := fset.PositionFor(function.Pos(), false)
		end := fset.PositionFor(function.End(), false)
		if start.Offset < 0 || end.Offset < start.Offset || end.Offset > len(content) {
			return blobResult{}, fmt.Errorf("rango AST inválido en %s", diagnosticPath)
		}
		canonicalFunction := *function
		canonicalFunction.Doc = nil
		canonical, err := formatNode(fset, &canonicalFunction)
		if err != nil {
			return blobResult{}, err
		}
		signatureFunction := *function
		signatureFunction.Doc = nil
		signatureFunction.Body = nil
		signature, err := formatNode(fset, &signatureFunction)
		if err != nil {
			return blobResult{}, err
		}
		receiver := ""
		kind := "func"
		if function.Recv != nil && len(function.Recv.List) > 0 {
			kind = "method"
			receiver, err = formatNode(fset, function.Recv.List[0].Type)
			if err != nil {
				return blobResult{}, err
			}
		}
		body := ""
		bodySHA := ""
		if function.Body != nil {
			body, err = formatNode(fset, function.Body)
			if err != nil {
				return blobResult{}, err
			}
			bodySHA = digestStrings("orquesta.legacy-go-function-body.v1", body)
		}
		docSHA := ""
		if function.Doc != nil {
			docStart := fset.PositionFor(function.Doc.Pos(), false).Offset
			docEnd := fset.PositionFor(function.Doc.End(), false).Offset
			if docStart >= 0 && docEnd >= docStart && docEnd <= len(content) {
				docSHA = digest("orquesta.legacy-go-function-doc.v1", content[docStart:docEnd])
			}
		}
		astSHA := digestStrings("orquesta.legacy-go-function-ast.v1", canonical)
		result.records = append(result.records, record{
			Package: file.Name.Name, BuildConstraints: constraints,
			TestFile: strings.HasSuffix(diagnosticPath, "_test.go"), Generated: generated,
			SymbolKind: kind, Name: function.Name.Name, Exported: ast.IsExported(function.Name.Name),
			Receiver: receiver, Signature: signature,
			StartOffset: start.Offset, EndOffset: end.Offset,
			StartLine: start.Line, EndLine: end.Line,
			SourceSHA: digest("orquesta.legacy-go-function-source.v1", content[start.Offset:end.Offset]),
			DocSHA:    docSHA, ASTCanonical: "gofmt-funcdecl.v1",
			ASTSHA: astSHA, BodySHA: bodySHA,
			VariantRef:      "go-function-variant:" + astSHA,
			CanonicalSource: canonical,
		})
	}
	return result, nil
}

func sourceMetadata(content []byte) ([]string, bool) {
	var constraints []string
	generated := false
	for index, raw := range strings.Split(string(content), "\n") {
		if index > 40 {
			break
		}
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "//go:build ") || strings.HasPrefix(line, "// +build ") {
			constraints = append(constraints, line)
		}
		if strings.Contains(line, "Code generated") && strings.Contains(line, "DO NOT EDIT.") {
			generated = true
		}
	}
	sort.Strings(constraints)
	return constraints, generated
}

func formatNode(fset *token.FileSet, node any) (string, error) {
	astNode, ok := node.(ast.Node)
	if !ok {
		return "", errors.New("nodo AST inválido")
	}
	var output bytes.Buffer
	if err := format.Node(&output, fset, astNode); err != nil {
		return "", err
	}
	return output.String(), nil
}

func readRefs(repository string, processStarted func()) ([]refInfo, error) {
	lines, err := gitLines(repository, processStarted, "for-each-ref",
		"--format=%(refname)%09%(objectname)", "refs/heads", "refs/remotes", "refs/tags")
	if err != nil {
		return nil, err
	}
	batch, err := newGitBatch(repository, processStarted)
	if err != nil {
		return nil, err
	}
	defer batch.close()
	var result []refInfo
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 2 || strings.HasSuffix(fields[0], "/HEAD") {
			continue
		}
		commit, err := batch.get(fields[0] + "^{commit}")
		if err != nil {
			if errors.Is(err, errGitObjectMissing) {
				continue
			}
			return nil, err
		}
		if commit.kind != "commit" {
			continue
		}
		result = append(result, refInfo{
			Name: fields[0], Object: fields[1], Commit: commit.oid,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	if err := batch.close(); err != nil {
		return nil, err
	}
	return result, nil
}

func equalRefs(left, right []refInfo) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sortRecords(records []record) {
	sort.Slice(records, func(i, j int) bool {
		left, right := records[i], records[j]
		leftKey := strings.Join([]string{
			left.RecordKind, left.RefName, left.CommitID, left.TreeID, left.Path,
			left.BlobID, left.VariantRef, strconv.Itoa(left.StartOffset),
		}, "\x00")
		rightKey := strings.Join([]string{
			right.RecordKind, right.RefName, right.CommitID, right.TreeID, right.Path,
			right.BlobID, right.VariantRef, strconv.Itoa(right.StartOffset),
		}, "\x00")
		return leftKey < rightKey
	})
}

func digest(domain string, content []byte) string {
	hash := sha256.New()
	writeDigestField(hash, []byte(domain))
	writeDigestField(hash, content)
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func digestStrings(domain string, values ...string) string {
	hash := sha256.New()
	writeDigestField(hash, []byte(domain))
	for _, value := range values {
		writeDigestField(hash, []byte(value))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func digestJSON(domain string, value any) string {
	content, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return digest(domain, content)
}

func writeDigestField(hash interface{ Write([]byte) (int, error) }, content []byte) {
	var length [8]byte
	for index := 7; index >= 0; index-- {
		length[index] = byte(len(content))
		contentLength := len(content) >> (8 * (7 - index))
		length[index] = byte(contentLength)
	}
	_, _ = hash.Write(length[:])
	_, _ = hash.Write(content)
}

func gitLines(repository string, processStarted func(), arguments ...string) ([]string, error) {
	content, err := gitBytes(repository, processStarted, arguments...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func gitText(repository string, processStarted func(), arguments ...string) (string, error) {
	content, err := gitBytes(repository, processStarted, arguments...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func gitBytes(repository string, processStarted func(), arguments ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	content, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(arguments, " "), err, strings.TrimSpace(stderr.String()))
	}
	if processStarted != nil {
		processStarted()
	}
	return content, nil
}

func newGitBatch(repository string, processStarted func()) (*gitBatch, error) {
	command := exec.Command("git", "-C", repository, "cat-file", "--batch")
	input, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := command.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	batch := &gitBatch{command: command, input: input, output: bufio.NewReader(output)}
	command.Stderr = &batch.stderr
	if err := command.Start(); err != nil {
		input.Close()
		return nil, fmt.Errorf("iniciar git cat-file --batch: %w", err)
	}
	if processStarted != nil {
		processStarted()
	}
	return batch, nil
}

func (batch *gitBatch) get(expression string) (gitObject, error) {
	if batch.closed {
		return gitObject{}, errors.New("lector Git por lotes cerrado")
	}
	if expression == "" || strings.ContainsAny(expression, "\r\n") {
		return gitObject{}, errors.New("expresión de objeto Git inválida")
	}
	if _, err := io.WriteString(batch.input, expression+"\n"); err != nil {
		return gitObject{}, fmt.Errorf("consultar objeto Git %s: %w", expression, err)
	}
	header, err := batch.output.ReadString('\n')
	if err != nil {
		return gitObject{}, fmt.Errorf("leer cabecera del objeto Git %s: %w", expression, err)
	}
	header = strings.TrimSuffix(header, "\n")
	if strings.HasSuffix(header, " missing") {
		return gitObject{}, fmt.Errorf("%w: %s", errGitObjectMissing, expression)
	}
	fields := strings.Fields(header)
	if len(fields) != 3 {
		return gitObject{}, fmt.Errorf("cabecera inválida del objeto Git %s: %q", expression, header)
	}
	size, err := strconv.Atoi(fields[2])
	if err != nil || size < 0 {
		return gitObject{}, fmt.Errorf("tamaño inválido del objeto Git %s: %q", expression, fields[2])
	}
	content := make([]byte, size)
	if _, err := io.ReadFull(batch.output, content); err != nil {
		return gitObject{}, fmt.Errorf("leer objeto Git %s: %w", expression, err)
	}
	trailing, err := batch.output.ReadByte()
	if err != nil {
		return gitObject{}, fmt.Errorf("leer terminador del objeto Git %s: %w", expression, err)
	}
	if trailing != '\n' {
		return gitObject{}, fmt.Errorf("terminador inválido del objeto Git %s", expression)
	}
	return gitObject{oid: fields[0], kind: fields[1], content: content}, nil
}

func (batch *gitBatch) close() error {
	if batch == nil || batch.closed {
		return nil
	}
	batch.closed = true
	inputErr := batch.input.Close()
	waitErr := batch.command.Wait()
	if inputErr != nil {
		return inputErr
	}
	if waitErr != nil {
		return fmt.Errorf("cerrar git cat-file --batch: %w: %s", waitErr, strings.TrimSpace(batch.stderr.String()))
	}
	return nil
}

func mustCommand(name string, arguments ...string) string {
	content, err := exec.Command(name, arguments...).Output()
	if err != nil {
		panic(err)
	}
	return string(content)
}

func writeAtomic(path string, content []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".legacy-function-inventory-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
