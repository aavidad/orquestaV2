package orquestaserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestRuntimeV0ResidentDirectorTickPersisteResultadoV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	director := &fakeResidentDirectorV0{results: []residentDirectorFakeResultV0{{
		result: ResidentDirectorResultV0{
			Status:          "completed",
			RunRef:          "run-ref-resident-director-001",
			ExecutedActions: 3,
			EvidenceRefs:    []string{"evidence-ref-resident-director-001"},
		},
	}}}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                   t.TempDir(),
		TickInterval:               time.Hour,
		ResidentDirectorEnabled:    true,
		ResidentDirectorMaxActions: 7,
		AuditDisabled:              true,
	}, RuntimeDepsV0{
		ResidentDirector: director,
		StateStore:       store,
		Clock:            fixedClockV0{now: time.Date(2026, 6, 8, 13, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runResidentDirectorTickV0(context.Background())

	if director.calls != 1 ||
		director.lastCommand.MaxActions != 7 ||
		store.last.ResidentDirectorStatus != "ok" ||
		store.last.ResidentDirectorLastResult != "completed" ||
		store.last.ResidentDirectorLastRunRef != "run-ref-resident-director-001" ||
		store.last.ResidentDirectorTicks != 1 ||
		store.last.ResidentDirectorExecutedActions != 3 ||
		store.last.ResidentDirectorErrorTicks != 0 {
		t.Fatalf("state=%+v director=%+v", store.last, director)
	}
}

func TestRuntimeV0ResidentDirectorTickErrorVisibleYRecuperaV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	director := &fakeResidentDirectorV0{results: []residentDirectorFakeResultV0{{
		result: ResidentDirectorResultV0{
			Status:          "external_pending",
			RunRef:          "run-ref-resident-director-error-001",
			ExecutedActions: 1,
			EvidenceRefs:    []string{"evidence-ref-resident-director-error"},
		},
		err: errors.New("fallo temporal del director residente"),
	}, {
		result: ResidentDirectorResultV0{
			Status:          "completed",
			RunRef:          "run-ref-resident-director-ok-001",
			ExecutedActions: 2,
		},
	}}}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                t.TempDir(),
		TickInterval:            time.Hour,
		ResidentDirectorEnabled: true,
		AuditDisabled:           true,
	}, RuntimeDepsV0{
		ResidentDirector: director,
		StateStore:       store,
		Clock:            fixedClockV0{now: time.Date(2026, 6, 8, 13, 5, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runResidentDirectorTickV0(context.Background())
	if store.last.ResidentDirectorStatus != "error" ||
		store.last.ResidentDirectorErrorTicks != 1 ||
		store.last.LastError == "" ||
		len(store.last.RecentErrors) != 1 ||
		store.last.RecentErrors[0].Scope != "resident_director" {
		t.Fatalf("error state=%+v", store.last)
	}

	runtime.runResidentDirectorTickV0(context.Background())
	if store.last.ResidentDirectorStatus != "ok" ||
		store.last.ResidentDirectorErrorTicks != 1 ||
		store.last.LastError != "" ||
		store.last.ResidentDirectorLastRunRef != "run-ref-resident-director-ok-001" ||
		store.last.ResidentDirectorExecutedActions != 3 {
		t.Fatalf("recovered state=%+v", store.last)
	}
}

func TestRuntimeV0ResidentDirectorAsyncCoalesceaUnTickPendienteV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	director := newBlockingResidentDirectorV0()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                t.TempDir(),
		TickInterval:            time.Hour,
		ResidentDirectorEnabled: true,
		AuditDisabled:           true,
	}, RuntimeDepsV0{
		ResidentDirector: director,
		StateStore:       store,
		Clock:            fixedClockV0{now: time.Date(2026, 6, 8, 13, 10, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	if !runtime.runResidentDirectorTickAsyncV0(context.Background()) {
		t.Fatalf("first async tick not started")
	}
	<-director.started
	if runtime.runResidentDirectorTickAsyncV0(context.Background()) {
		t.Fatalf("second async tick overlapped while first in flight")
	}
	director.release()
	director.waitDone(t)
	select {
	case <-director.started:
	case <-time.After(time.Second):
		t.Fatalf("pending resident director tick was not coalesced")
	}
	if runtime.runResidentDirectorTickAsyncV0(context.Background()) {
		t.Fatalf("third async tick overlapped while coalesced tick in flight")
	}
	director.release()
	director.waitDone(t)
	time.Sleep(10 * time.Millisecond)
	if store.last.ResidentDirectorTickActive {
		t.Fatalf("resident director tick stayed active: %+v", store.last)
	}
	if director.calls != 2 || store.last.ResidentDirectorTicks != 2 {
		t.Fatalf("calls=%d state=%+v", director.calls, store.last)
	}
}

func TestRuntimeV0ResidentDirectorOptInExplicitoV0(t *testing.T) {
	director := &fakeResidentDirectorV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		ResidentDirector: director,
		StateStore:       &memoryStateStoreV0{},
		Clock:            fixedClockV0{now: time.Date(2026, 6, 8, 13, 15, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	if runtime.runResidentDirectorTickAsyncV0(context.Background()) || director.calls != 0 {
		t.Fatalf("resident director started without opt-in")
	}
}

func TestRuntimeV0ResidentDirectorControlPausaYResumeV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	director := &fakeResidentDirectorV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                t.TempDir(),
		TickInterval:            time.Hour,
		ResidentDirectorEnabled: true,
		AuditDisabled:           true,
	}, RuntimeDepsV0{
		ResidentDirector: director,
		StateStore:       store,
		Clock:            fixedClockV0{now: time.Date(2026, 6, 8, 13, 18, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	pause := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(
		pause,
		httptest.NewRequest(http.MethodPost, serverResidentDirectorControlRoutePathV0, strings.NewReader(`{"action":"pause","reason":"sin_cuota"}`)),
	)
	if pause.Code != http.StatusOK {
		t.Fatalf("pause status=%d body=%s", pause.Code, pause.Body.String())
	}
	var paused serverResidentDirectorControlProjectionV0
	if err := json.Unmarshal(pause.Body.Bytes(), &paused); err != nil {
		t.Fatalf("pause json: %v", err)
	}
	if !paused.Paused || paused.Status != "paused" ||
		store.last.ResidentDirectorStatus != "paused" {
		t.Fatalf("paused=%+v state=%+v", paused, store.last)
	}
	if runtime.RequestResidentDirectorWakeupV0("test_paused") {
		t.Fatalf("wakeup aceptado con director residente pausado")
	}
	if runtime.runResidentDirectorTickAsyncV0(context.Background()) || director.calls != 0 {
		t.Fatalf("tick ejecutado con director residente pausado calls=%d", director.calls)
	}

	resume := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(
		resume,
		httptest.NewRequest(http.MethodPost, serverResidentDirectorControlRoutePathV0, strings.NewReader(`{"action":"resume","reason":"cuota_disponible"}`)),
	)
	if resume.Code != http.StatusOK {
		t.Fatalf("resume status=%d body=%s", resume.Code, resume.Body.String())
	}
	var resumed serverResidentDirectorControlProjectionV0
	if err := json.Unmarshal(resume.Body.Bytes(), &resumed); err != nil {
		t.Fatalf("resume json: %v", err)
	}
	if resumed.Paused || resumed.Status != "resumed" ||
		store.last.ResidentDirectorStatus != "resumed" {
		t.Fatalf("resumed=%+v state=%+v", resumed, store.last)
	}

	runtime.runResidentDirectorTickV0(context.Background())
	if director.calls != 1 || store.last.ResidentDirectorStatus != "ok" {
		t.Fatalf("tick tras resume no ejecuto: calls=%d state=%+v", director.calls, store.last)
	}
}

func TestRuntimeV0ResidentDirectorPanicQuedaComoErrorDurableV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                t.TempDir(),
		TickInterval:            time.Hour,
		ResidentDirectorEnabled: true,
		AuditDisabled:           true,
	}, RuntimeDepsV0{
		ResidentDirector: panicResidentDirectorV0{},
		StateStore:       store,
		Clock:            fixedClockV0{now: time.Date(2026, 6, 8, 13, 20, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runResidentDirectorTickV0(context.Background())

	if store.last.ResidentDirectorStatus != "error" ||
		store.last.ResidentDirectorLastResult != "panic" ||
		store.last.ResidentDirectorErrorTicks != 1 ||
		store.last.LastError == "" ||
		len(store.last.RecentErrors) != 1 {
		t.Fatalf("state=%+v", store.last)
	}
}

func TestResidentDirectorV0VisibleEnStatusPublicoYOperacional(t *testing.T) {
	now := time.Date(2026, 6, 8, 13, 25, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{}, now)
	state := tracker.MarkResidentDirectorV0(ResidentDirectorResultV0{
		Status:          "completed",
		RunRef:          "run-ref-resident-director-status-001",
		ExecutedActions: 4,
		EvidenceRefs:    []string{"evidence-ref-resident-director-status"},
	}, now)

	status := NewServerPublicStatusV0(state)
	counters := residentOperationalCountersV0(state)
	activity := residentOperationalActivityV0(state)
	if status.ResidentDirectorStatus != "ok" ||
		status.ResidentDirectorLastRunRef != "run-ref-resident-director-status-001" ||
		status.ResidentDirectorExecutedActions != 4 {
		t.Fatalf("status=%+v", status)
	}
	if counters["resident_director_ticks"] != 1 ||
		counters["resident_director_actions"] != 4 {
		t.Fatalf("counters=%+v", counters)
	}
	if !residentDirectorActivityFoundForTestV0(activity) {
		t.Fatalf("activity=%+v", activity)
	}
}

func TestResidentDirectorV0StatusPublicoNoReusaOkHistoricoSiEstaDesactivado(t *testing.T) {
	status := NewServerPublicStatusV0(StateV0{
		SchemaVersion:                   StateSchemaVersionV0,
		Status:                          "running",
		ResidentDirectorStatus:          "ok",
		ResidentDirectorTickActive:      true,
		ResidentDirectorTicks:           3,
		ResidentDirectorExecutedActions: 2,
		EffectiveConfig: ServerEffectiveConfigV0{
			Settings: []ServerConfigSettingV0{{
				Key:   "ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED",
				Value: "false",
			}},
		},
	})

	if status.ResidentDirectorStatus != "disabled" ||
		status.ResidentDirectorTickActive {
		t.Fatalf("status=%+v", status)
	}
}

type residentDirectorFakeResultV0 struct {
	result ResidentDirectorResultV0
	err    error
}

func residentDirectorActivityFoundForTestV0(
	activities []orquestaobservability.DiagnosticoActividadV0,
) bool {
	for _, activity := range activities {
		if activity.ActivityRef == "activity-ref-server-resident-director" {
			return true
		}
	}
	return false
}

type fakeResidentDirectorV0 struct {
	calls       int
	lastCommand ResidentDirectorCommandV0
	results     []residentDirectorFakeResultV0
}

func (fake *fakeResidentDirectorV0) RunResidentDirectorV0(
	_ context.Context,
	command ResidentDirectorCommandV0,
) (ResidentDirectorResultV0, error) {
	fake.calls++
	fake.lastCommand = command
	if len(fake.results) >= fake.calls {
		next := fake.results[fake.calls-1]
		return next.result, next.err
	}
	return ResidentDirectorResultV0{
		Status:          "completed",
		RunRef:          "run-ref-resident-director-default",
		ExecutedActions: 1,
	}, nil
}

type panicResidentDirectorV0 struct{}

func (panicResidentDirectorV0) RunResidentDirectorV0(
	context.Context,
	ResidentDirectorCommandV0,
) (ResidentDirectorResultV0, error) {
	panic("fallo panic residente")
}

type blockingResidentDirectorV0 struct {
	fakeResidentDirectorV0
	started  chan struct{}
	done     chan struct{}
	releaseC chan struct{}
}

func newBlockingResidentDirectorV0() *blockingResidentDirectorV0 {
	return &blockingResidentDirectorV0{
		started:  make(chan struct{}, 2),
		done:     make(chan struct{}, 2),
		releaseC: make(chan struct{}),
	}
}

func (director *blockingResidentDirectorV0) RunResidentDirectorV0(
	ctx context.Context,
	command ResidentDirectorCommandV0,
) (ResidentDirectorResultV0, error) {
	director.calls++
	director.lastCommand = command
	director.started <- struct{}{}
	select {
	case <-ctx.Done():
		return ResidentDirectorResultV0{}, ctx.Err()
	case <-director.releaseC:
		director.done <- struct{}{}
		return ResidentDirectorResultV0{
			Status:          "completed",
			RunRef:          "run-ref-resident-director-blocking",
			ExecutedActions: 1,
		}, nil
	}
}

func (director *blockingResidentDirectorV0) release() {
	director.releaseC <- struct{}{}
}

func (director *blockingResidentDirectorV0) waitDone(t *testing.T) {
	t.Helper()
	select {
	case <-director.done:
	case <-time.After(time.Second):
		t.Fatalf("resident director tick did not finish")
	}
}
