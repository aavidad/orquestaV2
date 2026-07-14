package config

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	managerFingerprintSchemaVersion = 1
	managerFingerprintDocumentType  = "orquesta.config.update.v1"
)

// Manager is the transport-neutral application service for desired
// configuration. Filesystem behavior remains behind DocumentStore.
type Manager struct {
	store         DocumentStore
	active        Snapshot
	environment   map[string]string
	sourcePath    string
	reservedPaths []string
	now           func() time.Time
	registry      registry
}

// NewManager validates immutable dependencies without reading the store.
func NewManager(options ManagerOptions) (*Manager, error) {
	if options.Store == nil || options.Now == nil {
		return nil, managerError(ErrorManagerInvalid, errors.New("config_manager_dependencies_required"))
	}
	loaded, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	if !snapshotMatchesRegistry(options.Active, loaded) {
		return nil, managerError(ErrorManagerInvalid, errors.New("config_manager_active_snapshot_invalid"))
	}
	environment := cloneEnvironment(options.Environment)
	if err := validateManagerEnvironment(loaded, environment); err != nil {
		return nil, err
	}
	return &Manager{
		store: options.Store, active: cloneSnapshot(options.Active), environment: environment,
		sourcePath: options.SourcePath, reservedPaths: append([]string(nil), options.ReservedPaths...),
		now: options.Now, registry: loaded,
	}, nil
}

// View resolves the latest desired document through the canonical registry.
func (manager *Manager) View(ctx context.Context) (ConfigView, error) {
	if err := manager.ready(ctx, ErrorManagerInvalid); err != nil {
		return ConfigView{}, err
	}
	document, err := manager.store.Read(ctx)
	if err != nil {
		return ConfigView{}, err
	}
	if err := validateManagerDocument(document); err != nil {
		return ConfigView{}, err
	}
	return manager.viewDocument(document)
}

// Update validates, canonicalizes and commits one confirmed mutation.
func (manager *Manager) Update(ctx context.Context, request UpdateRequest) (UpdateResult, error) {
	if err := manager.ready(ctx, ErrorUpdateInvalid); err != nil {
		return UpdateResult{}, err
	}
	changes, err := manager.normalizeUpdate(request)
	if err != nil {
		return UpdateResult{}, err
	}
	requestedKeys := managerChangeKeys(changes)
	fingerprint := updateFingerprint(request, changes)
	document, err := manager.store.Read(ctx)
	if err != nil {
		return UpdateResult{}, err
	}
	if err := validateManagerDocument(document); err != nil {
		return UpdateResult{}, err
	}
	if document.Revision != request.ExpectedRevision {
		return manager.commitUpdate(ctx, request, fingerprint, requestedKeys, document.Content, []Key{}, true)
	}
	explicit, err := ParseExplicit(document.Content)
	if err != nil {
		return UpdateResult{}, err
	}
	replacementValues := cloneExplicitValues(explicit)
	for _, change := range changes {
		if change.Unset {
			delete(replacementValues, change.Key)
			continue
		}
		replacementValues[change.Key] = cloneValue(change.Value)
	}
	replacement, err := RenderExplicit(replacementValues)
	if err != nil {
		return UpdateResult{}, err
	}
	desired, err := manager.resolve(replacement)
	if err != nil {
		return UpdateResult{}, err
	}
	changedKeys := changedManagerKeys(explicit, changes)
	if len(changedKeys) == 0 {
		return UpdateResult{}, managerError(ErrorUpdateNoChanges, errors.New("config_update_no_semantic_change"))
	}
	pendingKeys := manager.pendingRestartKeys(desired)
	return manager.commitUpdate(ctx, request, fingerprint, changedKeys, replacement, pendingKeys, false)
}

func (manager *Manager) commitUpdate(
	ctx context.Context,
	request UpdateRequest,
	fingerprint string,
	changedKeys []Key,
	replacement []byte,
	pendingKeys []Key,
	replayOnly bool,
) (UpdateResult, error) {
	if err := manager.ready(ctx, ErrorUpdateInvalid); err != nil {
		return UpdateResult{}, err
	}
	var changedAt time.Time
	if !replayOnly {
		changedAt = manager.now().UTC()
		if changedAt.IsZero() {
			return UpdateResult{}, managerError(ErrorUpdateInvalid, errors.New("config_update_time_invalid"))
		}
	}
	commitRequest := CommitRequest{
		ExpectedRevision:   request.ExpectedRevision,
		Replacement:        replacement,
		ActorRef:           request.ActorRef,
		RequestRef:         request.RequestRef,
		Fingerprint:        fingerprint,
		ChangedKeys:        changedKeys,
		PendingRestartKeys: sortedManagerKeys(pendingKeys),
		ChangedAt:          changedAt,
		ReplayOnly:         replayOnly,
	}
	committed, err := manager.store.Commit(ctx, commitRequest)
	if err != nil {
		return UpdateResult{}, err
	}
	if err := manager.validateCommitResult(commitRequest, committed); err != nil {
		return UpdateResult{}, err
	}
	view, err := manager.viewDocument(committed.Document)
	if err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{
		View: view, Receipt: cloneChangeReceipt(committed.Receipt), Replayed: committed.Replayed,
	}, nil
}

func (manager *Manager) validateCommitResult(request CommitRequest, result CommitResult) error {
	invalid := func() error {
		return managerError(ErrorStoreResultInvalid, errors.New("config_store_result_incoherent"))
	}
	if err := validateManagerDocument(result.Document); err != nil {
		return err
	}
	receipt := result.Receipt
	if !validManagerRef(receipt.ReceiptRef) || receipt.ActorRef != request.ActorRef ||
		receipt.RequestRef != request.RequestRef || receipt.Fingerprint != request.Fingerprint ||
		receipt.BeforeRevision != request.ExpectedRevision || !validManagerRevision(receipt.AfterRevision) ||
		receipt.AfterRevision == receipt.BeforeRevision || receipt.ChangedAt.IsZero() ||
		receipt.ChangedAt.Location() != time.UTC || !manager.validReceiptKeys(receipt.ChangedKeys, false) ||
		!manager.validReceiptKeys(receipt.PendingRestartKeys, true) {
		return invalid()
	}
	if request.ReplayOnly && !result.Replayed {
		return invalid()
	}
	if result.Replayed {
		if len(receipt.ChangedKeys) == 0 || !managerKeysSubset(receipt.ChangedKeys, request.ChangedKeys) {
			return invalid()
		}
		return nil
	}
	if !bytes.Equal(result.Document.Content, request.Replacement) ||
		result.Document.Revision != receipt.AfterRevision ||
		!reflect.DeepEqual(receipt.ChangedKeys, request.ChangedKeys) ||
		!reflect.DeepEqual(receipt.PendingRestartKeys, request.PendingRestartKeys) ||
		!receipt.ChangedAt.Equal(request.ChangedAt) {
		return invalid()
	}
	return nil
}

func (manager *Manager) validReceiptKeys(keys []Key, restartOnly bool) bool {
	if !sort.SliceIsSorted(keys, func(left, right int) bool { return keys[left] < keys[right] }) {
		return false
	}
	for index, key := range keys {
		definition, found := manager.registry.definition(key)
		if !found || (restartOnly && !definition.RestartRequired) || (index > 0 && keys[index-1] == key) {
			return false
		}
	}
	return true
}

func managerKeysSubset(subset, superset []Key) bool {
	available := make(map[Key]struct{}, len(superset))
	for _, key := range superset {
		available[key] = struct{}{}
	}
	for _, key := range subset {
		if _, found := available[key]; !found {
			return false
		}
	}
	return true
}

func validateManagerDocument(document StoredDocument) error {
	if validManagerRevision(document.Revision) && document.Revision == Revision(managerSHA256(document.Content)) {
		return nil
	}
	return managerError(ErrorStoreResultInvalid, errors.New("config_store_document_incoherent"))
}

func sortedManagerKeys(keys []Key) []Key {
	result := append([]Key{}, keys...)
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}

type normalizedManagerChange struct {
	Key   Key
	Value any
	Unset bool
}

func managerChangeKeys(changes []normalizedManagerChange) []Key {
	keys := make([]Key, len(changes))
	for index := range changes {
		keys[index] = changes[index].Key
	}
	return keys
}

func changedManagerKeys(before map[Key]any, changes []normalizedManagerChange) []Key {
	result := make([]Key, 0, len(changes))
	for _, change := range changes {
		previous, found := before[change.Key]
		if change.Unset {
			if found {
				result = append(result, change.Key)
			}
			continue
		}
		if !found || !reflect.DeepEqual(previous, change.Value) {
			result = append(result, change.Key)
		}
	}
	return result
}

func (manager *Manager) normalizeUpdate(request UpdateRequest) ([]normalizedManagerChange, error) {
	if !request.Confirm {
		return nil, managerError(ErrorUpdateConfirmation, nil)
	}
	if !validManagerRef(request.ActorRef) || !validManagerRef(request.RequestRef) ||
		!validManagerRevision(request.ExpectedRevision) || len(request.Changes) == 0 {
		return nil, managerError(ErrorUpdateInvalid, errors.New("config_update_identity_or_changes_invalid"))
	}
	result := make([]normalizedManagerChange, 0, len(request.Changes))
	seen := make(map[Key]struct{}, len(request.Changes))
	for _, change := range request.Changes {
		definition, found := manager.registry.definition(change.Key)
		if !found {
			return nil, &Error{Code: ErrorUnknownKey, Key: change.Key}
		}
		if _, duplicate := seen[change.Key]; duplicate {
			return nil, managerError(ErrorUpdateInvalid, errors.New("config_update_duplicate_key"))
		}
		seen[change.Key] = struct{}{}
		normalized := normalizedManagerChange{Key: change.Key, Unset: change.Unset}
		if change.Unset {
			if change.Value != nil {
				return nil, managerError(ErrorUpdateInvalid, errors.New("config_update_unset_has_value"))
			}
		} else {
			value, err := parseFileValue(definition, canonicalValue(change.Value))
			if err != nil {
				return nil, &Error{Code: ErrorValueInvalid, Key: change.Key, Cause: err}
			}
			normalized.Value = cloneValue(value)
		}
		result = append(result, normalized)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Key < result[right].Key })
	return result, nil
}

func (manager *Manager) viewDocument(document StoredDocument) (ConfigView, error) {
	desired, err := manager.resolve(document.Content)
	if err != nil {
		return ConfigView{}, err
	}
	return manager.buildView(document.Revision, desired), nil
}

func (manager *Manager) resolve(content []byte) (Snapshot, error) {
	resolve := func(sourcePath string) (Snapshot, error) {
		return Resolve(ResolveOptions{
			TOML: append([]byte(nil), content...), Environment: cloneEnvironment(manager.environment), SourcePath: sourcePath,
		})
	}
	desired, err := resolve(manager.sourcePath)
	if err != nil {
		return Snapshot{}, err
	}
	for _, reservedPath := range manager.reservedPaths {
		if reservedPath == "" || reservedPath == manager.sourcePath {
			continue
		}
		if _, err := resolve(reservedPath); err != nil {
			return Snapshot{}, err
		}
	}
	return desired, nil
}

func (manager *Manager) buildView(revision Revision, desired Snapshot) ConfigView {
	pendingKeys := manager.pendingRestartKeys(desired)
	pending := make(map[Key]struct{}, len(pendingKeys))
	for _, key := range pendingKeys {
		pending[key] = struct{}{}
	}
	keys := make([]ConfigKeyView, 0, len(manager.registry.keys))
	for _, definition := range manager.registry.keys {
		activeValue, _ := manager.active.value(definition.Key)
		desiredValue, _ := desired.value(definition.Key)
		metadata, _ := desired.Metadata(definition.Key)
		_, keyPending := pending[definition.Key]
		activeValue = cloneValue(canonicalValue(activeValue))
		desiredValue = cloneValue(canonicalValue(desiredValue))
		if definition.Sensitive {
			activeValue, desiredValue = redactedValue, redactedValue
		}
		keys = append(keys, ConfigKeyView{
			Key: definition.Key, ActiveValue: activeValue, DesiredValue: desiredValue,
			Source: metadata.Source, Type: string(definition.Type), Sensitive: definition.Sensitive,
			Scope: definition.Scope, RestartRequired: definition.RestartRequired, PendingRestart: keyPending,
		})
	}
	return ConfigView{
		SourceRevision: revision, ActiveHash: manager.active.Hash(), DesiredHash: desired.Hash(),
		PendingRestart: len(pendingKeys) > 0, PendingRestartKeys: append([]Key{}, pendingKeys...), Keys: keys,
	}
}

func (manager *Manager) pendingRestartKeys(desired Snapshot) []Key {
	result := make([]Key, 0)
	for _, definition := range manager.registry.keys {
		if !definition.RestartRequired {
			continue
		}
		activeValue, activeFound := manager.active.value(definition.Key)
		desiredValue, desiredFound := desired.value(definition.Key)
		if !activeFound || !desiredFound || !reflect.DeepEqual(activeValue, desiredValue) {
			result = append(result, definition.Key)
		}
	}
	return result
}

func (manager *Manager) ready(ctx context.Context, code ErrorCode) error {
	if manager == nil || manager.store == nil || ctx == nil {
		return managerError(code, errors.New("config_manager_or_context_invalid"))
	}
	select {
	case <-ctx.Done():
		return managerError(code, ctx.Err())
	default:
		return nil
	}
}

func updateFingerprint(request UpdateRequest, changes []normalizedManagerChange) string {
	type fingerprintChange struct {
		Key   Key  `json:"key"`
		Value any  `json:"value,omitempty"`
		Unset bool `json:"unset"`
	}
	type fingerprintDocument struct {
		SchemaVersion    int                 `json:"schema_version"`
		DocumentType     string              `json:"document_type"`
		ActorRef         string              `json:"actor_ref"`
		RequestRef       string              `json:"request_ref"`
		ExpectedRevision Revision            `json:"expected_revision"`
		Changes          []fingerprintChange `json:"changes"`
	}
	fingerprintChanges := make([]fingerprintChange, len(changes))
	for index, change := range changes {
		fingerprintChanges[index] = fingerprintChange{
			Key: change.Key, Value: cloneValue(canonicalValue(change.Value)), Unset: change.Unset,
		}
	}
	document := fingerprintDocument{
		SchemaVersion: managerFingerprintSchemaVersion, DocumentType: managerFingerprintDocumentType,
		ActorRef: request.ActorRef, RequestRef: request.RequestRef, ExpectedRevision: request.ExpectedRevision,
		Changes: fingerprintChanges,
	}
	payload, _ := json.Marshal(document)
	return managerSHA256(payload)
}

func managerSHA256(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validManagerRevision(revision Revision) bool {
	value := string(revision)
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func validManagerRef(value string) bool {
	if value == "" || len(value) > 1024 || strings.TrimSpace(value) != value || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func cloneExplicitValues(source map[Key]any) map[Key]any {
	result := make(map[Key]any, len(source))
	for key, value := range source {
		result[key] = cloneValue(value)
	}
	return result
}

func cloneEnvironment(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func validateManagerEnvironment(loaded registry, environment map[string]string) error {
	seen := make(map[Key]struct{}, len(environment))
	for name, raw := range environment {
		key, found := loaded.environmentTargets[name]
		if !found {
			key, found = loaded.environmentAliases[name]
		}
		if !found {
			return &Error{Code: ErrorUnknownKey, Key: Key(name)}
		}
		if _, duplicate := seen[key]; duplicate {
			return &Error{Code: ErrorValueInvalid, Key: key, Cause: errors.New("config_environment_alias_ambiguous")}
		}
		seen[key] = struct{}{}
		definition, _ := loaded.definition(key)
		if _, err := parseEnvironmentValue(definition, raw); err != nil {
			return &Error{Code: ErrorValueInvalid, Key: key, Cause: err}
		}
	}
	return nil
}

func cloneSnapshot(source Snapshot) Snapshot {
	result := source
	result.entries = make([]snapshotEntry, len(source.entries))
	for index, entry := range source.entries {
		result.entries[index] = snapshotEntry{key: entry.key, value: cloneValue(entry.value), metadata: cloneKeyMetadata(entry.metadata)}
	}
	return result
}

func snapshotMatchesRegistry(snapshot Snapshot, loaded registry) bool {
	if snapshot.schemaVersion != loaded.schemaVersion || snapshot.registryRevision != loaded.revision ||
		snapshot.registryHash != loaded.semanticHash || snapshot.hash == "" || len(snapshot.entries) != len(loaded.keys) {
		return false
	}
	for index, definition := range loaded.keys {
		if snapshot.entries[index].key != definition.Key {
			return false
		}
	}
	return true
}

func cloneChangeReceipt(receipt ChangeReceipt) ChangeReceipt {
	receipt.ChangedKeys = append([]Key{}, receipt.ChangedKeys...)
	receipt.PendingRestartKeys = append([]Key{}, receipt.PendingRestartKeys...)
	return receipt
}

func managerError(code ErrorCode, cause error) error {
	return &Error{Code: code, Cause: cause}
}
