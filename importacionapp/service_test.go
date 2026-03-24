package importacionapp

import (
	"errors"
	"testing"
)

type fakeStore struct {
	historyErr error
	wave2Err   error
	calls      []string
}

func (f *fakeStore) ImportLegacyProposalHistory() error {
	f.calls = append(f.calls, "history")
	return f.historyErr
}

func (f *fakeStore) ImportWave2Tasks() error {
	f.calls = append(f.calls, "wave2")
	return f.wave2Err
}

func TestImportHistoryRunsBothSteps(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)
	if err := svc.ImportHistory(); err != nil {
		t.Fatalf("ImportHistory: %v", err)
	}
	if len(store.calls) != 2 || store.calls[0] != "history" || store.calls[1] != "wave2" {
		t.Fatalf("orden inesperado: %#v", store.calls)
	}
}

func TestImportHistoryStopsOnLegacyError(t *testing.T) {
	t.Parallel()

	store := &fakeStore{historyErr: errors.New("boom")}
	svc := NewService(store)
	if err := svc.ImportHistory(); err == nil {
		t.Fatalf("expected error")
	}
	if len(store.calls) != 1 || store.calls[0] != "history" {
		t.Fatalf("llamadas inesperadas: %#v", store.calls)
	}
}
