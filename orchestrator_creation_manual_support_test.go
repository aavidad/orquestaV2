// Responsabilidad: comprobar la semántica, el JSON y las fuentes Git del manual.
// Alcance: soporte de prueba sin estado de producto ni acceso al legado.
// No acredita: estas comprobaciones solo protegen el contrato documental.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	creationManualLink        = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]+)\)`)
	creationManualAnchorNoise = regexp.MustCompile(`[^\pL\pN _-]`)
	creationManualEnglishText = regexp.MustCompile(`(?m)^ContextPackage$|^GoalRef$|^CommandEnvelope$|^AppDocumentationReport$|BuildContext\(|package ref|bounded resources|Goal failed|Scheduler sirve|write-sets|comportamiento legacy|schemas duplicados|^crash$| suite |no redeliver| manifest por|afinidad de host`)
)

type creationManualRoadmap struct {
	CatalogSize       int                               `json:"catalog_size"`
	CapabilityEntries []creationManualRoadmapCapability `json:"capability_entries"`
	Verticals         []creationManualRoadmapVertical   `json:"verticals"`
	Contracts         []creationManualRoadmapContract   `json:"acceptance_contracts"`
}
type creationManualRoadmapCapability struct {
	ID           string   `json:"id"`
	OwnerContext string   `json:"owner_context"`
	Status       string   `json:"status"`
	Contracts    []string `json:"acceptance_contracts"`
	EvidenceRefs []string `json:"evidence_refs"`
}
type creationManualRoadmapVertical struct {
	ID        string   `json:"id"`
	Sequence  int      `json:"sequence"`
	Contracts []string `json:"acceptance_contracts"`
}
type creationManualRoadmapContract struct {
	ID       string `json:"id"`
	Vertical string `json:"vertical"`
	Status   string `json:"status"`
}

func decodeCreationManualManifest(data []byte) (creationManualManifest, error) {
	var manifest creationManualManifest
	if err := walkCreationManualJSON(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return manifest, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return manifest, fmt.Errorf("datos posteriores al objeto JSON: %v", err)
	}
	canonical, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return manifest, err
	}
	if !bytes.Equal(data, append(canonical, '\n')) {
		return manifest, fmt.Errorf("representación JSON no canónica")
	}
	return manifest, nil
}
func walkCreationManualJSON(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	if delim == '[' {
		for decoder.More() {
			if err := walkCreationManualJSON(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	}
	seen := map[string]struct{}{}
	for decoder.More() {
		keyToken, keyErr := decoder.Token()
		if keyErr != nil {
			return keyErr
		}
		key := keyToken.(string)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("clave JSON duplicada: %s", key)
		}
		seen[key] = struct{}{}
		if err := walkCreationManualJSON(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}
func requireCreationManualJSONNegatives(t *testing.T, data []byte) {
	unknown := bytes.Replace(data, []byte("{\n"), []byte("{\n  \"desconocida\": true,\n"), 1)
	duplicate := append([]byte("{\"schema_version\":1,"), data[1:]...)
	nonCanonical := bytes.Replace(data, []byte(`"schema_version": 1`), []byte(`"schema_version":1`), 1)
	for _, invalid := range [][]byte{
		unknown, duplicate, append(append([]byte{}, data...), []byte("{}")...), nonCanonical,
	} {
		if _, err := decodeCreationManualManifest(invalid); err == nil {
			t.Fatal("el manifiesto inválido fue aceptado")
		}
	}
}
func readCreationManualRegularFile(t *testing.T, root, path string) []byte {
	info := requireCreationManualContained(t, root, path)
	if !info.Mode().IsRegular() {
		t.Fatalf("%s no es un fichero regular", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	return data
}
func requireCreationManualContained(t *testing.T, root, path string) os.FileInfo {
	relative, err := filepath.Rel(root, filepath.Clean(path))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatalf("ruta fuera del repositorio: %s", path)
	}
	current := root
	var info os.FileInfo
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err = os.Lstat(current)
		if err != nil {
			t.Fatalf("resolver %s: %v", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("no se siguen enlaces simbólicos: %s", current)
		}
	}
	return info
}
func requireCreationManualLinks(t *testing.T, root, source, content string) {
	for _, match := range creationManualLink.FindAllStringSubmatch(content, -1) {
		raw := strings.Trim(strings.TrimSpace(match[1]), "<>")
		if strings.Contains(raw, "://") || strings.HasPrefix(raw, "mailto:") {
			continue
		}
		target, fragment, _ := strings.Cut(raw, "#")
		target, _, _ = strings.Cut(target, "?")
		if filepath.IsAbs(target) {
			t.Fatalf("enlace local inválido en %s: %q", source, match[1])
		}
		path := source
		if target != "" {
			path = filepath.Join(filepath.Dir(source), filepath.FromSlash(target))
		}
		requireCreationManualContained(t, root, path)
		if fragment != "" {
			requireCreationManualAnchor(t, root, path, fragment)
		}
	}
}
func requireCreationManualAnchor(t *testing.T, root, path, fragment string) {
	content := readCreationManualRegularFile(t, root, path)
	if creationManualAnchorExists(string(content), fragment) {
		return
	}
	t.Fatalf("%s no contiene el ancla %q", path, fragment)
}
func creationManualAnchorExists(content, fragment string) bool {
	used := map[string]struct{}{}
	fence := ""
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```"):
			if fence == "" {
				fence = "```"
			} else if fence == "```" {
				fence = ""
			}
			continue
		case strings.HasPrefix(trimmed, "~~~"):
			if fence == "" {
				fence = "~~~"
			} else if fence == "~~~" {
				fence = ""
			}
			continue
		case fence != "":
			continue
		}
		heading, ok := creationManualATXHeading(line)
		if !ok {
			continue
		}
		base := strings.ReplaceAll(
			creationManualAnchorNoise.ReplaceAllString(strings.ToLower(heading), ""),
			" ",
			"-",
		)
		anchor := base
		for suffix := 1; ; suffix++ {
			if _, duplicate := used[anchor]; !duplicate {
				break
			}
			anchor = fmt.Sprintf("%s-%d", base, suffix)
		}
		used[anchor] = struct{}{}
		if anchor == fragment {
			return true
		}
	}
	return false
}
func creationManualATXHeading(line string) (string, bool) {
	hashes := 0
	for hashes < len(line) && line[hashes] == '#' {
		hashes++
	}
	if hashes == 0 || hashes > 6 || hashes == len(line) ||
		(line[hashes] != ' ' && line[hashes] != '\t') {
		return "", false
	}
	heading := strings.TrimSpace(line[hashes:])
	closingStart := len(heading)
	for closingStart > 0 && heading[closingStart-1] == '#' {
		closingStart--
	}
	if closingStart < len(heading) && closingStart > 0 &&
		(heading[closingStart-1] == ' ' || heading[closingStart-1] == '\t') {
		heading = strings.TrimSpace(heading[:closingStart])
	}
	return heading, heading != ""
}
func requireCreationManualAnchorNegatives(t *testing.T) {
	const content = "# Documento\n\n## Sección repetida\n\n## Sección repetida\n\n## Sección repetida\n"
	for _, exact := range []string{"documento", "sección-repetida", "sección-repetida-1", "sección-repetida-2"} {
		if !creationManualAnchorExists(content, exact) {
			t.Fatalf("no se reconoció el ancla GitHub exacta %q", exact)
		}
	}
	for _, invalid := range []string{"sección-repetida-0", "sección-repetida-3"} {
		if creationManualAnchorExists(content, invalid) {
			t.Fatalf("un sufijo de ancla incorrecto dio verde: %q", invalid)
		}
	}
}
func requireCreationManualStateCuts(t *testing.T, root string, cuts creationManualStateCuts) {
	base := creationManualStateCut{"base_versionada_257_38_38", "HEAD:product/roadmap.json", false, true, 257, 38, 38, 22, 16}
	wantV38 := creationManualV38Cut{
		Vertical:        "agent_runtime_elastic",
		Sequence:        38,
		Contract:        "AC-V38-AGENT-RUNTIME-ELASTIC",
		ContractStatus:  "planned",
		OwnedCapability: "ORC-28",
		NonOwnedCapabilities: []creationManualV38Requirement{
			{ID: "ORC-15", Vertical: 27},
			{ID: "OPS-16", Vertical: 32},
			{ID: "OPS-17", Vertical: 32},
		},
		Accredited: false,
	}
	if cuts.VersionedBase != base || !equalCreationManualV38Cut(cuts.V38, wantV38) ||
		!strings.Contains(strings.ToLower(cuts.Rule), "nunca") ||
		!strings.Contains(strings.ToLower(cuts.Rule), "planificada") ||
		!strings.Contains(strings.ToLower(cuts.Rule), "no acreditada") {
		t.Fatalf("cortes de estado inválidos: %+v", cuts)
	}
	head, err := exec.Command("git", "-C", root, "show", "HEAD:product/roadmap.json").Output()
	if err != nil {
		t.Fatalf("leer base versionada: %v", err)
	}
	requireCreationManualRoadmapStats(t, "base versionada", head, base)
	roadmap, err := decodeCreationManualRoadmap(head)
	if err != nil {
		t.Fatalf("decodificar semántica V38 de HEAD: %v", err)
	}
	if err := validateCreationManualV38Semantics(roadmap); err != nil {
		t.Fatalf("semántica V38 de HEAD inválida: %v", err)
	}
}
func requireCreationManualRoadmapStats(t *testing.T, name string, data []byte, want creationManualStateCut) {
	roadmap, err := decodeCreationManualRoadmap(data)
	if err != nil {
		t.Fatalf("%s ilegible: %v", name, err)
	}
	statuses := map[string]int{}
	if len(roadmap.CapabilityEntries) != roadmap.CatalogSize {
		t.Fatalf("%s no contiene todo su catálogo", name)
	}
	for _, contract := range roadmap.Contracts {
		statuses[contract.Status]++
	}
	if roadmap.CatalogSize != want.Catalog || len(roadmap.Verticals) != want.Verticals ||
		len(roadmap.Contracts) != want.Contracts || statuses["executable"] != want.ExecutableContracts ||
		statuses["planned"] != want.PlannedContracts {
		t.Fatalf("%s no coincide con el corte declarado", name)
	}
}
func decodeCreationManualRoadmap(data []byte) (creationManualRoadmap, error) {
	var roadmap creationManualRoadmap
	if err := json.Unmarshal(data, &roadmap); err != nil {
		return creationManualRoadmap{}, err
	}
	return roadmap, nil
}
func equalCreationManualV38Cut(left, right creationManualV38Cut) bool {
	if left.Vertical != right.Vertical || left.Sequence != right.Sequence ||
		left.Contract != right.Contract || left.ContractStatus != right.ContractStatus ||
		left.OwnedCapability != right.OwnedCapability || left.Accredited != right.Accredited ||
		len(left.NonOwnedCapabilities) != len(right.NonOwnedCapabilities) {
		return false
	}
	for index := range left.NonOwnedCapabilities {
		if left.NonOwnedCapabilities[index] != right.NonOwnedCapabilities[index] {
			return false
		}
	}
	return true
}
func validateCreationManualV38Semantics(roadmap creationManualRoadmap) error {
	verticals := make(map[string]creationManualRoadmapVertical, len(roadmap.Verticals))
	for _, vertical := range roadmap.Verticals {
		if _, duplicate := verticals[vertical.ID]; duplicate {
			return fmt.Errorf("vertical duplicada %s", vertical.ID)
		}
		verticals[vertical.ID] = vertical
	}
	contracts := make(map[string]creationManualRoadmapContract, len(roadmap.Contracts))
	for _, contract := range roadmap.Contracts {
		if _, duplicate := contracts[contract.ID]; duplicate {
			return fmt.Errorf("contrato duplicado %s", contract.ID)
		}
		contracts[contract.ID] = contract
	}
	capabilities := make(map[string]creationManualRoadmapCapability, len(roadmap.CapabilityEntries))
	var v38Owned []string
	for _, capability := range roadmap.CapabilityEntries {
		if _, duplicate := capabilities[capability.ID]; duplicate {
			return fmt.Errorf("capacidad duplicada %s", capability.ID)
		}
		capabilities[capability.ID] = capability
		if capability.OwnerContext == "agent_runtime_elastic" ||
			containsCreationManualString(capability.Contracts, "AC-V38-AGENT-RUNTIME-ELASTIC") {
			v38Owned = append(v38Owned, capability.ID)
		}
	}
	if err := validateCreationManualRoadmapVertical(
		verticals["agent_runtime_elastic"],
		"agent_runtime_elastic",
		38,
		"AC-V38-AGENT-RUNTIME-ELASTIC",
	); err != nil {
		return err
	}
	if err := validateCreationManualRoadmapVertical(
		verticals["context_rag_evals"],
		"context_rag_evals",
		27,
		"AC-V27-CONTEXT-RAG-EVALS",
	); err != nil {
		return err
	}
	if err := validateCreationManualRoadmapVertical(
		verticals["operations_telemetry"],
		"operations_telemetry",
		32,
		"AC-V32-OPERATIONS-TELEMETRY",
	); err != nil {
		return err
	}
	v38Contract := contracts["AC-V38-AGENT-RUNTIME-ELASTIC"]
	if v38Contract.ID != "AC-V38-AGENT-RUNTIME-ELASTIC" ||
		v38Contract.Vertical != "agent_runtime_elastic" || v38Contract.Status != "planned" {
		return fmt.Errorf("contrato V38 no es el plan canónico: %+v", v38Contract)
	}
	if len(v38Owned) != 1 || v38Owned[0] != "ORC-28" {
		return fmt.Errorf("propiedad V38 distinta de ORC-28: %v", v38Owned)
	}
	for _, expected := range []struct {
		id, owner, contract string
	}{
		{"ORC-28", "agent_runtime_elastic", "AC-V38-AGENT-RUNTIME-ELASTIC"},
		{"ORC-15", "context_rag_evals", "AC-V27-CONTEXT-RAG-EVALS"},
		{"OPS-16", "operations_telemetry", "AC-V32-OPERATIONS-TELEMETRY"},
		{"OPS-17", "operations_telemetry", "AC-V32-OPERATIONS-TELEMETRY"},
	} {
		capability := capabilities[expected.id]
		if capability.ID != expected.id || capability.OwnerContext != expected.owner ||
			capability.Status != "declared" || len(capability.EvidenceRefs) != 0 ||
			len(capability.Contracts) != 1 || capability.Contracts[0] != expected.contract {
			return fmt.Errorf("asignación o estado inválidos para %s: %+v", expected.id, capability)
		}
	}
	return nil
}
func validateCreationManualRoadmapVertical(
	vertical creationManualRoadmapVertical,
	id string,
	sequence int,
	contract string,
) error {
	if vertical.ID != id || vertical.Sequence != sequence ||
		len(vertical.Contracts) != 1 || vertical.Contracts[0] != contract {
		return fmt.Errorf("vertical %s inválida: %+v", id, vertical)
	}
	return nil
}
func containsCreationManualString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
func requireCreationManualV38SemanticNegatives(t *testing.T, root string) {
	head, err := exec.Command("git", "-C", root, "show", "HEAD:product/roadmap.json").Output()
	if err != nil {
		t.Fatalf("leer base versionada para mutaciones: %v", err)
	}
	mutations := []struct {
		name string
		edit func(*creationManualRoadmap)
	}{
		{"vertical sustituida", func(roadmap *creationManualRoadmap) {
			for index := range roadmap.Verticals {
				if roadmap.Verticals[index].ID == "agent_runtime_elastic" {
					roadmap.Verticals[index].ID = "runtime_distinto"
				}
			}
		}},
		{"contrato falsamente ejecutable", func(roadmap *creationManualRoadmap) {
			for index := range roadmap.Contracts {
				if roadmap.Contracts[index].ID == "AC-V38-AGENT-RUNTIME-ELASTIC" {
					roadmap.Contracts[index].Status = "executable"
				}
			}
		}},
		{"ORC-28 fuera de V38", func(roadmap *creationManualRoadmap) {
			for index := range roadmap.CapabilityEntries {
				if roadmap.CapabilityEntries[index].ID == "ORC-28" {
					roadmap.CapabilityEntries[index].OwnerContext = "provider_adapters"
				}
			}
		}},
		{"ORC-15 absorbida por V38", func(roadmap *creationManualRoadmap) {
			for index := range roadmap.CapabilityEntries {
				if roadmap.CapabilityEntries[index].ID == "ORC-15" {
					roadmap.CapabilityEntries[index].OwnerContext = "agent_runtime_elastic"
					roadmap.CapabilityEntries[index].Contracts = []string{"AC-V38-AGENT-RUNTIME-ELASTIC"}
				}
			}
		}},
		{"OPS-16 absorbida por V38", func(roadmap *creationManualRoadmap) {
			for index := range roadmap.CapabilityEntries {
				if roadmap.CapabilityEntries[index].ID == "OPS-16" {
					roadmap.CapabilityEntries[index].OwnerContext = "agent_runtime_elastic"
					roadmap.CapabilityEntries[index].Contracts = []string{"AC-V38-AGENT-RUNTIME-ELASTIC"}
				}
			}
		}},
	}
	for _, mutation := range mutations {
		t.Run(strings.ReplaceAll(mutation.name, " ", "_"), func(t *testing.T) {
			roadmap, err := decodeCreationManualRoadmap(head)
			if err != nil {
				t.Fatal(err)
			}
			mutation.edit(&roadmap)
			if err := validateCreationManualV38Semantics(roadmap); err == nil {
				t.Fatalf("la mutación semántica dio verde: %s", mutation.name)
			}
		})
	}
}
func requireCreationManualSemanticMarkers(t *testing.T, root string, got []creationManualSemanticMarker) {
	want := "equidad_contrapresion_histeresis|04_gestion_elastica_de_agentes.md|siempre\nsin_atestacion_remota_de_hardware|05_aislamiento_firecracker_y_microvm.md|siempre\nv31_multianfitrion_con_fencing|03_estado_transacciones_y_recuperacion.md|siempre\nconsultas_sin_escritura_de_producto|08_superficies_i18n_y_contratos_de_aplicacion.md|siempre\nintento_durable_antes_del_efecto|09_proceso_de_construccion_pruebas_y_acreditacion.md|siempre\napagado_estado_al_final|10_operacion_observabilidad_y_continuidad.md|siempre\nbase_versionada_257_38_38|12_auditoria_y_mapa_de_completitud.md|siempre"
	values := make([]string, 0, len(got))
	for _, marker := range got {
		values = append(values, marker.ID+"|"+marker.Chapter+"|"+marker.Condition)
	}
	if strings.Join(values, "\n") != want {
		t.Fatalf("marcadores semánticos inválidos: %+v", got)
	}
	for _, marker := range got {
		path := filepath.Join(root, manualRoot, marker.Chapter)
		if !bytes.Contains(readCreationManualRegularFile(t, root, path), []byte("`"+marker.ID+"`")) {
			t.Fatalf("%s no documenta %s", marker.Chapter, marker.ID)
		}
	}
}
