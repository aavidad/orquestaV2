package lenguajeapp

import (
	"testing"
)

type fakeStore struct {
	policyInput     *LanguagePolicy
	policyUpdatedBy string
	matrixEntry     []string
	matrixUpdatedBy string
	resolveProject  string
	resolveContext  string
	resolveTask     *int64
	deleteScope     string
	deleteSelector  string
	deleteContext   string
	matrixEntries   []*LanguageMatrixEntry
	resolveResponse *LanguageResolution
	policyResponse  *LanguagePolicy
}

func (f *fakeStore) GetLanguagePolicy() (*LanguagePolicy, error) { return f.policyResponse, nil }
func (f *fakeStore) SetLanguagePolicy(p *LanguagePolicy, updatedBy string) error {
	f.policyInput = p
	f.policyUpdatedBy = updatedBy
	return nil
}
func (f *fakeStore) ListLanguageMatrixEntries() ([]*LanguageMatrixEntry, error) {
	return f.matrixEntries, nil
}
func (f *fakeStore) SetLanguageMatrixEntry(kind, selector, contexto, language, reason, updatedBy string) (*LanguageMatrixEntry, error) {
	f.matrixEntry = []string{kind, selector, contexto, language, reason}
	f.matrixUpdatedBy = updatedBy
	return &LanguageMatrixEntry{Scope: kind, Selector: selector, Context: contexto, Language: language, Reason: reason}, nil
}
func (f *fakeStore) DeleteLanguageMatrixEntry(kind, selector, contexto string) error {
	f.deleteScope = kind
	f.deleteSelector = selector
	f.deleteContext = contexto
	return nil
}
func (f *fakeStore) ResolveLanguage(project string, taskID *int64, contexto string) (*LanguageResolution, error) {
	f.resolveProject = project
	f.resolveTask = taskID
	f.resolveContext = contexto
	return f.resolveResponse, nil
}

func TestServiceNormalizesInputsAndDelegates(t *testing.T) {
	taskID := int64(42)
	store := &fakeStore{
		policyResponse:  &LanguagePolicy{DefaultLanguage: "es"},
		matrixEntries:   []*LanguageMatrixEntry{{Scope: "project", Selector: "orquestador"}},
		resolveResponse: &LanguageResolution{Idioma: "en", Origen: "matrix"},
	}
	service := NewService(store)

	if _, err := service.GetPolicy(); err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if err := service.SetPolicy(&LanguagePolicy{DefaultLanguage: "en"}, " Codex2 "); err != nil {
		t.Fatalf("SetPolicy: %v", err)
	}
	if store.policyUpdatedBy != "Codex2" {
		t.Fatalf("policyUpdatedBy=%q", store.policyUpdatedBy)
	}
	if _, err := service.ListMatrixEntries(); err != nil {
		t.Fatalf("ListMatrixEntries: %v", err)
	}
	if _, err := service.Resolve(ResolveInput{
		Proyecto: " orquestador ",
		TareaID:  &taskID,
		Contexto: " apps ",
	}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if store.resolveProject != "orquestador" || store.resolveContext != "apps" || store.resolveTask == nil || *store.resolveTask != 42 {
		t.Fatalf("resolve args project=%q context=%q task=%v", store.resolveProject, store.resolveContext, store.resolveTask)
	}
	if _, err := service.SetMatrixEntry(SetMatrixEntryInput{
		Scope:     " project ",
		Selector:  " orquestador ",
		Contexto:  " apps ",
		Language:  " en ",
		Reason:    " preferencia ",
		UpdatedBy: " Codex2 ",
	}); err != nil {
		t.Fatalf("SetMatrixEntry: %v", err)
	}
	wantMatrix := []string{"project", "orquestador", "apps", "en", "preferencia"}
	for i, want := range wantMatrix {
		if store.matrixEntry[i] != want {
			t.Fatalf("matrixEntry[%d]=%q, want %q", i, store.matrixEntry[i], want)
		}
	}
	if store.matrixUpdatedBy != "Codex2" {
		t.Fatalf("matrixUpdatedBy=%q", store.matrixUpdatedBy)
	}
	if err := service.DeleteMatrixEntry(" project ", " orquestador ", " apps "); err != nil {
		t.Fatalf("DeleteMatrixEntry: %v", err)
	}
	if store.deleteScope != "project" || store.deleteSelector != "orquestador" || store.deleteContext != "apps" {
		t.Fatalf("delete args scope=%q selector=%q context=%q", store.deleteScope, store.deleteSelector, store.deleteContext)
	}
}
