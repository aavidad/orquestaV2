// Estas pruebas cubren pérdidas históricas, mutaciones implícitas y alias físicos.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestInventoryTraversesDirectCommitTreeAndBlobReferences(t *testing.T) {
	repository := newTestRepository(t)
	writeTestBytes(t, repository, "base.txt", []byte("base\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "base")

	directBlob := strings.TrimSpace(gitTestInput(
		t, repository, []byte("blob directo\n"), "hash-object", "-w", "--stdin",
	))
	treeBlob := strings.TrimSpace(gitTestInput(
		t, repository, []byte("# árbol directo\n"), "hash-object", "-w", "--stdin",
	))
	directTree := strings.TrimSpace(gitTestInput(
		t, repository,
		[]byte(fmt.Sprintf("100644 blob %s\tdirecto.md\n", treeBlob)),
		"mktree",
	))
	gitTest(t, repository, "update-ref", "refs/inventario/blob", directBlob)
	gitTest(t, repository, "update-ref", "refs/inventario/tree", directTree)

	records := runAndReadInventory(t, repository)
	facts := factsByBlob(records)
	if facts[directBlob].BlobID == "" || facts[treeBlob].BlobID == "" {
		t.Fatalf("faltan blobs alcanzables directamente: %#v", facts)
	}
	var foundTree, foundBlobTarget, foundTreeTarget, foundCommitTarget bool
	for _, item := range records {
		switch {
		case item.RecordKind == "objeto_arbol" && item.TreeID == directTree:
			foundTree = true
		case item.RecordKind == "referencia" && item.TargetID == directBlob && item.TargetType == "blob":
			foundBlobTarget = true
		case item.RecordKind == "referencia" && item.TargetID == directTree && item.TargetType == "tree":
			foundTreeTarget = true
		case item.RecordKind == "referencia" && item.TargetType == "commit":
			foundCommitTarget = true
		}
	}
	if !foundTree || !foundBlobTarget || !foundTreeTarget || !foundCommitTarget {
		t.Fatalf("referencias o raíces incompletas: tree=%t blob=%t tree_ref=%t commit=%t",
			foundTree, foundBlobTarget, foundTreeTarget, foundCommitTarget)
	}
}

func TestVendoredBoundaryKeepsOwnMaterialIdentityWithoutReadingBlobs(t *testing.T) {
	repository := newTestRepository(t)
	writeTestBytes(t, repository, "vendor", []byte("fichero propio llamado vendor\n"))
	writeTestBytes(t, repository, "deps/vendor/parches/orquesta.patch", []byte("parche propio\n"))
	writeTestBytes(t, repository, "deps/vendor/LICENSE.orquesta", []byte("licencia\n"))
	writeTestBytes(t, repository, "deps/vendor/conexion/orquesta.go", []byte("package conexion\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "material vendorizado")

	records := runAndReadInventory(t, repository)
	facts := factsByBlob(records)
	paths := reconstructAllGraphPaths(records)
	regularVendor := paths["vendor"]
	if regularVendor.ExclusionCause != "" || facts[regularVendor.ChildObjectID].BlobID == "" {
		t.Fatalf("un fichero regular vendor fue excluido: %#v", regularVendor)
	}
	boundary := paths["deps/vendor"]
	if boundary.ChildType != "tree" ||
		boundary.ExclusionCause != "dependencia_vendorizada_vendor" ||
		boundary.ReviewObligation != vendoredReviewDuty {
		t.Fatalf("frontera vendorizada sin obligación explícita: %#v", boundary)
	}
	for _, expected := range []string{
		"deps/vendor/parches/orquesta.patch",
		"deps/vendor/LICENSE.orquesta",
		"deps/vendor/conexion/orquesta.go",
	} {
		item := paths[expected]
		if item.ChildObjectID == "" {
			t.Fatalf("no se enumeró la identidad y ruta de %s", expected)
		}
		if facts[item.ChildObjectID].BlobID != "" {
			t.Fatalf("se leyó contenido excluido de %s", expected)
		}
	}
}

func TestInventoryPreservesMergeParentOrder(t *testing.T) {
	repository := newTestRepository(t)
	blob := strings.TrimSpace(gitTestInput(t, repository, []byte("base\n"), "hash-object", "-w", "--stdin"))
	tree := strings.TrimSpace(gitTestInput(
		t, repository, []byte(fmt.Sprintf("100644 blob %s\tbase.txt\n", blob)), "mktree",
	))
	first := strings.TrimSpace(gitTest(t, repository, "commit-tree", tree, "-m", "primero"))
	second := strings.TrimSpace(gitTest(t, repository, "commit-tree", tree, "-m", "segundo"))
	merge := strings.TrimSpace(gitTest(
		t, repository, "commit-tree", tree, "-p", second, "-p", first, "-m", "fusión",
	))
	gitTest(t, repository, "update-ref", "refs/heads/main", merge)

	for _, item := range runAndReadInventory(t, repository) {
		if item.RecordKind == "confirmacion" && item.CommitID == merge {
			if !reflect.DeepEqual(item.Parents, []string{second, first}) {
				t.Fatalf("orden de padres=%q; esperado=%q", item.Parents, []string{second, first})
			}
			return
		}
	}
	t.Fatal("no se encontró la confirmación de fusión")
}

func TestInventoryRejectsSymbolicParentsAndPhysicalAliases(t *testing.T) {
	repository := newTestRepository(t)
	writeTestBytes(t, repository, "base.txt", []byte("base\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "base")

	directory := t.TempDir()
	repositoryAlias := filepath.Join(directory, "repositorio-enlazado")
	if err := os.Symlink(repository, repositoryAlias); err != nil {
		t.Fatal(err)
	}
	if err := run(options{
		repository: repository,
		jsonl:      filepath.Join(repositoryAlias, "colado.jsonl"),
		manifest:   filepath.Join(directory, "manifest.json"),
	}); err == nil {
		t.Fatal("se aceptó una salida que atravesaba un enlace hacia el repositorio")
	}
	if _, err := os.Lstat(filepath.Join(repository, "colado.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("la validación llegó a escribir dentro del repositorio: %v", err)
	}

	left := filepath.Join(directory, "izquierda")
	right := filepath.Join(directory, "derecha")
	if err := os.WriteFile(left, []byte("anterior"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(left, right); err != nil {
		t.Fatal(err)
	}
	if err := run(options{repository: repository, jsonl: left, manifest: right}); err == nil {
		t.Fatal("se aceptaron dos alias físicos del mismo fichero")
	}
}

func TestInventoryDoesNotLazyFetchMissingPromisorBlob(t *testing.T) {
	origin := filepath.Join(t.TempDir(), "origen.git")
	gitCommandForTest(t, "", "init", "--bare", "-q", origin)
	gitTest(t, origin, "config", "user.name", "Inventario")
	gitTest(t, origin, "config", "user.email", "inventario@example.invalid")
	gitTest(t, origin, "config", "uploadpack.allowFilter", "true")
	blob := strings.TrimSpace(gitTestInput(
		t, origin, []byte("contenido remoto\n"), "hash-object", "-w", "--stdin",
	))
	excludedBlob := strings.TrimSpace(gitTestInput(
		t, origin, []byte("contenido vendorizado remoto\n"), "hash-object", "-w", "--stdin",
	))
	vendorTree := strings.TrimSpace(gitTestInput(
		t, origin, []byte(fmt.Sprintf("100644 blob %s\tomitido.md\n", excludedBlob)), "mktree",
	))
	tree := strings.TrimSpace(gitTestInput(
		t, origin, []byte(fmt.Sprintf(
			"100644 blob %s\tremoto.md\n040000 tree %s\tvendor\n",
			blob, vendorTree,
		)), "mktree",
	))
	commit := strings.TrimSpace(gitTest(t, origin, "commit-tree", tree, "-m", "remoto"))
	gitTest(t, origin, "update-ref", "refs/heads/main", commit)
	gitTest(t, origin, "symbolic-ref", "HEAD", "refs/heads/main")

	partial := filepath.Join(t.TempDir(), "parcial")
	gitCommandForTest(t, "", "clone", "-q", "--filter=blob:none", "--no-checkout", "file://"+origin, partial)
	if objectExistsWithoutFetch(t, partial, blob) {
		t.Fatal("el servidor de prueba no dejó el blob como promisor ausente")
	}
	if objectExistsWithoutFetch(t, partial, excludedBlob) {
		t.Fatal("el servidor de prueba no dejó ausente el blob vendorizado")
	}
	before := objectStoreSnapshot(t, partial)
	records := runAndReadInventory(t, partial)
	if objectExistsWithoutFetch(t, partial, blob) {
		t.Fatal("el censo descargó un blob promisor ausente")
	}
	if objectExistsWithoutFetch(t, partial, excludedBlob) {
		t.Fatal("el censo descargó un blob vendorizado excluido")
	}
	if after := objectStoreSnapshot(t, partial); !reflect.DeepEqual(after, before) {
		t.Fatalf("el almacén Git cambió: antes=%q después=%q", before, after)
	}
	var missingRecorded, excludedReadAttempt bool
	for _, item := range records {
		if item.RecordKind == "fallo" && item.BlobID == blob && item.ErrorCode == "blob_no_legible" {
			missingRecorded = true
		}
		if item.RecordKind == "fallo" && item.BlobID == excludedBlob {
			excludedReadAttempt = true
		}
	}
	if !missingRecorded {
		t.Fatal("el blob promisor ausente no quedó registrado como fallo")
	}
	if excludedReadAttempt {
		t.Fatal("el censo intentó leer el blob vendorizado excluido")
	}
}

func runAndReadInventory(t *testing.T, repository string) []record {
	t.Helper()
	directory := t.TempDir()
	jsonl := filepath.Join(directory, "inventario.jsonl")
	if err := run(options{
		repository: repository, jsonl: jsonl,
		manifest: filepath.Join(directory, "manifiesto.json"),
	}); err != nil {
		t.Fatal(err)
	}
	return readRecords(t, jsonl)
}

func factsByBlob(records []record) map[string]record {
	result := map[string]record{}
	for _, item := range records {
		if item.RecordKind == "hechos_blob" {
			result[item.BlobID] = item
		}
	}
	return result
}

func reconstructAllGraphPaths(records []record) map[string]record {
	trees := map[string][]record{}
	var roots []string
	for _, item := range records {
		switch item.RecordKind {
		case "confirmacion":
			roots = append(roots, item.TreeID)
		case "entrada_arbol":
			trees[item.TreeID] = append(trees[item.TreeID], item)
		}
	}
	result := map[string]record{}
	var walk func(string, []byte)
	walk = func(treeID string, prefix []byte) {
		for _, entry := range trees[treeID] {
			segment, _ := decodeSegment(entry)
			fullPath := joinGitPath(prefix, segment)
			result[string(fullPath)] = entry
			if entry.ChildType == "tree" {
				walk(entry.ChildObjectID, fullPath)
			}
		}
	}
	for _, root := range roots {
		walk(root, nil)
	}
	return result
}

func gitCommandForTest(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", arguments...)
	if directory != "" {
		command.Dir = directory
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(arguments, " "), err, output)
	}
}

func objectExistsWithoutFetch(t *testing.T, repository, oid string) bool {
	t.Helper()
	command := exec.Command("git", "--no-lazy-fetch", "-C", repository, "cat-file", "-e", oid)
	return command.Run() == nil
}

func objectStoreSnapshot(t *testing.T, repository string) []string {
	t.Helper()
	var result []string
	root := filepath.Join(repository, ".git", "objects")
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result = append(result, fmt.Sprintf("%s:%d", relative, info.Size()))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(result)
	return result
}
