// Responsabilidad: validar el contrato estructural del manual de orquestación.
// Alcance: documentos y manifiesto del manual, sin escribir estado de producto.
// No acredita: esta prueba protege documentación; no acredita capacidades.
package orquesta_test

import (
	"path/filepath"
	"strings"
	"testing"
)

const manualRoot = "docs/reconstruccion/manual_orquestador"

type creationManualManifest struct {
	SchemaVersion int `json:"schema_version"`
	Authority     struct {
		Nature           string                 `json:"naturaleza"`
		ProductAuthority bool                   `json:"es_autoridad_de_producto"`
		Accredits        bool                   `json:"acredita_capacidades"`
		Description      string                 `json:"descripcion"`
		CurrentSources   []creationManualSource `json:"fuentes_vigentes"`
	} `json:"autoridad"`
	Purpose   string `json:"proposito"`
	Audiences []struct {
		ID          string `json:"id"`
		Description string `json:"descripcion"`
	} `json:"publicos"`
	Language struct {
		Tag      string `json:"etiqueta"`
		Register string `json:"registro"`
		Policy   string `json:"politica"`
	} `json:"lenguaje"`
	Prohibitions []string `json:"prohibiciones"`
	HonestStates struct {
		Order      []string `json:"orden"`
		Completion string   `json:"estado_que_cuenta_como_terminado"`
		Rule       string   `json:"regla"`
	} `json:"estados_honestos"`
	StateCuts       creationManualStateCuts        `json:"cortes_de_estado"`
	SemanticMarkers []creationManualSemanticMarker `json:"marcadores_semanticos"`
	Chapters        []creationManualChapter        `json:"capitulos"`
}
type creationManualSource struct {
	Priority  int    `json:"prioridad"`
	Path      string `json:"ruta"`
	Authority string `json:"autoridad"`
}
type creationManualStateCut struct {
	Marker              string `json:"marcador"`
	Source              string `json:"fuente"`
	Optional            bool   `json:"es_opcional,omitempty"`
	Authority           bool   `json:"es_autoridad"`
	Catalog             int    `json:"catalogo"`
	Verticals           int    `json:"verticales"`
	Contracts           int    `json:"contratos"`
	ExecutableContracts int    `json:"contratos_ejecutables"`
	PlannedContracts    int    `json:"contratos_planificados"`
}
type creationManualStateCuts struct {
	VersionedBase creationManualStateCut `json:"base_versionada"`
	V38           creationManualV38Cut   `json:"v38"`
	Rule          string                 `json:"regla"`
}
type creationManualV38Cut struct {
	Vertical             string                         `json:"vertical"`
	Sequence             int                            `json:"secuencia"`
	Contract             string                         `json:"contrato"`
	ContractStatus       string                         `json:"estado_contrato"`
	OwnedCapability      string                         `json:"capacidad_propia"`
	NonOwnedCapabilities []creationManualV38Requirement `json:"capacidades_no_propias"`
	Accredited           bool                           `json:"acreditada"`
}
type creationManualV38Requirement struct {
	ID       string `json:"id"`
	Vertical int    `json:"vertical"`
}
type creationManualSemanticMarker struct {
	ID        string `json:"id"`
	Chapter   string `json:"capitulo"`
	Condition string `json:"condicion"`
}
type creationManualChapter struct {
	Number         int      `json:"numero"`
	File           string   `json:"archivo"`
	Responsibility string   `json:"responsabilidad"`
	RequiredTopics []string `json:"temas_obligatorios"`
}

var creationManualChapters = map[string]struct {
	number int
	topics string
}{
	"01_mision_limites_y_vocabulario.md":                   {1, "aplicación de orquestación|glosario técnico común|Goal|WorkItem|autoridades|invariantes|criterio de finalización"},
	"02_nucleo_lifecycle_y_dag.md":                         {2, "Goal|WorkItem|DAG|escritor único|planificador único|Director"},
	"03_estado_transacciones_y_recuperacion.md":            {3, "transacciones|revisión esperada|idempotencia|bandeja transaccional|arrendamientos|recuperación|v31_multianfitrion_con_fencing|referencia de nodo (`NodeRef`)|batería contractual"},
	"04_gestion_elastica_de_agentes.md":                    {4, "demanda|capacidad|reconciliador|cuota|parada|conservación"},
	"05_aislamiento_firecracker_y_microvm.md":              {5, "Firecracker|microVM|vsock|credenciales|espacio de trabajo|conservación"},
	"06_seguridad_configuracion_identidad_y_efectos.md":    {6, "configuración|credenciales|identidad|aislamiento de proyecto|efectos|recibo"},
	"07_colaboracion_consejo_revision_y_contexto.md":       {7, "Director|subagentes|buzón causal|revisión primaria|revisión adversarial|Consejo|contexto|paquete de contexto (`ContextPackage`)|recursos acotados"},
	"08_superficies_i18n_y_contratos_de_aplicacion.md":     {8, "registro único de órdenes|HTTP|MCP|internacionalización|BCP-47|accesibilidad|APP-13|consultas_sin_escritura_de_producto|envolvente de orden (`CommandEnvelope`)|informe de documentación (`AppDocumentationReport`)"},
	"09_proceso_de_construccion_pruebas_y_acreditacion.md": {9, "catálogo|cortes verticales|microtareas|pruebas de mutación|revisión independiente|acreditación"},
	"10_operacion_observabilidad_y_continuidad.md":         {10, "arranque|observabilidad|diagnóstico|parada ordenada|copias verificables|restauración|incidencias"},
	"11_lecciones_antipatrones_y_diagnostico.md":           {11, "fallos históricos|antipatrones|diagnóstico|ciclos|perfiles estáticos|legado"},
	"12_auditoria_y_mapa_de_completitud.md":                {12, "auditoría|inventario|completitud|capacidades|huecos|tareas"},
}

func TestOrchestratorCreationManualContract(t *testing.T) {
	root := "."
	for _, name := range []string{"orchestrator_creation_manual_test.go", "orchestrator_creation_manual_support_test.go"} {
		header := strings.SplitN(string(readCreationManualRegularFile(t, root, filepath.Join(root, name))), "package ", 2)[0]
		requireCreationManualPhrases(t, name, header, []string{"// Responsabilidad:", "// Alcance:", "// No acredita:"})
	}
	manifestPath := filepath.Join(root, manualRoot, "manual_manifest.json")
	data := readCreationManualRegularFile(t, root, manifestPath)
	manifest, err := decodeCreationManualManifest(data)
	if err != nil {
		t.Fatalf("decodificar manifiesto: %v", err)
	}
	t.Run("json_invalido", func(t *testing.T) { requireCreationManualJSONNegatives(t, data) })
	if manifest.SchemaVersion != 1 || manifest.Authority.Nature != "explicativa" ||
		manifest.Authority.ProductAuthority || manifest.Authority.Accredits {
		t.Fatalf("la autoridad del manual debe ser explicativa, no de producto ni acreditadora: %+v", manifest.Authority)
	}
	if strings.TrimSpace(manifest.Purpose) == "" || strings.TrimSpace(manifest.Authority.Description) == "" {
		t.Fatal("el propósito y la descripción de autoridad no pueden estar vacíos")
	}
	audiences := map[string]bool{"desde_cero": false, "continuacion": false, "auditoria": false}
	for _, audience := range manifest.Audiences {
		if _, ok := audiences[audience.ID]; !ok || audiences[audience.ID] || strings.TrimSpace(audience.Description) == "" {
			t.Fatalf("público inválido o duplicado: %+v", audience)
		}
		audiences[audience.ID] = true
	}
	if len(manifest.Audiences) != len(audiences) {
		t.Fatalf("deben existir exactamente tres públicos: %+v", manifest.Audiences)
	}
	language := strings.ToLower(manifest.Language.Register + " " + manifest.Language.Policy)
	if manifest.Language.Tag != "es" || !strings.Contains(language, "castellano") || !strings.Contains(language, "programadores") {
		t.Fatalf("la política de lenguaje debe exigir castellano para programadores: %+v", manifest.Language)
	}
	requireCreationManualPhrases(t, "prohibiciones", strings.Join(manifest.Prohibitions, "\n"), []string{"catálogo de capacidades", "atribuir acreditación", "autorreferenciales", "legado"})
	if strings.Join(manifest.HonestStates.Order, "|") != "declarada|implementada|conectada|ejercitada|acreditada" ||
		manifest.HonestStates.Completion != "acreditada" {
		t.Fatalf("estados honestos inválidos: %+v", manifest.HonestStates)
	}
	honestRule := strings.ToLower(manifest.HonestStates.Rule)
	if !strings.Contains(honestRule, "evidencia") || !strings.Contains(honestRule, "candidato") {
		t.Fatalf("la regla de cierre debe ligar evidencia y candidato: %q", manifest.HonestStates.Rule)
	}
	if len(manifest.Chapters) != len(creationManualChapters) {
		t.Fatalf("capítulos: obtenidos %d, esperados %d", len(manifest.Chapters), len(creationManualChapters))
	}
	requireCreationManualSources(t, root, manifest.Authority.CurrentSources)
	requireCreationManualStateCuts(t, root, manifest.StateCuts)
	t.Run("v38_semantica_inmutable", func(t *testing.T) {
		requireCreationManualV38SemanticNegatives(t, root)
	})
	requireCreationManualSemanticMarkers(t, root, manifest.SemanticMarkers)
	t.Run("anclas_duplicadas", requireCreationManualAnchorNegatives)

	indexPath := filepath.Join(root, manualRoot, "README.md")
	index := readCreationManualRegularFile(t, root, indexPath)
	if !strings.Contains(string(index), "](manual_manifest.json)") {
		t.Fatal("el índice no enlaza manual_manifest.json")
	}
	requireCreationManualLinks(t, root, indexPath, string(index))
	seen := make(map[string]struct{}, len(manifest.Chapters))
	for _, chapter := range manifest.Chapters {
		expected, ok := creationManualChapters[chapter.File]
		if !ok {
			t.Fatalf("capítulo no previsto: %q", chapter.File)
		}
		if _, duplicate := seen[chapter.File]; duplicate {
			t.Fatalf("capítulo duplicado: %q", chapter.File)
		}
		seen[chapter.File] = struct{}{}
		if chapter.Number != expected.number || strings.TrimSpace(chapter.Responsibility) == "" ||
			strings.Join(chapter.RequiredTopics, "|") != expected.topics {
			t.Fatalf("contrato inválido para %s: %+v", chapter.File, chapter)
		}
		if !strings.Contains(string(index), "]("+chapter.File+")") {
			t.Fatalf("el índice no enlaza %s", chapter.File)
		}
		path := filepath.Join(root, manualRoot, chapter.File)
		content := readCreationManualRegularFile(t, root, path)
		requireCreationManualHeader(t, chapter, string(content))
		requireCreationManualPhrases(t, chapter.File, string(content), strings.Split(expected.topics, "|"))
		if creationManualEnglishText.Match(content) {
			t.Fatalf("%s conserva seudocódigo o prosa sustituible en inglés", chapter.File)
		}
		requireCreationManualLinks(t, root, path, string(content))
	}
}

func requireCreationManualHeader(t *testing.T, chapter creationManualChapter, content string) {
	lower := strings.ToLower(content)
	header := strings.SplitN(lower, "\n## ", 2)[0]
	if !strings.HasPrefix(content, "# "+chapter.File[:2]+". ") ||
		!strings.Contains(header, "responsabilidad:") || !strings.Contains(header, "alcance:") ||
		!strings.Contains(header, "no acredita:") ||
		!strings.Contains(lower, "## al terminar este capítulo, el lector sabrá") {
		t.Fatalf("%s incumple título, cabeceras o resultados de aprendizaje", chapter.File)
	}
}

func requireCreationManualPhrases(t *testing.T, name, text string, required []string) {
	text = strings.ToLower(text)
	for _, phrase := range required {
		if !strings.Contains(text, strings.ToLower(phrase)) {
			t.Fatalf("%s no contiene el tema obligatorio %q", name, phrase)
		}
	}
}

func requireCreationManualSources(t *testing.T, root string, got []creationManualSource) {
	want := []creationManualSource{
		{1, "AGENTS.md", "instrucciones y fronteras de trabajo"},
		{2, "HEAD:product/roadmap.json", "base versionada del catálogo, decisiones, dependencias y aceptación"},
		{3, "product/capabilities.json", "estado funcional acreditado por revisión"},
		{4, "product/evidence/", "evidencia de candidatos concretos"},
		{5, "docs/reconstruccion/ruta_total_100.md", "orden causal y compuertas globales"},
	}
	if len(got) != len(want) {
		t.Fatalf("fuentes del manual inválidas: %+v", got)
	}
	for position, source := range got {
		if source != want[position] {
			t.Fatalf("autoridad de fuente inválida: %+v", source)
		}
		if !strings.HasPrefix(source.Path, "HEAD:") {
			requireCreationManualContained(t, root, filepath.Join(root, filepath.FromSlash(source.Path)))
		}
	}
}
