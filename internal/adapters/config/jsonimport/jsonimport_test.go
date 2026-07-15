package jsonimport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/config"
)

func TestCanonicalMappingsAreCompleteDetachedAndProductOwned(t *testing.T) {
	want := []string{
		"autoprogramming.checkpoint_only_high_consumption_tokens",
		"control_plane.token",
		"operator_director_mailbox.enabled",
		"runtime_models.enabled",
		"schema_version",
		"server.addr",
		"server.state_dir",
	}
	first := CanonicalMappings()
	if len(first) != len(want) {
		t.Fatalf("mapping count=%d want=%d", len(first), len(want))
	}
	for index, mapping := range first {
		if mapping.LegacyPath != want[index] {
			t.Fatalf("mapping[%d]=%q want=%q", index, mapping.LegacyPath, want[index])
		}
		switch mapping.Disposition {
		case DispositionMapped:
			if mapping.LegacyPath != "server.addr" || mapping.TargetKey != config.KeyServerListen ||
				mapping.Transform != transformIdentityString || mapping.Reason != "" {
				t.Fatalf("invalid mapped row: %+v", mapping)
			}
		case DispositionSchema, DispositionDeferred, DispositionSecretRequired:
			if mapping.TargetKey != "" || mapping.Transform != "" || mapping.Reason == "" {
				t.Fatalf("invalid accounting row: %+v", mapping)
			}
		default:
			t.Fatalf("unknown disposition: %+v", mapping)
		}
	}
	first[0].LegacyPath = "mutated"
	if CanonicalMappings()[0].LegacyPath != want[0] {
		t.Fatal("caller mutated canonical mapping authority")
	}
	optionsType := reflect.TypeOf(Options{})
	if optionsType.NumField() != 1 || optionsType.Field(0).Name != "Manager" {
		t.Fatalf("mapping injection surface added to Options: %v", optionsType)
	}
}

func TestDeclaredTransformIsExecutedAndUnknownTransformFailsClosed(t *testing.T) {
	mapping := Mapping{LegacyPath: "server.addr", Transform: transformIdentityString}
	value, err := applyTransform(mapping, "127.0.0.1:19090")
	if err != nil || value != "127.0.0.1:19090" {
		t.Fatalf("identity transform=%v err=%v", value, err)
	}
	mapping.Transform = "invented_transform"
	if _, err := applyTransform(mapping, "127.0.0.1:19090"); !HasErrorCode(err, ErrorMappingInvalid) {
		t.Fatalf("unknown transform error=%v", err)
	}
	mapping.Transform = transformIdentityString
	if _, err := applyTransform(mapping, true); !HasErrorCode(err, ErrorValueInvalid) {
		t.Fatalf("identity type error=%v", err)
	}
}

func TestPreviewAccountsEveryLeafDeterministicallyWithoutWrites(t *testing.T) {
	importer, _, store := openTestImporter(t)
	source := []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"127.0.0.1:19090"}}`)
	before := store.snapshot()
	first, err := importer.Preview(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := importer.Preview(context.Background(), append([]byte(nil), source...))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !first.Ready || first.SourceSHA256 != bytesSHA256(source) ||
		first.MappingSHA256 == "" || first.PlanSHA256 == "" || len(first.Entries) != 2 || len(first.changes) != 1 {
		t.Fatalf("unexpected compatible plan: %+v changes=%+v", first, first.changes)
	}
	if first.Entries[0].LegacyPath != "schema_version" || first.Entries[1].LegacyPath != "server.addr" ||
		first.Entries[1].TargetKey != config.KeyServerListen || first.Entries[1].Value != "127.0.0.1:19090" {
		t.Fatalf("unexpected entries: %+v", first.Entries)
	}
	if after := store.snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("Preview touched store: before=%+v after=%+v", before, after)
	}

	first.Entries[0].LegacyPath = "mutated"
	first.UnresolvedPaths = append(first.UnresolvedPaths, "mutated")
	again, err := importer.Preview(context.Background(), source)
	if err != nil || again.Entries[0].LegacyPath != "schema_version" || len(again.UnresolvedPaths) != 0 {
		t.Fatalf("caller mutated importer state: %+v err=%v", again, err)
	}
	whitespace, err := importer.Preview(context.Background(), append(source, '\n'))
	if err != nil {
		t.Fatal(err)
	}
	if whitespace.SourceSHA256 == second.SourceSHA256 || whitespace.PlanSHA256 == second.PlanSHA256 {
		t.Fatal("plan does not bind exact source bytes")
	}
}

func TestPreviewBlocksCurrentShapeAndNeverProjectsSecretValue(t *testing.T) {
	importer, _, store := openTestImporter(t)
	current := []byte(`{
  "schema_version":"orquesta_config.v0",
  "server":{"addr":"127.0.0.1:18080","state_dir":"./legacy-state"},
  "runtime_models":{"enabled":true},
  "operator_director_mailbox":{"enabled":false},
  "autoprogramming":{"checkpoint_only_high_consumption_tokens":true}
}`)
	plan, err := importer.Preview(context.Background(), current)
	if err != nil {
		t.Fatal(err)
	}
	wantUnresolved := []string{
		"autoprogramming.checkpoint_only_high_consumption_tokens",
		"operator_director_mailbox.enabled",
		"runtime_models.enabled",
		"server.state_dir",
	}
	if plan.Ready || !reflect.DeepEqual(plan.UnresolvedPaths, wantUnresolved) ||
		!reflect.DeepEqual(entryPaths(plan.Entries), []string{
			"autoprogramming.checkpoint_only_high_consumption_tokens",
			"operator_director_mailbox.enabled", "runtime_models.enabled", "schema_version", "server.addr", "server.state_dir",
		}) {
		t.Fatalf("current-shape plan false green: %+v", plan)
	}

	secret := "v09-super-secret-must-not-escape"
	secretSource := []byte(fmt.Sprintf(`{"schema_version":"orquesta_config.v0","control_plane":{"token":%q}}`, secret))
	secretPlan, err := importer.Preview(context.Background(), secretSource)
	if err != nil {
		t.Fatal(err)
	}
	serialized, marshalErr := json.Marshal(secretPlan)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	rawSecretSHA := bytesSHA256(secretSource)
	if secretPlan.Ready || secretPlan.SourceSHA256 != "" ||
		!reflect.DeepEqual(secretPlan.SecretRequiredPaths, []string{"control_plane.token"}) ||
		strings.Contains(fmt.Sprintf("%+v", secretPlan), secret) || bytes.Contains(serialized, []byte(secret)) {
		t.Fatalf("secret leaked or became ready: plan=%+v json=%s", secretPlan, serialized)
	}
	if bytes.Contains(serialized, []byte(rawSecretSHA)) {
		t.Fatalf("secret source fingerprint leaked: plan=%s", serialized)
	}
	beforeSecretApply := store.snapshot()
	_, applyErr := importer.Apply(context.Background(), ApplyRequest{
		Source: secretSource, ExpectedPlanSHA256: secretPlan.PlanSHA256, Confirm: true,
	})
	if !HasErrorCode(applyErr, ErrorPlanNotReady) || strings.Contains(fmt.Sprintf("%+v", applyErr), secret) {
		t.Fatalf("secret blocked error leaked or changed code: %v", applyErr)
	}
	if afterSecretApply := store.snapshot(); !reflect.DeepEqual(beforeSecretApply, afterSecretApply) {
		t.Fatalf("secret-blocked Apply touched Manager store: before=%+v after=%+v", beforeSecretApply, afterSecretApply)
	}
	for _, entry := range secretPlan.Entries {
		if entry.LegacyPath == "control_plane.token" && entry.Value != nil {
			t.Fatalf("secret retained in plan entry: %+v", entry)
		}
	}
	otherSecretPlan, err := importer.Preview(context.Background(), []byte(
		`{"schema_version":"orquesta_config.v0","control_plane":{"token":"different-low-entropy-secret"}}`,
	))
	if err != nil || otherSecretPlan.PlanSHA256 != secretPlan.PlanSHA256 {
		t.Fatalf("redacted secret plan retained source entropy: first=%+v other=%+v err=%v", secretPlan, otherSecretPlan, err)
	}
}

func TestPreviewRejectsAmbiguousMalformedAndUnaccountedJSON(t *testing.T) {
	importer, _, _ := openTestImporter(t)
	cases := []struct {
		name   string
		source []byte
		code   ErrorCode
	}{
		{"duplicate_nested", []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"a","addr":"b"}}`), ErrorSourceInvalid},
		{"duplicate_escape_equivalent", []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"a","\u0061ddr":"b"}}`), ErrorSourceInvalid},
		{"duplicate_root", []byte(`{"schema_version":"orquesta_config.v0","schema_version":"orquesta_config.v0"}`), ErrorSourceInvalid},
		{"trailing_value", []byte(`{"schema_version":"orquesta_config.v0"}{}`), ErrorSourceInvalid},
		{"trailing_bytes", []byte(`{"schema_version":"orquesta_config.v0"} nope`), ErrorSourceInvalid},
		{"missing_schema", []byte(`{"server":{"addr":"127.0.0.1:19090"}}`), ErrorSchemaRequired},
		{"unsupported_schema", []byte(`{"schema_version":"other.v0"}`), ErrorSchemaUnsupported},
		{"unknown_leaf", []byte(`{"schema_version":"orquesta_config.v0","unknown":true}`), ErrorUnknownPath},
		{"unknown_object", []byte(`{"schema_version":"orquesta_config.v0","unknown":{"value":true}}`), ErrorUnknownPath},
		{"dotted_segment", []byte(`{"schema_version":"orquesta_config.v0","server.addr":"value"}`), ErrorSourceInvalid},
		{"array", []byte(`{"schema_version":"orquesta_config.v0","server":[]}`), ErrorSourceInvalid},
		{"empty_object", []byte(`{"schema_version":"orquesta_config.v0","server":{}}`), ErrorSourceInvalid},
		{"wrong_mapped_type", []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":true}}`), ErrorValueInvalid},
		{"mapped_value_fails_registry", []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"0.0.0.0:19090"}}`), ErrorValueInvalid},
		{"wrong_deferred_type", []byte(`{"schema_version":"orquesta_config.v0","runtime_models":{"enabled":"true"}}`), ErrorValueInvalid},
		{"scalar_root", []byte(`true`), ErrorSourceInvalid},
		{"null_root", []byte(`null`), ErrorSourceInvalid},
		{"invalid_utf8", []byte{'{', '"', 0xff, '"', ':', '1', '}'}, ErrorSourceInvalid},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := importer.Preview(context.Background(), test.source)
			if !HasErrorCode(err, test.code) {
				t.Fatalf("error=%v want code=%s", err, test.code)
			}
		})
	}

	deep := []byte(`{"schema_version":"orquesta_config.v0","server":` + strings.Repeat(`{"x":`, maximumJSONDepth+1) +
		`true` + strings.Repeat(`}`, maximumJSONDepth+1) + `}`)
	if _, err := importer.Preview(context.Background(), deep); !HasErrorCode(err, ErrorSourceInvalid) {
		t.Fatalf("deep JSON error=%v", err)
	}
	oversized := bytes.Repeat([]byte{' '}, int(config.SourceMaxBytes())+1)
	if _, err := importer.Preview(context.Background(), oversized); !HasErrorCode(err, ErrorSourceTooLarge) {
		t.Fatalf("oversize error=%v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := importer.Preview(canceled, []byte(`{}`)); !HasErrorCode(err, ErrorContextInvalid) {
		t.Fatalf("canceled context error=%v", err)
	}
	if _, err := importer.Preview(nil, []byte(`{}`)); !HasErrorCode(err, ErrorContextInvalid) {
		t.Fatalf("nil context error=%v", err)
	}
}

func TestApplyUsesManagerCASReplayAndRejectsUnconfirmedOrStalePlan(t *testing.T) {
	importer, manager, store := openTestImporter(t)
	source := []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"127.0.0.1:19090"}}`)
	plan, err := importer.Preview(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	view, err := manager.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	base := ApplyRequest{
		Source: source, ExpectedPlanSHA256: plan.PlanSHA256, ExpectedRevision: view.SourceRevision,
		ActorRef: "actor:v09-test", RequestRef: "request:v09-test", Confirm: true,
	}
	before := store.snapshot()
	mismatch := base
	mismatch.ExpectedPlanSHA256 = strings.Repeat("0", len(plan.PlanSHA256))
	if _, err := importer.Apply(context.Background(), mismatch); !HasErrorCode(err, ErrorPlanMismatch) {
		t.Fatalf("plan mismatch error=%v", err)
	}
	unconfirmed := base
	unconfirmed.Confirm = false
	if _, err := importer.Apply(context.Background(), unconfirmed); !HasErrorCode(err, ErrorConfirmationRequired) {
		t.Fatalf("confirmation error=%v", err)
	}
	if after := store.snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("rejected Apply touched store: before=%+v after=%+v", before, after)
	}

	first, err := importer.Apply(context.Background(), base)
	if err != nil || first.Replayed || first.Receipt.BeforeRevision != view.SourceRevision ||
		first.Receipt.AfterRevision == first.Receipt.BeforeRevision ||
		!reflect.DeepEqual(first.Receipt.ChangedKeys, []config.Key{config.KeyServerListen}) {
		t.Fatalf("first Apply=%+v err=%v", first, err)
	}
	replayed, err := importer.Apply(context.Background(), base)
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Receipt, first.Receipt) {
		t.Fatalf("replayed Apply=%+v err=%v", replayed, err)
	}
	content := store.snapshot().document.Content
	explicit, err := config.ParseExplicit(content)
	if err != nil || explicit[config.KeyServerListen] != "127.0.0.1:19090" {
		t.Fatalf("TOML not committed through Manager: values=%v err=%v", explicit, err)
	}

	changedBytes := append(append([]byte(nil), source...), '\n')
	changedPlan, err := importer.Preview(context.Background(), changedBytes)
	if err != nil || changedPlan.PlanSHA256 == plan.PlanSHA256 {
		t.Fatalf("changed source plan=%+v err=%v", changedPlan, err)
	}
	changedRequest := base
	changedRequest.Source = changedBytes
	if _, err := importer.Apply(context.Background(), changedRequest); !HasErrorCode(err, ErrorPlanMismatch) {
		t.Fatalf("source/plan mismatch error=%v", err)
	}

	notReadySource := []byte(`{"schema_version":"orquesta_config.v0","server":{"state_dir":"./legacy"}}`)
	notReadyPlan, err := importer.Preview(context.Background(), notReadySource)
	if err != nil {
		t.Fatal(err)
	}
	notReadyRequest := base
	notReadyRequest.Source = notReadySource
	notReadyRequest.ExpectedPlanSHA256 = notReadyPlan.PlanSHA256
	notReadyRequest.ExpectedRevision = first.Receipt.AfterRevision
	notReadyRequest.RequestRef = "request:v09-not-ready"
	beforeNotReady := store.snapshot()
	if _, err := importer.Apply(context.Background(), notReadyRequest); !HasErrorCode(err, ErrorPlanNotReady) {
		t.Fatalf("not-ready error=%v", err)
	}
	if afterNotReady := store.snapshot(); !reflect.DeepEqual(beforeNotReady, afterNotReady) {
		t.Fatalf("not-ready Apply mutated store: before=%+v after=%+v", beforeNotReady, afterNotReady)
	}
}

func TestNewAndApplyRejectInvalidDependenciesAndContexts(t *testing.T) {
	if _, err := New(Options{}); !HasErrorCode(err, ErrorDependenciesRequired) {
		t.Fatalf("New nil manager error=%v", err)
	}
	var nilImporter *Importer
	if _, err := nilImporter.Preview(context.Background(), []byte(`{}`)); !HasErrorCode(err, ErrorDependenciesRequired) {
		t.Fatalf("nil Preview error=%v", err)
	}
	if _, err := nilImporter.Apply(context.Background(), ApplyRequest{}); !HasErrorCode(err, ErrorDependenciesRequired) {
		t.Fatalf("nil Apply error=%v", err)
	}
	importer, _, _ := openTestImporter(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := importer.Apply(canceled, ApplyRequest{Confirm: true}); !HasErrorCode(err, ErrorContextInvalid) {
		t.Fatalf("canceled Apply error=%v", err)
	}
}

func openTestImporter(t *testing.T) (*Importer, *config.Manager, *memoryDocumentStore) {
	t.Helper()
	active, err := config.Resolve(config.ResolveOptions{TOML: []byte{}})
	if err != nil {
		t.Fatal(err)
	}
	store := newMemoryDocumentStore()
	manager, err := config.NewManager(config.ManagerOptions{
		Store: store, Active: active,
		Now: func() time.Time { return time.Date(2026, 7, 15, 4, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	importer, err := New(Options{Manager: manager})
	if err != nil {
		t.Fatal(err)
	}
	return importer, manager, store
}

func entryPaths(entries []Entry) []string {
	result := make([]string, len(entries))
	for index, entry := range entries {
		result[index] = entry.LegacyPath
	}
	sort.Strings(result)
	return result
}

type memoryReceipt struct {
	fingerprint string
	receipt     config.ChangeReceipt
}

type memoryDocumentStore struct {
	mu       sync.Mutex
	document config.StoredDocument
	receipts map[string]memoryReceipt
	commits  int
}

type memoryStoreSnapshot struct {
	document config.StoredDocument
	receipts int
	commits  int
}

func newMemoryDocumentStore() *memoryDocumentStore {
	return &memoryDocumentStore{
		document: config.StoredDocument{Revision: config.Revision(bytesSHA256(nil)), Content: []byte{}},
		receipts: make(map[string]memoryReceipt),
	}
}

func (store *memoryDocumentStore) Read(ctx context.Context) (config.StoredDocument, error) {
	if ctx == nil {
		return config.StoredDocument{}, errors.New("context_required")
	}
	select {
	case <-ctx.Done():
		return config.StoredDocument{}, ctx.Err()
	default:
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return cloneStoredDocument(store.document), nil
}

func (store *memoryDocumentStore) Commit(ctx context.Context, request config.CommitRequest) (config.CommitResult, error) {
	if ctx == nil {
		return config.CommitResult{}, errors.New("context_required")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	identity := request.ActorRef + "\x00" + request.RequestRef
	if persisted, found := store.receipts[identity]; found {
		if persisted.fingerprint != request.Fingerprint {
			return config.CommitResult{}, &config.DocumentStoreError{Code: config.DocumentStoreReplayConflict}
		}
		return config.CommitResult{
			Document: cloneStoredDocument(store.document), Receipt: persisted.receipt, Replayed: true,
		}, nil
	}
	if request.ReplayOnly || request.ExpectedRevision != store.document.Revision {
		return config.CommitResult{}, &config.DocumentStoreError{Code: config.DocumentStoreRevisionConflict}
	}
	after := config.Revision(bytesSHA256(request.Replacement))
	receipt := config.ChangeReceipt{
		ReceiptRef: "receipt:" + strings.TrimPrefix(bytesSHA256([]byte(identity)), "sha256:"),
		ActorRef:   request.ActorRef, RequestRef: request.RequestRef, Fingerprint: request.Fingerprint,
		BeforeRevision: store.document.Revision, AfterRevision: after,
		ChangedKeys:        append([]config.Key(nil), request.ChangedKeys...),
		PendingRestartKeys: append([]config.Key(nil), request.PendingRestartKeys...),
		ChangedAt:          request.ChangedAt,
	}
	store.document = config.StoredDocument{Revision: after, Content: append([]byte(nil), request.Replacement...)}
	store.receipts[identity] = memoryReceipt{fingerprint: request.Fingerprint, receipt: receipt}
	store.commits++
	return config.CommitResult{Document: cloneStoredDocument(store.document), Receipt: receipt}, nil
}

func (store *memoryDocumentStore) snapshot() memoryStoreSnapshot {
	store.mu.Lock()
	defer store.mu.Unlock()
	return memoryStoreSnapshot{document: cloneStoredDocument(store.document), receipts: len(store.receipts), commits: store.commits}
}

func cloneStoredDocument(document config.StoredDocument) config.StoredDocument {
	document.Content = append([]byte(nil), document.Content...)
	return document
}
