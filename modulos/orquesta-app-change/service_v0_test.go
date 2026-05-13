package orquestaappchange

import (
	"context"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestRequestAppChangeV0PersisteYNotificaDirector(t *testing.T) {
	store := &fakeAppChangeStoreV0{}
	notifier := &fakeAppChangeNotifierV0{
		result: AppChangeDirectorNotificationV0{
			DirectorQuestionRef: "question-ref-change-001",
			EvidenceRefs:        []string{"change-ref-001"},
		},
	}
	result, err := RequestAppChangeV0(
		context.Background(),
		validAppChangeRequestForTestV0(),
		AppChangePortsV0{Store: store, DirectorNotifier: notifier},
	)
	if err != nil {
		t.Fatalf("RequestAppChangeV0: %v", err)
	}
	if result.Status != AppChangeStatusAcceptedV0 ||
		result.DirectorQuestionRef != "question-ref-change-001" ||
		store.saved.Request.ChangeRef != "change-ref-001" ||
		notifier.notified.Request.RunRef != "run-ref-agenda-001" ||
		store.saved.Request.ExternalWork == nil ||
		store.saved.Request.ExternalWork.ProjectRef != "project-ref-agenda" {
		t.Fatalf("resultado inesperado: %+v store=%+v notifier=%+v", result, store.saved, notifier.notified)
	}
}

func TestRequestAppChangeV0InvalidoNoTocaPuertos(t *testing.T) {
	store := &fakeAppChangeStoreV0{}
	notifier := &fakeAppChangeNotifierV0{}
	request := validAppChangeRequestForTestV0()
	request.RunRef = ""
	request.UserIntent = ""

	result, err := RequestAppChangeV0(
		context.Background(),
		request,
		AppChangePortsV0{Store: store, DirectorNotifier: notifier},
	)
	if err != nil {
		t.Fatalf("RequestAppChangeV0: %v", err)
	}
	if result.Status != AppChangeStatusInvalidV0 || len(result.Issues) != 2 {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if store.calls != 0 || notifier.calls != 0 {
		t.Fatalf("puertos tocados store=%d notifier=%d", store.calls, notifier.calls)
	}
}

func TestRequestAppChangeV0RechazaWriteSetInseguro(t *testing.T) {
	request := validAppChangeRequestForTestV0()
	request.AllowedWriteSet = []string{"../web"}

	result, err := RequestAppChangeV0(
		context.Background(),
		request,
		AppChangePortsV0{Store: &fakeAppChangeStoreV0{}, DirectorNotifier: &fakeAppChangeNotifierV0{}},
	)
	if err != nil {
		t.Fatalf("RequestAppChangeV0: %v", err)
	}
	if result.Status != AppChangeStatusInvalidV0 || result.Issues[0].Code != ErrAppChangeWriteSetInvalidV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRequestAppChangeV0RechazaRefsNoCompactas(t *testing.T) {
	request := validAppChangeRequestForTestV0()
	request.ChangeRef = "change/ref"

	result, err := RequestAppChangeV0(
		context.Background(),
		request,
		AppChangePortsV0{Store: &fakeAppChangeStoreV0{}, DirectorNotifier: &fakeAppChangeNotifierV0{}},
	)
	if err != nil {
		t.Fatalf("RequestAppChangeV0: %v", err)
	}
	if result.Status != AppChangeStatusInvalidV0 || result.Issues[0].Code != ErrAppChangeChangeRefInvalidV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRequestAppChangeV0RechazaExternalWorkNoCompacto(t *testing.T) {
	request := validAppChangeRequestForTestV0()
	request.ExternalWork = &AppChangeExternalWorkV0{
		ProjectRef: "project ref unsafe",
	}

	result, err := RequestAppChangeV0(
		context.Background(),
		request,
		AppChangePortsV0{Store: &fakeAppChangeStoreV0{}, DirectorNotifier: &fakeAppChangeNotifierV0{}},
	)
	if err != nil {
		t.Fatalf("RequestAppChangeV0: %v", err)
	}
	if result.Status != AppChangeStatusInvalidV0 || result.Issues[0].Code != ErrAppChangeExternalWorkRefV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRequestAppChangeV0RechazaExternalWorkInputFieldNoCompacto(t *testing.T) {
	request := validAppChangeRequestForTestV0()
	request.ExternalWork.InputFields = []orquestadomainwork.DomainWorkFieldV0{
		{Name: "topic id", Value: "topic-ref-week-view"},
	}

	result, err := RequestAppChangeV0(
		context.Background(),
		request,
		AppChangePortsV0{Store: &fakeAppChangeStoreV0{}, DirectorNotifier: &fakeAppChangeNotifierV0{}},
	)
	if err != nil {
		t.Fatalf("RequestAppChangeV0: %v", err)
	}
	if result.Status != AppChangeStatusInvalidV0 ||
		result.Issues[0].Code != ErrAppChangeExternalWorkFieldNameV0 ||
		result.Issues[0].Field != "external_work.input_fields" {
		t.Fatalf("result=%+v", result)
	}
}

func TestRequestAppChangeV0RequierePuertos(t *testing.T) {
	result, err := RequestAppChangeV0(context.Background(), validAppChangeRequestForTestV0(), AppChangePortsV0{})
	if err != nil {
		t.Fatalf("RequestAppChangeV0: %v", err)
	}
	if result.Status != AppChangeStatusInvalidV0 || len(result.Issues) != 2 {
		t.Fatalf("result=%+v", result)
	}
}

func validAppChangeRequestForTestV0() AppChangeRequestV0 {
	return AppChangeRequestV0{
		SchemaVersion:      AppChangeRequestSchemaV0,
		RequestID:          "req-change-001",
		CorrelationID:      "corr-change-001",
		RunRef:             "run-ref-agenda-001",
		AppRef:             "app-ref-agenda",
		ChangeRef:          "change-ref-001",
		Locale:             "es-ES",
		UserIntent:         "Quiero mejorar la web con vista semanal.",
		TargetArea:         "web",
		CurrentStateRefs:   []string{"delivery-ref-web-001"},
		AcceptanceCriteria: []string{"vista semanal visible"},
		AllowedWriteSet:    []string{"web/agenda", "internal/agenda/delivery"},
		ExternalWork: &AppChangeExternalWorkV0{
			ProjectRef:    "project-ref-agenda",
			JobRef:        "job-ref-agenda-week-view",
			InterfaceRefs: []string{"mcp-contract-ref-agenda-v0"},
			WorkKind:      "programming",
			WorkRefs:      []string{"domain-work-ref-agenda-week-view"},
			InputFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "topic_ref", Value: "topic-ref-week-view"},
				{Name: "source_refs", Values: []string{"source-ref-calendar"}},
			},
		},
	}
}

type fakeAppChangeStoreV0 struct {
	calls int
	saved AppChangeRecordV0
}

func (store *fakeAppChangeStoreV0) SaveAppChangeRequestV0(_ context.Context, record AppChangeRecordV0) error {
	store.calls++
	store.saved = record
	return nil
}

type fakeAppChangeNotifierV0 struct {
	calls    int
	notified AppChangeRecordV0
	result   AppChangeDirectorNotificationV0
}

func (notifier *fakeAppChangeNotifierV0) NotifyAppChangeRequestedV0(
	_ context.Context,
	record AppChangeRecordV0,
) (AppChangeDirectorNotificationV0, error) {
	notifier.calls++
	notifier.notified = record
	return notifier.result, nil
}
