package application

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/intake"
)

func TestIntakeDossierProjectionGeneratesCanonicalMinimumContentAndDiagrams(
	t *testing.T,
) {
	record := validIntakeChain(t)[2]
	projection, err := NewIntakeDossierProjection(
		record,
		validIntakeDossierProjectionSpec(),
	)
	if err != nil {
		t.Fatal(err)
	}
	input := projection.Input()
	if input.Statement != "Necesito coordinar calendarios compartidos" ||
		input.Objective != "Crear **agenda** [segura]" ||
		len(input.Sections) != len(intakeDossierProjectionSectionOrder) ||
		len(input.Diagrams) != len(intakeDossierProjectionDiagramOrder)+1 ||
		!reflect.DeepEqual(input.RiskRefs, []IntakeRiskRef{
			"intake-risk:calendar-provider",
			"intake-risk:offline-conflict",
		}) {
		t.Fatalf("projection envelope = %+v", input)
	}
	for index, section := range input.Sections {
		wantKind := intakeDossierProjectionSectionOrder[index]
		if section.Kind != wantKind ||
			section.Ref != IntakeDossierSectionRef(
				"intake-dossier-section:"+string(wantKind),
			) ||
			section.TitleKey != intake.MessageKey(
				"intake.dossier."+string(wantKind)+".title",
			) ||
			strings.TrimSpace(section.Markdown) == "" {
			t.Fatalf("section[%d] = %+v", index, section)
		}
	}
	product := input.Sections[0].Markdown
	if strings.Contains(product, "**agenda**") ||
		!strings.Contains(product, "\\*\\*agenda\\*\\*") ||
		!strings.Contains(product, "recommendation_rationale_key") ||
		!strings.Contains(product, "deviation_justification") {
		t.Fatalf("product projection did not escape/render typed facts:\n%s", product)
	}
	plan := input.Sections[9].Markdown
	for _, fact := range []string{
		"phases", "work_items", "role", "write_set", "required_tests",
		"arguments", "output_contract", "criterion_refs",
	} {
		if !strings.Contains(plan, fact) {
			t.Fatalf("plan missing %q:\n%s", fact, plan)
		}
	}
	for index, diagram := range input.Diagrams {
		wantPurpose := IntakeDossierDiagramDecisions
		if index < len(intakeDossierProjectionDiagramOrder) {
			wantPurpose = intakeDossierProjectionDiagramOrder[index]
		}
		if diagram.Purpose != wantPurpose ||
			diagram.Kind != IntakeDossierDiagramMermaid ||
			diagram.Ref != IntakeDossierDiagramRef(
				"intake-dossier-diagram:"+string(wantPurpose),
			) ||
			diagram.AltTextKey != intake.MessageKey(
				"intake.dossier.diagram."+string(wantPurpose)+".alt",
			) {
			t.Fatalf("diagram[%d] = %+v", index, diagram)
		}
		assertProjectionMermaidVerifiable(t, diagram.Source)
	}
	architecture := input.Diagrams[0].Source
	for _, node := range []string{
		"composition_root", "adapter_0001", "port_0001",
		"application", "domain",
	} {
		if !strings.Contains(architecture, node) {
			t.Fatalf("architecture diagram missing %q:\n%s", node, architecture)
		}
	}
	if err := validateIntakeDossierInput(input); err != nil {
		t.Fatalf("projector emitted invalid dossier input: %v", err)
	}
	if err := validateIntakeDossierPlan(projection.Plan()); err != nil {
		t.Fatalf("projector emitted invalid canonical plan: %v", err)
	}
	dossier, err := BuildIntakeDossier(
		record, projection.Plan(), projection.Input(),
	)
	if err != nil {
		t.Fatalf("projected input did not build canonical dossier: %v", err)
	}
	if !reflect.DeepEqual(dossier.Decisions(), record.State.Decisions()) ||
		!reflect.DeepEqual(dossier.Sections(), input.Sections) ||
		!reflect.DeepEqual(dossier.Diagrams(), input.Diagrams) {
		t.Fatal("built dossier lost projected or durable facts")
	}
}

func TestIntakeDossierProjectionIsStableAcrossEquivalentReordering(
	t *testing.T,
) {
	first := validIntakeDossierProjectionSpec()
	second := cloneIntakeDossierProjectionSpec(first)
	reverseIntakeDossierProjectionSpec(&second)
	record := validIntakeChain(t)[2]

	projectedFirst, err := NewIntakeDossierProjection(record, first)
	if err != nil {
		t.Fatal(err)
	}
	projectedSecond, err := NewIntakeDossierProjection(record, second)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projectedFirst.Input(), projectedSecond.Input()) ||
		!reflect.DeepEqual(projectedFirst.Plan(), projectedSecond.Plan()) ||
		!reflect.DeepEqual(projectedFirst.Decisions(), projectedSecond.Decisions()) ||
		!reflect.DeepEqual(
			projectedFirst.DecisionViews(), projectedSecond.DecisionViews(),
		) {
		t.Fatalf(
			"equivalent reorder changed projection:\nfirst=%+v\nsecond=%+v",
			projectedFirst.Input(), projectedSecond.Input(),
		)
	}
}

func TestIntakeDossierProjectionPreservesExactOrderSensitivePlan(
	t *testing.T,
) {
	spec := validIntakeDossierProjectionSpec()
	spec.Plan.Phases[0].Key = "phase.z-build"
	spec.Plan.Phases[1].Key = "phase.a-verify"
	spec.Plan.WorkItems[0].Phase = "phase.z-build"
	spec.Plan.WorkItems[1].Phase = "phase.a-verify"
	spec.Plan.WorkItems[0].Key = "work:z-build"
	spec.Plan.WorkItems[1].Key = "work:a-verify"
	spec.Plan.WorkItems[1].Parent = "work:z-build"
	spec.Plan.WorkItems[1].Dependencies = []string{"work:z-build"}
	slices.Reverse(spec.Plan.Phases[0].InputRefs)
	slices.Reverse(spec.Plan.Phases[0].CriterionRefs)
	slices.Reverse(spec.Plan.WorkItems[0].WriteSet)
	slices.Reverse(spec.Plan.WorkItems[0].RequiredTests)
	slices.Reverse(spec.Plan.WorkItems[0].SkillRefs)
	slices.Reverse(spec.Plan.WorkItems[0].ToolRefs)
	slices.Reverse(spec.Plan.WorkItems[0].CapabilityRefs)

	projection, err := NewIntakeDossierProjection(
		validIntakeChain(t)[2], spec,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projection.Plan(), spec.Plan) {
		t.Fatalf(
			"order-sensitive PlanSpec changed:\nwant=%+v\ngot=%+v",
			spec.Plan, projection.Plan(),
		)
	}
	markdown := projection.Input().Sections[9].Markdown
	assertBefore := func(first, second string) {
		t.Helper()
		firstIndex, secondIndex := strings.Index(markdown, first), strings.Index(markdown, second)
		if firstIndex < 0 || secondIndex < 0 || firstIndex >= secondIndex {
			t.Fatalf("%q not before %q in plan:\n%s", first, second, markdown)
		}
	}
	assertBefore("- `phase.z-build`", "- `phase.a-verify`")
	assertBefore("- `work:z-build`", "- `work:a-verify`")
	assertBefore("required-test:build-vet", "required-test:build-unit")
	for _, ordered := range []string{
		"`input:dossier`, `input:architecture`",
		"`internal/domain`, `internal/application`",
		"`skill:programming`, `skill:architecture`",
		"`tool:go-vet`, `tool:go-test`",
		"`capability:repository-write`, `capability:repository-read`",
	} {
		if !strings.Contains(markdown, ordered) {
			t.Fatalf("plan did not preserve ordered list %q:\n%s", ordered, markdown)
		}
	}
}

func TestIntakeDossierProjectionCopiesInputsOutputsAndSupportsConcurrentReads(
	t *testing.T,
) {
	spec := validIntakeDossierProjectionSpec()
	projection, err := NewIntakeDossierProjection(validIntakeChain(t)[2], spec)
	if err != nil {
		t.Fatal(err)
	}
	wantInput := projection.Input()
	wantPlan := projection.Plan()
	wantDecisions := projection.Decisions()
	wantDecisionViews := projection.DecisionViews()

	spec.ProductScope.InScope[0].Summary = "caller mutation"
	spec.UsersRoles.Roles[0].PermissionSummaries[0].Summary = "caller mutation"
	spec.Plan.WorkItems[0].RequiredTests[0].Arguments[0] = "caller mutation"
	spec.Decisions[0].ChoiceSummary = "caller mutation"
	if !reflect.DeepEqual(projection.Input(), wantInput) ||
		!reflect.DeepEqual(projection.Plan(), wantPlan) ||
		!reflect.DeepEqual(projection.Decisions(), wantDecisions) ||
		!reflect.DeepEqual(projection.DecisionViews(), wantDecisionViews) {
		t.Fatal("constructor retained caller-owned memory")
	}

	input := projection.Input()
	plan := projection.Plan()
	decisions := projection.Decisions()
	decisionViews := projection.DecisionViews()
	input.Sections[0].Markdown = "output mutation"
	input.Diagrams[0].Source = "output mutation"
	input.RiskRefs[0] = "intake-risk:output-mutation"
	plan.WorkItems[0].RequiredTests[0].Arguments[0] = "output mutation"
	decisions[0].Choice = "intake-option:output-mutation"
	decisionViews[0].ChoiceSummary = "output mutation"
	if !reflect.DeepEqual(projection.Input(), wantInput) ||
		!reflect.DeepEqual(projection.Plan(), wantPlan) ||
		!reflect.DeepEqual(projection.Decisions(), wantDecisions) ||
		!reflect.DeepEqual(projection.DecisionViews(), wantDecisionViews) {
		t.Fatal("accessor exposed projector-owned memory")
	}

	var group sync.WaitGroup
	failures := make(chan struct{}, 1)
	for worker := 0; worker < 16; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for iteration := 0; iteration < 100; iteration++ {
				if !reflect.DeepEqual(projection.Input(), wantInput) ||
					!reflect.DeepEqual(projection.Plan(), wantPlan) ||
					!reflect.DeepEqual(projection.Decisions(), wantDecisions) ||
					!reflect.DeepEqual(
						projection.DecisionViews(), wantDecisionViews,
					) {
					select {
					case failures <- struct{}{}:
					default:
					}
					return
				}
			}
		}()
	}
	group.Wait()
	select {
	case <-failures:
		t.Fatal("concurrent projection read changed immutable output")
	default:
	}
}

func TestIntakeDossierProjectionRejectsStructurallyIncompleteOrDivergentInput(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(*IntakeDossierProjectionSpec)
	}{
		{name: "blank objective", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.Objective = ""
		}},
		{name: "missing mandatory content", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.UIUX.Accessibility = nil
		}},
		{name: "duplicate semantic key", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.Data.Entities[1].Key = spec.Data.Entities[0].Key
		}},
		{name: "flow without edge", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.UsersRoles.MainFlow = spec.UsersRoles.MainFlow[:1]
		}},
		{name: "duplicate flow order", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.UsersRoles.MainFlow[1].Order = spec.UsersRoles.MainFlow[0].Order
		}},
		{name: "fallback not declared", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.I18NL10N.FallbackLocale = "fr-FR"
		}},
		{name: "invalid decision ref", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.Decisions[0].QuestionRef = "question:invalid"
		}},
		{name: "duplicate decision", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.Decisions = append(spec.Decisions, spec.Decisions[0])
		}},
		{name: "missing risk", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.Risks = nil
		}},
		{name: "invalid risk status key", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.Risks[0].StatusKey = "INVALID KEY"
		}},
		{name: "plan without required tests", mutate: func(spec *IntakeDossierProjectionSpec) {
			for index := range spec.Plan.WorkItems {
				spec.Plan.WorkItems[index].RequiredTests = nil
			}
		}},
		{name: "plan without acceptance criterion", mutate: func(spec *IntakeDossierProjectionSpec) {
			for index := range spec.Plan.Phases {
				spec.Plan.Phases[index].CriterionRefs = nil
			}
		}},
		{name: "compiler invalid plan", mutate: func(spec *IntakeDossierProjectionSpec) {
			spec.Plan.WorkItems[0].Phase = "phase.unknown"
		}},
	}
	record := validIntakeChain(t)[2]
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := validIntakeDossierProjectionSpec()
			test.mutate(&spec)
			projection, err := NewIntakeDossierProjection(record, spec)
			if !errors.Is(err, ErrIntakeDossierInvalid) ||
				!reflect.DeepEqual(projection, IntakeDossierProjection{}) {
				t.Fatalf("projection=%+v err=%v", projection, err)
			}
			input, err := ProjectIntakeDossierInput(record, spec)
			if !errors.Is(err, ErrIntakeDossierInvalid) ||
				!reflect.DeepEqual(input, IntakeDossierInput{}) {
				t.Fatalf("one-shot input=%+v err=%v", input, err)
			}
		})
	}
}

func TestIntakeDossierProjectionEscapesMarkdownAndMermaidControlSyntax(
	t *testing.T,
) {
	spec := validIntakeDossierProjectionSpec()
	spec.Architecture.Domain = `Domain"] --> injected["edge`
	spec.ProductScope.InScope[0].Summary = `Texto [enlace](destino) **fuerte**`
	projection, err := NewIntakeDossierProjection(validIntakeChain(t)[2], spec)
	if err != nil {
		t.Fatal(err)
	}
	input := projection.Input()
	if strings.Contains(input.Sections[0].Markdown, "[enlace](destino)") ||
		strings.Contains(input.Sections[0].Markdown, "**fuerte**") {
		t.Fatalf("caller Markdown remained active:\n%s", input.Sections[0].Markdown)
	}
	architecture := input.Diagrams[0].Source
	if strings.Contains(architecture, `domain["Domain"] --> injected`) ||
		!strings.Contains(architecture, "&#34;") {
		t.Fatalf("caller Mermaid syntax remained active:\n%s", architecture)
	}
	assertProjectionMermaidVerifiable(t, architecture)
}

func TestIntakeDossierProjectionDiagramGrowthIsLinearAndIndicesAreUnbounded(
	t *testing.T,
) {
	if got := projectionDiagramIndex(9_999); got != "10000" {
		t.Fatalf("index 10000 = %q", got)
	}
	if got := projectionDiagramIndex(10_000); got != "10001" {
		t.Fatalf("index 10001 = %q", got)
	}

	const itemCount = 256
	items := make([]IntakeDossierProjectionItem, itemCount)
	for index := range items {
		items[index] = IntakeDossierProjectionItem{
			Key: "item", Summary: "Elemento",
		}
	}
	architecture := renderProjectionArchitectureDiagram(
		IntakeDossierArchitectureProjection{
			Domain: "Dominio", Application: "Aplicación",
			CompositionRoot: "Composición", Ports: items, Adapters: items,
		},
	)
	architectureLines := strings.Count(architecture, "\n") + 1
	if architectureLines > 4*itemCount+20 {
		t.Fatalf(
			"architecture output is not linear: items=%d lines=%d",
			itemCount, architectureLines,
		)
	}
	deployment := renderProjectionDeploymentDiagram(
		IntakeDossierDeployOperationsProjection{
			Environments: items, DeploymentUnits: items, Network: items,
			Observability: items, Backups: items, Updates: items, Rollback: items,
		},
	)
	deploymentLines := strings.Count(deployment, "\n") + 1
	if deploymentLines > 2*7*itemCount+30 {
		t.Fatalf(
			"deployment output is not linear: items=%d lines=%d",
			itemCount, deploymentLines,
		)
	}

	var large strings.Builder
	writeProjectionDiagramItems(
		&large, "item", make([]IntakeDossierProjectionItem, 10_001),
	)
	if !strings.Contains(large.String(), `item_10001[""]`) {
		t.Fatal("diagram item helper did not safely render index above 10000")
	}
}

func TestIntakeDossierProjectionBindsDecisionViewsToDurableIntakeRecord(
	t *testing.T,
) {
	record := validIntakeChain(t)[2]
	spec := validIntakeDossierProjectionSpec()
	projection, err := NewIntakeDossierProjection(record, spec)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projection.Decisions(), record.State.Decisions()) {
		t.Fatalf(
			"projection decisions=%+v durable=%+v",
			projection.Decisions(), record.State.Decisions(),
		)
	}

	substituted := validIntakeDossierProjectionSpec()
	substituted.Decisions[0].QuestionRef = "intake-question:substituted"
	if _, err := NewIntakeDossierProjection(record, substituted); !errors.Is(err, ErrIntakeDossierInvalid) {
		t.Fatalf("substituted decision view accepted: %v", err)
	}
}

func TestIntakeDossierProjectionPreservesDurableDecisionOrderWhileViewsReorder(
	t *testing.T,
) {
	durable := []intake.Decision{
		{QuestionRef: "intake-question:z-last-lexically"},
		{QuestionRef: "intake-question:a-first-lexically"},
	}
	views := []IntakeDossierDecisionProjection{
		{
			QuestionRef:   "intake-question:a-first-lexically",
			ChoiceSummary: "A", RecommendationSummary: "A",
		},
		{
			QuestionRef:   "intake-question:z-last-lexically",
			ChoiceSummary: "Z", RecommendationSummary: "Z",
		},
	}
	bound, err := bindIntakeDossierProjectionDecisions(views, durable)
	if err != nil {
		t.Fatal(err)
	}
	if bound[0].Decision.QuestionRef != durable[0].QuestionRef ||
		bound[1].Decision.QuestionRef != durable[1].QuestionRef ||
		bound[0].View.QuestionRef != durable[0].QuestionRef ||
		bound[1].View.QuestionRef != durable[1].QuestionRef {
		t.Fatalf("durable decision order changed: %+v", bound)
	}
}

func TestIntakeDossierProjectionRecordsUnjustifiedDeviationWithoutBlocking(
	t *testing.T,
) {
	record := validIntakeChain(t)[2]
	withJustification := validIntakeDossierProjectionSpec()
	projected, err := NewIntakeDossierProjection(record, withJustification)
	if err != nil {
		t.Fatal(err)
	}
	markdown := projected.Input().Sections[0].Markdown
	if !strings.Contains(markdown, "`deviation_justified` — `true`") ||
		!strings.Contains(
			markdown, "`deviation_justification_status` — `provided`",
		) {
		t.Fatalf("justified deviation markers missing:\n%s", markdown)
	}

	withoutJustification := validIntakeDossierProjectionSpec()
	withoutJustification.Decisions[0].DeviationJustification = ""
	projected, err = NewIntakeDossierProjection(record, withoutJustification)
	if err != nil {
		t.Fatalf("unjustified deviation blocked projection: %v", err)
	}
	markdown = projected.Input().Sections[0].Markdown
	if !strings.Contains(markdown, "`deviation_justified` — `false`") ||
		!strings.Contains(
			markdown, "`deviation_justification_status` — `not_provided`",
		) ||
		strings.Contains(markdown, "`deviation_justification` —") {
		t.Fatalf("unjustified deviation markers invalid:\n%s", markdown)
	}
}

func assertProjectionMermaidVerifiable(t *testing.T, source string) {
	t.Helper()
	lines := strings.Split(source, "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "flowchart ") {
		t.Fatalf("not a generated flowchart:\n%s", source)
	}
	nodes := make(map[string]struct{})
	type edge struct{ from, to string }
	var edges []edge
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if parts := strings.Split(line, " --> "); len(parts) == 2 {
			edges = append(edges, edge{from: parts[0], to: parts[1]})
			continue
		}
		nodeEnd := strings.Index(line, "[\"")
		if nodeEnd <= 0 || !strings.HasSuffix(line, "\"]") {
			t.Fatalf("unverifiable Mermaid line %q", line)
		}
		nodes[line[:nodeEnd]] = struct{}{}
	}
	if len(nodes) < 2 || len(edges) == 0 {
		t.Fatalf("diagram has no verifiable graph:\n%s", source)
	}
	for _, edge := range edges {
		if _, found := nodes[edge.from]; !found {
			t.Fatalf("edge source %q is undefined:\n%s", edge.from, source)
		}
		if _, found := nodes[edge.to]; !found {
			t.Fatalf("edge target %q is undefined:\n%s", edge.to, source)
		}
	}
}

func validIntakeDossierProjectionSpec() IntakeDossierProjectionSpec {
	item := func(key, summary string) IntakeDossierProjectionItem {
		return IntakeDossierProjectionItem{Key: key, Summary: summary}
	}
	return IntakeDossierProjectionSpec{
		Statement: "Necesito coordinar calendarios compartidos",
		Objective: "Crear **agenda** [segura]",
		ProductScope: IntakeDossierProductScopeProjection{
			InScope: []IntakeDossierProjectionItem{
				item("calendar", "Calendario compartido y eventos"),
				item("reminders", "Recordatorios configurables"),
			},
			OutOfScope: []IntakeDossierProjectionItem{
				item("billing", "Cobros dentro de la aplicación"),
				item("video", "Videollamadas propias"),
			},
		},
		UsersRoles: IntakeDossierUsersRolesProjection{
			Users: []IntakeDossierProjectionItem{
				item("administrator", "Administra espacios y acceso"),
				item("member", "Consulta y edita eventos permitidos"),
			},
			Roles: []IntakeDossierProjectionRole{
				{
					Key: "administrator", Summary: "Responsable del espacio",
					PermissionSummaries: []IntakeDossierProjectionItem{
						item("members.manage", "Invita o revoca miembros"),
						item("settings.manage", "Cambia reglas del espacio"),
					},
				},
				{
					Key: "member", Summary: "Participante del espacio",
					PermissionSummaries: []IntakeDossierProjectionItem{
						item("events.read", "Consulta eventos autorizados"),
						item("events.write", "Edita eventos autorizados"),
					},
				},
			},
			MainFlow: []IntakeDossierProjectionStep{
				{Order: 1, Key: "sign_in", Summary: "La persona inicia sesión"},
				{Order: 2, Key: "select_calendar", Summary: "Elige un calendario"},
				{Order: 3, Key: "create_event", Summary: "Crea y comparte un evento"},
			},
		},
		Architecture: IntakeDossierArchitectureProjection{
			Rationale:   "Arquitectura hexagonal para aislar dominio y efectos externos",
			Domain:      "Reglas de calendarios, eventos, membresías y permisos",
			Application: "Casos de uso que coordinan dominio mediante puertos",
			Ports: []IntakeDossierProjectionItem{
				item("calendar_store", "Persistencia abstracta de calendarios"),
				item("notification_sender", "Envío abstracto de recordatorios"),
			},
			Adapters: []IntakeDossierProjectionItem{
				item("http", "Entrada HTTP autenticada"),
				item("sqlite", "Persistencia local sustituible"),
			},
			CompositionRoot: "Bootstrap único que elige adaptadores",
			CoreLimits: []IntakeDossierProjectionItem{
				item("no_database", "Dominio y aplicación no conocen la base de datos"),
				item("no_provider", "Dominio y aplicación no conocen proveedores"),
			},
		},
		Data: IntakeDossierDataProjection{
			Entities: []IntakeDossierProjectionItem{
				item("calendar", "Calendario con miembros y zona horaria"),
				item("event", "Evento versionado con participantes"),
			},
			Lifecycles: []IntakeDossierProjectionItem{
				item("calendar", "Activo, archivado y restaurado"),
				item("event", "Borrador, confirmado, cancelado y archivado"),
			},
			ImportExport: []IntakeDossierProjectionItem{
				item("ical", "Importación y exportación iCalendar"),
				item("json", "Exportación portable del espacio"),
			},
			Backups: []IntakeDossierProjectionItem{
				item("daily", "Copia diaria verificable"),
				item("restore", "Restauración ensayada"),
			},
			Versioning: []IntakeDossierProjectionItem{
				item("event_history", "Histórico inmutable de cambios de evento"),
				item("optimistic_lock", "Revisión esperada en escrituras"),
			},
		},
		Integrations: IntakeDossierIntegrationsProjection{
			Connectors: []IntakeDossierIntegrationProjection{
				{
					Key: "email", Summary: "Notificaciones por correo",
					Authentication: "Credencial referenciada por composición",
					DataFlow:       "Orquesta entrega una notificación mínima",
				},
				{
					Key: "google_calendar", Summary: "Sincronización opt-in",
					Authentication: "OAuth con alcance mínimo",
					DataFlow:       "Eventos autorizados viajan en ambas direcciones",
				},
			},
		},
		SecurityPrivacy: IntakeDossierSecurityPrivacyProjection{
			DataClasses: []IntakeDossierProjectionItem{
				item("personal", "Nombre, correo y disponibilidad"),
				item("sensitive", "El contenido sensible se minimiza"),
			},
			Authentication: []IntakeDossierProjectionItem{
				item("session", "Sesión autenticada y revocable"),
				item("service", "Identidad de servicio con alcance exacto"),
			},
			Authorization: []IntakeDossierProjectionItem{
				item("project_scope", "Permisos aislados por espacio"),
				item("role", "RBAC comprobado antes de cada caso de uso"),
			},
			Audit: []IntakeDossierProjectionItem{
				item("changes", "Cambios críticos quedan auditados"),
				item("exports", "Exportaciones dejan receipt"),
			},
			Retention: []IntakeDossierProjectionItem{
				item("events", "Retención configurable de eventos archivados"),
				item("logs", "Logs técnicos con plazo acotado"),
			},
			PrivacyRequirements: []IntakeDossierProjectionItem{
				item("erasure", "Borrado o anonimización gobernada"),
				item("rgpd", "Base jurídica y derechos documentados"),
			},
		},
		UIUX: IntakeDossierUIUXProjection{
			Views: []IntakeDossierProjectionItem{
				item("agenda", "Agenda diaria, semanal y mensual"),
				item("event_editor", "Editor accesible de evento"),
			},
			Navigation: []IntakeDossierProjectionItem{
				item("primary", "Navegación principal persistente"),
				item("shortcuts", "Atajos de teclado documentados"),
			},
			EmptyStates: []IntakeDossierProjectionItem{
				item("calendar", "Guía para crear el primer calendario"),
				item("events", "Acción para crear el primer evento"),
			},
			ErrorStates: []IntakeDossierProjectionItem{
				item("conflict", "Conflicto editable sin perder datos"),
				item("offline", "Estado sin conexión recuperable"),
			},
			Accessibility: []IntakeDossierProjectionItem{
				item("keyboard", "Flujo principal usable por teclado"),
				item("wcag", "Contraste y semántica WCAG 2.2 AA"),
			},
			Delivery: []IntakeDossierProjectionItem{
				item("pwa", "Web responsive instalable"),
				item("responsive", "Móvil, tableta y escritorio"),
			},
		},
		I18NL10N: IntakeDossierI18NL10NProjection{
			Locales: []IntakeDossierLocaleProjection{
				{Tag: "en-US", Summary: "English for United States"},
				{Tag: "es-ES", Summary: "Español de España"},
			},
			FallbackLocale: "es-ES",
			Catalog:        "Catálogo único con paridad de claves",
			Formats: []IntakeDossierProjectionItem{
				item("currency", "Moneda según locale"),
				item("datetime", "Fecha y hora con zona explícita"),
			},
			VisibleSurfaces: []IntakeDossierProjectionItem{
				item("cli", "CLI usa el catálogo"),
				item("web", "Web usa el catálogo"),
			},
			VisibleTextRule: "Ningún texto visible queda hardcodeado",
		},
		DeployOperations: IntakeDossierDeployOperationsProjection{
			Environments: []IntakeDossierProjectionItem{
				item("production", "Entorno productivo aislado"),
				item("staging", "Entorno previo equivalente"),
			},
			DeploymentUnits: []IntakeDossierProjectionItem{
				item("app", "Aplicación desplegable"),
				item("database", "Estado transaccional"),
			},
			Network: []IntakeDossierProjectionItem{
				item("domain", "Dominio gestionado"),
				item("https", "HTTPS obligatorio"),
			},
			Observability: []IntakeDossierProjectionItem{
				item("logs", "Logs estructurados sin secretos"),
				item("metrics", "Métricas de salud y latencia"),
			},
			Backups: []IntakeDossierProjectionItem{
				item("database", "Backup cifrado y restauración probada"),
				item("retention", "Retención operativa definida"),
			},
			Updates: []IntakeDossierProjectionItem{
				item("migration", "Migración compatible y ensayada"),
				item("release", "Release inmutable"),
			},
			Rollback: []IntakeDossierProjectionItem{
				item("application", "Reversión de aplicación"),
				item("data", "Plan seguro para datos"),
			},
		},
		Decisions: []IntakeDossierDecisionProjection{
			{
				QuestionRef:            "intake-question:audience",
				ChoiceSummary:          "Uso personal",
				RecommendationSummary:  "Uso compartido por un equipo",
				DeviationJustification: "La primera entrega será para una persona",
			},
		},
		Risks: []IntakeDossierRiskProjection{
			{
				Ref:        "intake-risk:calendar-provider",
				Summary:    "La API externa puede cambiar",
				Mitigation: "Adaptador sustituible y pruebas contractuales",
				StatusKey:  "intake.risk.calendar_provider.open",
			},
			{
				Ref:        "intake-risk:offline-conflict",
				Summary:    "Dos ediciones offline pueden competir",
				Mitigation: "Versionado y resolución explícita",
				StatusKey:  "intake.risk.offline_conflict.open",
			},
		},
		OpenIssues: []IntakeDossierProjectionItem{
			item("retention_days", "Acordar plazo final de retención"),
			item("sync_frequency", "Acordar frecuencia de sincronización"),
		},
		DeferredDecisions: []IntakeDossierProjectionItem{
			item("mobile_native", "Evaluar cliente móvil nativo después"),
			item("payments", "Reevaluar cobros en otra revisión"),
		},
		Plan:                    validIntakeDossierProjectionPlan(),
		IncludeDecisionsDiagram: true,
	}
}

func validIntakeDossierProjectionPlan() PlanSpec {
	return PlanSpec{
		Phases: []PhaseSpec{
			{
				Ref: "phase-instance:projection-build", Key: "phase.build",
				TemplateRef: "phase-template:build",
				InputRefs: []string{
					"input:architecture", "input:dossier",
				},
				CriterionRefs: []string{
					"criterion:architecture", "criterion:unit-tests",
				},
			},
			{
				Ref: "phase-instance:projection-verify", Key: "phase.verify",
				TemplateRef: "phase-template:verify",
				InputRefs: []string{
					"input:build", "input:dossier",
				},
				CriterionRefs: []string{
					"criterion:acceptance", "criterion:required-tests",
				},
			},
		},
		WorkItems: []WorkItemSpec{
			{
				Key: "work:build", Objective: "Implementar aplicación hexagonal",
				Phase: "phase.build", Role: goal.DefaultRoleKey().String(),
				WriteSet: []string{
					"internal/application", "internal/domain",
				},
				CouncilPolicy: council.PolicyRequired,
				RequiredTests: []RequiredTestSpec{
					{
						Ref: "required-test:build-unit", ToolRef: "tool:go-test",
						Arguments:        []string{"go", "test", "./internal/..."},
						WorkingDirectory: ".",
					},
					{
						Ref: "required-test:build-vet", ToolRef: "tool:go-vet",
						Arguments:        []string{"go", "vet", "./internal/..."},
						WorkingDirectory: ".",
					},
				},
				SkillRefs: []string{"skill:architecture", "skill:programming"},
				ToolRefs:  []string{"tool:go-test", "tool:go-vet"},
				CapabilityRefs: []string{
					"capability:repository-read", "capability:repository-write",
				},
				OutputContract: goal.OutputContractEvidenceBundle,
			},
			{
				Key: "work:verify", Objective: "Verificar criterios y artefactos",
				Phase: "phase.verify", Role: goal.DefaultRoleKey().String(),
				Parent: "work:build", Dependencies: []string{"work:build"},
				RequiredTests: []RequiredTestSpec{
					{
						Ref: "required-test:acceptance", ToolRef: "tool:go-test",
						Arguments:        []string{"go", "test", "./acceptance"},
						WorkingDirectory: ".",
					},
				},
				SkillRefs:      []string{"skill:review"},
				ToolRefs:       []string{"tool:go-test"},
				CapabilityRefs: []string{"capability:repository-read"},
				OutputContract: goal.OutputContractEvidenceBundle,
			},
		},
	}
}

func reverseIntakeDossierProjectionSpec(spec *IntakeDossierProjectionSpec) {
	for _, values := range [][]IntakeDossierProjectionItem{
		spec.ProductScope.InScope, spec.ProductScope.OutOfScope,
		spec.UsersRoles.Users, spec.Architecture.Ports,
		spec.Architecture.Adapters, spec.Architecture.CoreLimits,
		spec.Data.Entities, spec.Data.Lifecycles, spec.Data.ImportExport,
		spec.Data.Backups, spec.Data.Versioning, spec.SecurityPrivacy.DataClasses,
		spec.SecurityPrivacy.Authentication, spec.SecurityPrivacy.Authorization,
		spec.SecurityPrivacy.Audit, spec.SecurityPrivacy.Retention,
		spec.SecurityPrivacy.PrivacyRequirements, spec.UIUX.Views,
		spec.UIUX.Navigation, spec.UIUX.EmptyStates, spec.UIUX.ErrorStates,
		spec.UIUX.Accessibility, spec.UIUX.Delivery, spec.I18NL10N.Formats,
		spec.I18NL10N.VisibleSurfaces, spec.DeployOperations.Environments,
		spec.DeployOperations.DeploymentUnits, spec.DeployOperations.Network,
		spec.DeployOperations.Observability, spec.DeployOperations.Backups,
		spec.DeployOperations.Updates, spec.DeployOperations.Rollback,
		spec.OpenIssues, spec.DeferredDecisions,
	} {
		slices.Reverse(values)
	}
	slices.Reverse(spec.UsersRoles.Roles)
	for index := range spec.UsersRoles.Roles {
		slices.Reverse(spec.UsersRoles.Roles[index].PermissionSummaries)
	}
	slices.Reverse(spec.UsersRoles.MainFlow)
	slices.Reverse(spec.Integrations.Connectors)
	slices.Reverse(spec.I18NL10N.Locales)
	slices.Reverse(spec.Decisions)
	slices.Reverse(spec.Risks)
}
