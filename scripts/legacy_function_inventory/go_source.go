// Este fichero caracteriza declaraciones de funciones a partir de un blob Go.
// La salida depende solo de sus bytes: no consulta Git, el sistema de ficheros
// ni decisiones semánticas del inventario.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

type blobResult struct {
	size    int
	sha     string
	records []record
	failure *record
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
