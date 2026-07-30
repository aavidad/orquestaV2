// Validación de colecciones históricas: miembros exactos, alias redactados,
// metadatos coherentes y huellas inmutables sin abrir fuentes externas.
package orquesta_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"testing"
)

func legacyRootsValidateCollections(t *testing.T, document legacyRootsDocument, roots map[string]legacyRootsRoot) {
	t.Helper()
	wantCounts := map[string]int{
		"legacy_principal_worktrees_restantes": 7,
		"legacy_autonomia_worktrees":           4,
		"legacy_copia_temprana_worktrees":      13,
		"municipal_worktrees":                  2,
		"vec_principal_worktrees":              89,
		"consumidor_opes_candidato":            3,
		"grxgo_respaldos":                      7,
		"municipal_scripts_orquestacion":       4,
		"legacy_rebuild_worktrees":             23,
		"legacy_worktrees_operativos":          11,
		"v2_agent_worktrees":                   22,
		"v2_backups":                           1,
		"v2_runtime_workspaces":                4,
		"v2_manual_worktrees":                  12,
		"goal_workspaces":                      83,
	}
	wantDigests := map[string]string{
		"legacy_principal_worktrees_restantes": "sha256:2293e9b0eb545ecdeb309c3c145dd135f3520bff01927e1da7fed78da4529bb0",
		"legacy_autonomia_worktrees":           "sha256:89f849c771cd87cbbb89e871e5f4873fa91d4eb5c6085b5cb18d0349ca04d6fb",
		"legacy_copia_temprana_worktrees":      "sha256:3937deed17ce767bdd8b81b1ec27347a6ea5c60bba25e85249dc2e93d1f0c1a6",
		"municipal_worktrees":                  "sha256:4942a5c1dbaa2f882fd85e68fdde29f77bf71e22e23820855226bc10dd176b97",
		"vec_principal_worktrees":              "sha256:bed855e42cc2fc26fe9c457a41ae7d6012dbfd7f094f3f23fb301715a16a232b",
		"consumidor_opes_candidato":            "sha256:d08e440c0c9b23f417fdc01ff2ae9719f7f359c3467c413d53d34b25421d080f",
		"grxgo_respaldos":                      "sha256:4c990b2aca500f8bae3ab5997515ba75eb6c41ff45c31a407c53f8901e35e2da",
		"municipal_scripts_orquestacion":       "sha256:034f6fbd32cbe6a904ba15be48c7118d0e086ff46fddc90dc444c1827bd4159d",
		"legacy_rebuild_worktrees":             "sha256:ba327b5fecb1c91c01ae1823a318856d70d1a8e4bebfa8c0616626e2e64d5871",
		"legacy_worktrees_operativos":          "sha256:fe941bf370ff1e7daab609872015695672088302ddc50cbac0830e9b884b6b43",
		"v2_agent_worktrees":                   "sha256:af0322f12800cd674df31d982c0d441283148a974f38bb64e30bb54126b1d014",
		"v2_backups":                           "sha256:30422e0ba8396ffb3234a1a1c73fc33e0829fa7be76cec105678c54a4e3de6db",
		"v2_runtime_workspaces":                "sha256:7be40372ac5eda0f058fb7fa25118b0928de6de33dc7cead27c6d47e9db26c9a",
		"v2_manual_worktrees":                  "sha256:2d3fa010079b41c8ad0d79ba77ef8beb05b84c7b3eb61b96c8ab86aa991e5e0b",
		"goal_workspaces":                      "sha256:03828dd1e986657f4ad48b08498c5fe3abf3b5888906260039b5543953c17697",
	}
	collections := make(map[string]struct{}, len(document.Collections))
	natures := legacyStringSet(document.Vocabularies.Natures)
	oid, digest := regexp.MustCompile(`^[0-9a-f]{40}$`), regexp.MustCompile(`^[0-9a-f]{64}$`)
	vecDirtyMembers := 0
	for _, collection := range document.Collections {
		root, rootExists := roots[collection.RootID]
		count, countExists := wantCounts[collection.RootID]
		wantDigest, digestExists := wantDigests[collection.RootID]
		if !rootExists || root.ScopeKind != "collection" || !countExists || !digestExists ||
			len(collection.Members) != count || collection.ObservationBatchID != root.ObservationBatchID ||
			root.PhysicalCensusStatus != "pendiente" {
			t.Fatalf("colección inválida: %q", collection.RootID)
		}
		if _, duplicate := collections[collection.RootID]; duplicate {
			t.Fatalf("colección duplicada: %q", collection.RootID)
		}
		aliases := make(map[string]struct{}, len(collection.Members))
		for _, member := range collection.Members {
			if member.PathAlias == "" || !legacyInSet(natures, member.Nature) {
				t.Fatalf("miembro incompleto en %q: %#v", collection.RootID, member)
			}
			if _, duplicate := aliases[member.PathAlias]; duplicate {
				t.Fatalf("alias duplicado en %q: %q", collection.RootID, member.PathAlias)
			}
			if member.DirtyEntryCount != nil && *member.DirtyEntryCount < 0 ||
				member.ByteSize != nil && *member.ByteSize < 0 ||
				member.SHA256 != nil && !digest.MatchString(*member.SHA256) {
				t.Fatalf("metadata inválida en %q: %#v", collection.RootID, member)
			}
			if member.Nature == "git_worktree" {
				if member.HeadOID == nil || member.CommonGitDirRef == nil || member.Prunable == nil ||
					!oid.MatchString(*member.HeadOID) {
					t.Fatalf("miembro Git incompleto en %q: %#v", collection.RootID, member)
				}
				if member.ExistsAtObservation && member.DirtyEntryCount == nil ||
					!member.ExistsAtObservation && (member.DirtyEntryCount != nil || !*member.Prunable) {
					t.Fatalf("presencia Git incoherente en %q: %#v", collection.RootID, member)
				}
			}
			if member.DirtyEntryCount != nil && *member.DirtyEntryCount > 0 {
				if root.PhysicalCensusStatus != "pendiente" {
					t.Fatalf("colección sucia cerrada: %q", collection.RootID)
				}
				if collection.RootID == "vec_principal_worktrees" {
					vecDirtyMembers++
				}
			}
			if member.Nature == "orchestration_script" && (member.SHA256 == nil || member.ByteSize == nil) {
				t.Fatalf("guion municipal sin tamaño y digest: %#v", member)
			}
			aliases[member.PathAlias] = struct{}{}
		}
		encoded, err := json.Marshal(collection.Members)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		hash.Write(encoded)
		hash.Write([]byte{'\n'})
		got := "sha256:" + hex.EncodeToString(hash.Sum(nil))
		if got != collection.MembersSHA256 || got != wantDigest {
			t.Fatalf("digest de miembros inesperado en %q: %s", collection.RootID, got)
		}
		collections[collection.RootID] = struct{}{}
	}
	if len(collections) != len(wantCounts) || vecDirtyMembers != 2 {
		t.Fatalf("cobertura de colecciones incompleta o suciedad VEC incoherente: colecciones=%d sucios=%d", len(collections), vecDirtyMembers)
	}
	for id, root := range roots {
		if root.ScopeKind == "collection" {
			if _, exists := collections[id]; !exists {
				t.Fatalf("colección sin miembros: %q", id)
			}
		}
	}
}
