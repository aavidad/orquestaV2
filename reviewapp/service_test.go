package reviewapp

import (
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	projects map[string]*ProjectRef
	gates    map[int64]*Gate
	nextID   int64
}

func (f *fakeStore) GetProject(ref string) (*ProjectRef, error) {
	if item, ok := f.projects[ref]; ok {
		return item, nil
	}
	return nil, errors.New("project not found")
}

func (f *fakeStore) GetReviewGate(id int64) (*Gate, error) {
	if item, ok := f.gates[id]; ok {
		cp := *item
		return &cp, nil
	}
	return nil, nil
}

func (f *fakeStore) ListReviewGates(filter GateFilter) ([]*Gate, error) {
	var out []*Gate
	for _, item := range f.gates {
		if filter.ProyectoID != nil && item.ProyectoID != *filter.ProyectoID {
			continue
		}
		if filter.Estado != nil && item.Estado != *filter.Estado {
			continue
		}
		if filter.ReviewerAgente != nil && item.ReviewerAgente != *filter.ReviewerAgente {
			continue
		}
		cp := *item
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeStore) CreateReviewGate(gate *Gate) (int64, error) {
	f.nextID++
	cp := *gate
	cp.ID = f.nextID
	f.gates[cp.ID] = &cp
	return cp.ID, nil
}

func (f *fakeStore) UpdateReviewGate(gate *Gate) error {
	cp := *gate
	f.gates[cp.ID] = &cp
	return nil
}

func TestNormalizeGateState(t *testing.T) {
	state, err := NormalizeGateState(" En_Revision ")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if state != GateStateInReview {
		t.Fatalf("estado normalizado inesperado: %s", state)
	}
	if _, err := NormalizeGateState("inventado"); err == nil {
		t.Fatal("esperaba error para estado inválido")
	}
}

func TestCanTransitionGateState(t *testing.T) {
	if !CanTransitionGateState(GateStatePending, GateStateInReview) {
		t.Fatal("pending -> in_review debería permitirse")
	}
	if CanTransitionGateState(GateStateApproved, GateStatePending) {
		t.Fatal("approved no debería volver a pending")
	}
}

func TestCreateGateResolvesProjectAndDefaultsState(t *testing.T) {
	store := &fakeStore{
		projects: map[string]*ProjectRef{
			"orquestador": {ID: 7, Slug: "orquestador"},
		},
		gates:  map[int64]*Gate{},
		nextID: 100,
	}
	service := NewService(store)

	gate, err := service.Create(CreateGateInput{
		ProyectoRef:    "orquestador",
		RequestedBy:    "",
		ReviewerAgente: "CodexReview",
		SeverityMax:    "HIGH",
	})
	if err != nil {
		t.Fatalf("create gate: %v", err)
	}
	if gate.ID != 101 {
		t.Fatalf("id inesperado: %d", gate.ID)
	}
	if gate.Estado != GateStatePending {
		t.Fatalf("estado inicial inesperado: %s", gate.Estado)
	}
	if gate.RequestedBy != defaultRequestedByActor {
		t.Fatalf("actor por defecto inesperado: %s", gate.RequestedBy)
	}
	if gate.SeverityMax != "high" {
		t.Fatalf("severity_max inesperada: %s", gate.SeverityMax)
	}
}

func TestListFiltersByProjectStateAndReviewer(t *testing.T) {
	store := &fakeStore{
		projects: map[string]*ProjectRef{
			"orquestador": {ID: 7, Slug: "orquestador"},
		},
		gates: map[int64]*Gate{
			1: {ID: 1, ProyectoID: 7, Estado: GateStatePending, ReviewerAgente: "CodexReview"},
			2: {ID: 2, ProyectoID: 7, Estado: GateStateApproved, ReviewerAgente: "Otra"},
		},
	}
	service := NewService(store)

	items, err := service.List(ListInput{
		ProyectoRef:    "orquestador",
		Estado:         GateStatePending,
		ReviewerAgente: "CodexReview",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("lista filtrada inesperada: %+v", items)
	}
}

func TestResolveGateUpdatesStateReviewerAndResolvedAt(t *testing.T) {
	store := &fakeStore{
		projects: map[string]*ProjectRef{},
		gates: map[int64]*Gate{
			3: {
				ID:             3,
				ProyectoID:     7,
				Estado:         GateStateInReview,
				ReviewerAgente: "CodexReview",
				CreatedAt:      time.Now().UTC().Add(-time.Hour),
			},
		},
	}
	service := NewService(store)

	gate, err := service.Resolve(ResolveGateInput{
		ID:             3,
		Estado:         GateStateApproved,
		ReviewerAgente: "CodexReview2",
		FindingsJSON:   `{"summary":"ok"}`,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if gate.Estado != GateStateApproved {
		t.Fatalf("estado final inesperado: %s", gate.Estado)
	}
	if gate.ReviewerAgente != "CodexReview2" {
		t.Fatalf("reviewer inesperado: %s", gate.ReviewerAgente)
	}
	if gate.ResolvedAt == nil {
		t.Fatal("resolved_at debería quedar informado")
	}
	if store.gates[3].FindingsJSON != `{"summary":"ok"}` {
		t.Fatalf("findings no persistidos: %s", store.gates[3].FindingsJSON)
	}
}

func TestResolveGateRejectsInvalidTransition(t *testing.T) {
	store := &fakeStore{
		gates: map[int64]*Gate{
			4: {ID: 4, ProyectoID: 7, Estado: GateStateApproved},
		},
	}
	service := NewService(store)

	if _, err := service.Resolve(ResolveGateInput{ID: 4, Estado: GateStatePending}); err == nil {
		t.Fatal("esperaba error para transición inválida desde aprobado")
	}
}

func TestHelpersDeEstado(t *testing.T) {
	if !IsResolvedGateState(GateStateApproved) {
		t.Fatal("approved debería ser terminal")
	}
	if !IsBlockingGateState(GateStateBlocked) {
		t.Fatal("blocked debería ser bloqueante")
	}
	if !GateNeedsReviewAction(&Gate{Estado: GateStatePending}) {
		t.Fatal("pending debería requerir acción")
	}
	if GateNeedsReviewAction(&Gate{Estado: GateStateApproved}) {
		t.Fatal("approved no debería requerir acción")
	}
}

func TestUpdateGatePermiteCambiosSinResolver(t *testing.T) {
	store := &fakeStore{
		gates: map[int64]*Gate{
			5: {
				ID:             5,
				ProyectoID:     7,
				Estado:         GateStatePending,
				ReviewerAgente: "CodexReview",
				SeverityMax:    "medium",
			},
		},
	}
	service := NewService(store)
	reviewer := "CodexReview2"
	taskID := int64(77)
	worktreeID := int64(88)
	severity := "critical"
	findings := `{"summary":"needs work"}`

	gate, err := service.Update(UpdateGateInput{
		ID:             5,
		ReviewerAgente: &reviewer,
		TareaID:        &taskID,
		WorktreeID:     &worktreeID,
		SeverityMax:    &severity,
		FindingsJSON:   &findings,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if gate.ReviewerAgente != reviewer {
		t.Fatalf("reviewer inesperado: %s", gate.ReviewerAgente)
	}
	if gate.SeverityMax != "critical" {
		t.Fatalf("severity inesperada: %s", gate.SeverityMax)
	}
	if gate.FindingsJSON != findings {
		t.Fatalf("findings inesperados: %s", gate.FindingsJSON)
	}
	if gate.TareaID == nil || *gate.TareaID != taskID {
		t.Fatalf("tarea review inesperada: %+v", gate.TareaID)
	}
	if gate.WorktreeID == nil || *gate.WorktreeID != worktreeID {
		t.Fatalf("worktree review inesperada: %+v", gate.WorktreeID)
	}
	if gate.ResolvedAt != nil {
		t.Fatal("resolved_at no debería informarse sin aprobación")
	}
}

func TestUpdateGatePuedeResolverViaEstado(t *testing.T) {
	store := &fakeStore{
		gates: map[int64]*Gate{
			6: {
				ID:             6,
				ProyectoID:     7,
				Estado:         GateStateInReview,
				ReviewerAgente: "CodexReview",
			},
		},
	}
	service := NewService(store)
	findings := `{"summary":"ok"}`

	gate, err := service.Update(UpdateGateInput{
		ID:           6,
		Estado:       GateStateApproved,
		FindingsJSON: &findings,
	})
	if err != nil {
		t.Fatalf("update resolve: %v", err)
	}
	if gate.Estado != GateStateApproved {
		t.Fatalf("estado inesperado: %s", gate.Estado)
	}
	if gate.ResolvedAt == nil {
		t.Fatal("resolved_at debería quedar informado")
	}
}
