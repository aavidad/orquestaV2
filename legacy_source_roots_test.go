// Contrato en castellano del universo histórico: valida el manifiesto sin
// consultar rutas físicas del equipo ni convertir una observación pendiente en cierre.
package orquesta_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestLegacySourceRootsPolicy(t *testing.T) {
	raw := legacyRootsReadAndCheckKeys(t)
	var document legacyRootsDocument
	traceDecodeStrict(t, legacySourceRootsPath, &document)

	if document.DocumentKind != "orquesta_legacy_source_roots" || document.SchemaVersion != 3 {
		t.Fatalf("cabecera inesperada: %q/%d", document.DocumentKind, document.SchemaVersion)
	}
	if document.Closed || strings.TrimSpace(document.ClosureReason) == "" || strings.TrimSpace(document.SourceUniverseStatement) == "" {
		t.Fatal("el universo con bloqueos y recibo pendiente debe permanecer abierto")
	}
	if _, err := time.Parse(time.RFC3339, document.AssembledAt); err != nil {
		t.Fatalf("instante de ensamblado inválido: %v", err)
	}
	if strings.Contains(string(raw), "/home/") || regexp.MustCompile(`Codex[0-9]+|wt-alberto`).Match(raw) {
		t.Fatal("el manifiesto publica una ruta física o un identificador local de perfil")
	}
	if document.LogicalPathPolicy.LocalMappingTracked || document.LogicalPathPolicy.ProfileIDsPublished ||
		strings.TrimSpace(document.LogicalPathPolicy.Rule) == "" {
		t.Fatal("política de rutas lógicas insegura")
	}
	legacyRequireSameStrings(t, document.LogicalPathPolicy.RootRefs, []string{"trabajo", "temporal", "externo"}, "raíces lógicas")
	legacyRequireSameStrings(t, document.Vocabularies.RootClasses, []string{"repositorio_independiente", "copia_consulta", "worktree", "paquete", "consumidor_externo", "producto_actual"}, "clases")
	legacyRequireSameStrings(t, document.Vocabularies.ScopeKinds, []string{"single", "collection"}, "alcances")
	legacyRequireSameStrings(t, document.Vocabularies.SourceDecisions, []string{"incluir", "excluir", "pendiente"}, "decisiones")
	legacyRequireSameStrings(t, document.Vocabularies.GitCensusStatuses, []string{"observado", "pendiente", "no_aplica"}, "estados Git")
	legacyRequireSameStrings(t, document.Vocabularies.PhysicalStatuses, []string{"completo", "pendiente"}, "estados físicos")
	legacyRequireSameStrings(t, document.Vocabularies.GitObjectCoverages, []string{"propio", "contenido_en_producto_actual", "compartido_legacy", "compartido_producto_actual", "paquete_verificado", "sin_resolver", "no_aplica"}, "coberturas Git")
	method := document.ObservationMethod
	if method.GitVersion == "" || len(method.Commands) < 8 || len(method.Semantics) != 10 || len(method.ComparedRefSets) != 3 ||
		!strings.Contains(method.Semantics["dirty_entry_count"], "omite ignorados") {
		t.Fatalf("método de observación incompleto: %#v", method)
	}

	batches := legacyRootsValidateBatches(t, document)
	roots := legacyRootsValidateRoots(t, document, batches)
	legacyRootsValidateFacts(t, document, roots)
	legacyRootsValidateCollections(t, document, roots)
	currentWorktrees := 0
	legacyPrincipalFacts := 0
	for _, fact := range document.GitFacts {
		if fact.CommonGitDirRef == "producto_actual" {
			currentWorktrees++
		}
		if fact.CommonGitDirRef == "legacy_principal" {
			legacyPrincipalFacts++
		}
	}
	legacyPrincipalMembers := 0
	for _, collection := range document.Collections {
		for _, member := range collection.Members {
			if member.CommonGitDirRef != nil && *member.CommonGitDirRef == "producto_actual" {
				currentWorktrees++
			}
			if member.CommonGitDirRef != nil && *member.CommonGitDirRef == "legacy_principal" {
				legacyPrincipalMembers++
			}
		}
	}
	if currentWorktrees != legacyRootsObservedCount(t, document, "producto_actual").TotalCount {
		t.Fatalf("los worktrees V2 representados no coinciden con el lote: %d", currentWorktrees)
	}
	if legacyPrincipalFacts != 31 || legacyPrincipalMembers != 122 ||
		legacyPrincipalFacts+legacyPrincipalMembers != legacyRootsObservedCount(t, document, "legacy_principal").TotalCount {
		t.Fatalf("los 153 worktrees legacy no están reconciliados: hechos=%d miembros=%d", legacyPrincipalFacts, legacyPrincipalMembers)
	}
	legacyRootsValidateHistory(t, document, roots)
}

func legacyRootsValidateRoots(t *testing.T, document legacyRootsDocument, batches map[string]legacyRootsBatch) map[string]legacyRootsRoot {
	t.Helper()
	classes := legacyStringSet(document.Vocabularies.RootClasses)
	scopes := legacyStringSet(document.Vocabularies.ScopeKinds)
	natures := legacyStringSet(document.Vocabularies.Natures)
	decisions := legacyStringSet(document.Vocabularies.SourceDecisions)
	gitStatuses := legacyStringSet(document.Vocabularies.GitCensusStatuses)
	physicalStatuses := legacyStringSet(document.Vocabularies.PhysicalStatuses)
	coverages := legacyStringSet(document.Vocabularies.GitObjectCoverages)
	logicalRoots := legacyStringSet(document.LogicalPathPolicy.RootRefs)
	roots := make(map[string]legacyRootsRoot, len(document.Roots))
	paths := make(map[string]struct{}, len(document.Roots))
	var ids, unresolved, blocking, physicalComplete []string
	for _, root := range document.Roots {
		pathKey := root.LogicalRoot + ":" + root.RelativePath
		if root.ID == "" || root.RelativePath == "" || strings.HasPrefix(root.RelativePath, "/") ||
			strings.TrimSpace(root.SourceDecisionReason) == "" || strings.TrimSpace(root.GitCensusStatusReason) == "" ||
			strings.TrimSpace(root.PhysicalStatusReason) == "" {
			t.Fatalf("raíz incompleta: %#v", root)
		}
		if _, duplicate := roots[root.ID]; duplicate {
			t.Fatalf("ID duplicado: %q", root.ID)
		}
		if _, duplicate := paths[pathKey]; duplicate {
			t.Fatalf("ruta lógica duplicada: %q", pathKey)
		}
		if !legacyInSet(classes, root.Class) || !legacyInSet(scopes, root.ScopeKind) ||
			!legacyInSet(natures, root.Nature) || !legacyInSet(decisions, root.SourceDecision) ||
			!legacyInSet(gitStatuses, root.GitCensusStatus) || !legacyInSet(physicalStatuses, root.PhysicalCensusStatus) ||
			!legacyInSet(coverages, root.GitObjectCoverage) ||
			!legacyInSet(logicalRoots, root.LogicalRoot) {
			t.Fatalf("vocabulario no canónico en %q", root.ID)
		}
		if _, exists := batches[root.ObservationBatchID]; !exists {
			t.Fatalf("raíz %q sin lote de observación válido", root.ID)
		}
		if root.PhysicalCensusRequired != (root.PhysicalCensusStatus == "pendiente") {
			t.Fatalf("estado físico incoherente en %q", root.ID)
		}
		if root.SourceDecision == "pendiente" {
			unresolved = append(unresolved, root.ID)
		}
		if root.PhysicalCensusStatus == "pendiente" {
			blocking = append(blocking, root.ID)
		} else {
			physicalComplete = append(physicalComplete, root.ID)
		}
		ids = append(ids, root.ID)
		roots[root.ID], paths[pathKey] = root, struct{}{}
	}
	legacyRequireSameStrings(t, physicalComplete, []string{"self_server_binario", "server_e2e_binario", "paquete_separacion_legacy", "backup_autonomia_tar", "v2_temporal_bug460", "legacy_temporal_v22_01", "legacy_temporal_v22_02", "legacy_temporal_v22_03", "legacy_temporal_v22_04", "legacy_temporal_v22_05"}, "únicos cierres físicos permitidos")
	if len(roots) != 122 {
		t.Fatalf("conjunto de raíces incompleto: %d", len(roots))
	}
	if got := legacyLinesDigest(ids); got != document.RootIDsSHA256 ||
		got != "sha256:c4a93e4679a045ac68d835accb7bfd0ff0b6cb918bc2a7c9ab1248d3803291ec" {
		t.Fatalf("digest de IDs inesperado: %s", got)
	}
	legacyRequireSameStrings(t, unresolved, document.UnresolvedSourceDecisionRootIDs, "decisiones pendientes")
	legacyRequireSameStrings(t, blocking, document.BlockingPhysicalCensusRootIDs, "censos físicos bloqueantes")
	if len(unresolved) != 28 || len(blocking) != 112 {
		t.Fatalf("el documento perdió pendientes: decisiones=%d censos=%d", len(unresolved), len(blocking))
	}
	return roots
}

func legacyRootsValidateFacts(t *testing.T, document legacyRootsDocument, roots map[string]legacyRootsRoot) {
	t.Helper()
	oid := regexp.MustCompile(`^[0-9a-f]{40}$`)
	gitFacts := make(map[string]legacyRootsGitFact, len(document.GitFacts))
	for _, fact := range document.GitFacts {
		root, exists := roots[fact.RootID]
		if !exists || fact.CommonGitDirRef == "" || !oid.MatchString(fact.HeadOID) ||
			fact.ObservationBatchID != root.ObservationBatchID || root.GitCensusStatus != "observado" {
			t.Fatalf("hecho Git inválido: %#v", fact)
		}
		if _, duplicate := gitFacts[fact.RootID]; duplicate {
			t.Fatalf("hecho Git duplicado: %q", fact.RootID)
		}
		for _, count := range []*int{fact.RefCount, fact.CommitCount, fact.ReachableOnlyFromRoot, fact.DirtyEntryCount} {
			if count != nil && *count < 0 {
				t.Fatalf("contador Git negativo en %q", fact.RootID)
			}
		}
		if fact.DirtyEntryCount != nil && *fact.DirtyEntryCount > 0 && root.PhysicalCensusStatus != "pendiente" {
			t.Fatalf("raíz Git sucia cerrada: %q", fact.RootID)
		}
		gitFacts[fact.RootID] = fact
	}
	for id, root := range roots {
		if root.Nature == "git_repository" || root.Nature == "consult_copy" || root.Nature == "git_worktree" {
			if _, exists := gitFacts[id]; !exists {
				t.Fatalf("raíz Git sin hechos: %q", id)
			}
		}
	}
	fileFacts := make(map[string]struct{}, len(document.FileFacts))
	for _, fact := range document.FileFacts {
		root, exists := roots[fact.RootID]
		if !exists || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(fact.SHA256) ||
			fact.ByteSize < 0 || strings.TrimSpace(fact.Verification) == "" ||
			fact.ObservationBatchID != "evidencia_fichero_historica" || root.PhysicalCensusStatus != "completo" {
			t.Fatalf("hecho de fichero inválido: %#v", fact)
		}
		if _, duplicate := fileFacts[fact.RootID]; duplicate {
			t.Fatalf("hecho de fichero duplicado: %q", fact.RootID)
		}
		fileFacts[fact.RootID] = struct{}{}
	}
	for _, required := range []string{"paquete_separacion_legacy", "backup_autonomia_tar", "self_server_binario", "server_e2e_binario"} {
		if _, exists := fileFacts[required]; !exists {
			t.Fatalf("falta hecho de fichero para %q", required)
		}
	}
	for id, root := range roots {
		if root.ExistsAtObservation && root.PhysicalCensusStatus == "completo" {
			if _, exactFile := fileFacts[id]; !exactFile {
				t.Fatalf("raíz existente cerrada sin hecho de fichero exacto: %q", id)
			}
		}
		if root.ExistsAtObservation && (root.Nature == "git_repository" || root.Nature == "consult_copy" ||
			root.Nature == "git_worktree" || root.Nature == "worktree_collection") &&
			root.PhysicalCensusStatus != "pendiente" {
			t.Fatalf("raíz o worktree existente falsamente cerrado: %q", id)
		}
	}
	if len(gitFacts) != 48 || len(fileFacts) != 4 {
		t.Fatalf("conjuntos de hechos incompletos: git=%d ficheros=%d", len(gitFacts), len(fileFacts))
	}
}

func legacyRootsValidateHistory(t *testing.T, document legacyRootsDocument, roots map[string]legacyRootsRoot) {
	t.Helper()
	wantTags := legacyRootsExpectedTagOIDs()
	if len(document.V1V22Tags) != 22 {
		t.Fatalf("cobertura V1-V22 incompleta: %d", len(document.V1V22Tags))
	}
	for index, tag := range document.V1V22Tags {
		name := fmt.Sprintf("v%d", index+1)
		want := wantTags[name]
		if tag.Name != name || tag.TagObjectOID != want[0] || tag.CommitOID != want[1] ||
			tag.ObservationBatchID != "base_git_023829" {
			t.Fatalf("etiqueta histórica incorrecta: %#v", tag)
		}
		legacyRequireSameStrings(t, tag.ObservedRootIDs, []string{"legacy_principal", "legacy_consulta", "producto_actual", "paquete_separacion_legacy"}, "raíces de "+name)
		for _, rootID := range tag.ObservedRootIDs {
			if _, exists := roots[rootID]; !exists {
				t.Fatalf("%s referencia raíz inexistente %q", name, rootID)
			}
		}
	}
	type reconciliation struct {
		only      int
		contained bool
	}
	wantReconcile := map[string]reconciliation{"legacy_principal": {296, false}, "legacy_copia_temprana": {6, false}, "legacy_autonomia": {12, false}, "legacy_consulta": {0, true}, "old_director_source": {0, true}, "consumidor_municipal": {0, true}}
	if len(document.GitObjectReconciliation) != len(wantReconcile) {
		t.Fatal("reconciliación Git incompleta")
	}
	for _, item := range document.GitObjectReconciliation {
		want, exists := wantReconcile[item.RootID]
		if !exists || item.ReachableOnlyFromRoot != want.only || item.ContainedInCurrentProduct != want.contained || item.EvidenceStatus != "pendiente" ||
			item.ObservationBatchID != "base_git_023829" ||
			strings.TrimSpace(item.Reason) == "" {
			t.Fatalf("reconciliación no acreditada honestamente: %#v", item)
		}
	}
}

func legacyLinesDigest(values []string) string {
	values = append([]string(nil), values...)
	sort.Strings(values)
	hash := sha256.New()
	for _, value := range values {
		hash.Write([]byte(value))
		hash.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func legacyRequireSameStrings(t *testing.T, left, right []string, subject string) {
	t.Helper()
	left, right = append([]string(nil), left...), append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	if strings.Join(left, "\x00") != strings.Join(right, "\x00") {
		t.Fatalf("%s no coincide: %v != %v", subject, left, right)
	}
}

func legacyStringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func legacyInSet(values map[string]struct{}, value string) bool {
	_, exists := values[value]
	return exists
}

func legacyRootsExpectedTagOIDs() map[string][2]string {
	return map[string][2]string{
		"v1": {"ed77a36033bb6f50f00d490123dcaf34ab3303c0", "e68ad92280e0b60a78bcb191909734fdc5d224e3"}, "v2": {"80de7883d85157358aaff712a717ff0a74d39cd9", "e68ad92280e0b60a78bcb191909734fdc5d224e3"},
		"v3": {"6ed3ed26e32c020c8e4a730209a1c5380a3b2a86", "e68ad92280e0b60a78bcb191909734fdc5d224e3"}, "v4": {"71279f06f04cd2f5ef338ae9cf96fc8edbf6e92d", "e68ad92280e0b60a78bcb191909734fdc5d224e3"},
		"v5": {"d499d5e16691c3f82783ca145baeb8fa71c4a68d", "fb38152787e0de748a9f8c53d3865c5e7aeaece1"}, "v6": {"e46ac5e1189303f437bf120b7f79c40d5794a42d", "dc54f283919da15195cf026bad76677f1cf418b2"},
		"v7": {"bd69d9e8c9289a6861e376146215b2b27fa1fe34", "6d88f0f2d53c840686a93ed19f69f708dcbe7af3"}, "v8": {"e66247808d679fc05a53ff472ab39cc348c483ff", "afb7857ed37c20f8863e95a1bab2485d757e5e2d"},
		"v9": {"62fe2e587c784d72d9af70af762d1d8e7baeed1d", "70fb59e9fcdeb8cdeba2ba10273f409af0f27cfa"}, "v10": {"8e566a4fb18c5d734962e9b301b9f7fe448cc274", "c74b6d7664266ce7fccd5eb9366bc99fbe2ce7e0"},
		"v11": {"48fdf1d8e67e4b42d398b91147f630edd2e55305", "5e826de0e40c29d1e84f3dc5082e11667d045caa"}, "v12": {"ed145a3f8cea6fd8b44f730be6c0af51b7a80a0b", "5e826de0e40c29d1e84f3dc5082e11667d045caa"},
		"v13": {"86a05a6dfd90092c73f509649c1a3554a142b182", "5daf174bde3ec5d9a98f387de05491f258634264"}, "v14": {"dabfd2f79bb285b5965b9431a674dcbfadde4c88", "e3e7c28e669ccd7e67a8661c40333d649ab82dd5"},
		"v15": {"0fb0233967216f6e916507437ac6eaaf58bbafe0", "3eaa533d6fd1e7e1a7ef6bda956b8328bd0395a6"}, "v16": {"738d5e5bf30262c4115fb63378e885abf088039c", "a4f602ab01c4e77f0d876c79c2c8e86b44b68fa4"},
		"v17": {"81fd4d91796f5664f0bec21d1994a444569755b9", "a97ea3bc3771c6d89ec055e8189bda1bc6f97ce6"}, "v18": {"f88e6f1cc8c1c32637189ffc064eb4217955c65f", "32ee17e407006d9e0aeb46557b1e160769dd4848"},
		"v19": {"225d1c0ac76748b15c0c654c795b0e69ec19f26b", "f7a574e36528871ee07a5529a88be1f0a88511d7"}, "v20": {"d488900e64eabd09c57f1575e9921e51443e402c", "7f27685d992c9a8f5f308d94bbfc3e9828009d84"},
		"v21": {"988071519e6642bafc808cb9949b39be84661e86", "216e61e80c1b46ae049fbf2c5caefb1a69e5f7b3"}, "v22": {"af1cb2d06d436ae4a5b6ba63ba513c19c17ca11b", "d1b551a136fe133560e9c1228e54477acd429473"},
	}
}
