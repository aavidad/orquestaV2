package orquestaappchange

import (
	"context"
	"strings"
	"sync"
)

type InMemoryAppChangeStoreV0 struct {
	mu      sync.RWMutex
	records []AppChangeRecordV0
}

var _ AppChangeRecordStorePortV0 = (*InMemoryAppChangeStoreV0)(nil)

func NewInMemoryAppChangeStoreV0(
	records ...AppChangeRecordV0,
) *InMemoryAppChangeStoreV0 {
	store := &InMemoryAppChangeStoreV0{}
	for _, record := range records {
		store.saveV0(normalizeAppChangeRecordV0(record))
	}
	return store
}

func (store *InMemoryAppChangeStoreV0) SaveAppChangeRequestV0(
	ctx context.Context,
	record AppChangeRecordV0,
) error {
	if err := appChangeContextErrV0(ctx); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.saveV0(normalizeAppChangeRecordV0(record))
	return nil
}

func (store *InMemoryAppChangeStoreV0) ListAppChangeRecordsV0(
	ctx context.Context,
	filter AppChangeRecordFilterV0,
) ([]AppChangeRecordV0, error) {
	if err := appChangeContextErrV0(ctx); err != nil {
		return nil, err
	}
	runRef := strings.TrimSpace(filter.RunRef)
	store.mu.RLock()
	defer store.mu.RUnlock()
	out := make([]AppChangeRecordV0, 0, len(store.records))
	for _, record := range store.records {
		if runRef != "" && record.Request.RunRef != runRef {
			continue
		}
		out = append(out, copyAppChangeRecordV0(record))
	}
	return out, nil
}

func (store *InMemoryAppChangeStoreV0) saveV0(record AppChangeRecordV0) {
	for index, existing := range store.records {
		if appChangeRecordKeyV0(existing) == appChangeRecordKeyV0(record) {
			store.records[index] = copyAppChangeRecordV0(record)
			return
		}
	}
	store.records = append(store.records, copyAppChangeRecordV0(record))
}

func appChangeContextErrV0(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func copyAppChangeRecordV0(record AppChangeRecordV0) AppChangeRecordV0 {
	record = normalizeAppChangeRecordV0(record)
	record.Request.CurrentStateRefs = append([]string(nil), record.Request.CurrentStateRefs...)
	record.Request.Scope = append([]string(nil), record.Request.Scope...)
	record.Request.AcceptanceCriteria = append([]string(nil), record.Request.AcceptanceCriteria...)
	record.Request.Constraints = append([]string(nil), record.Request.Constraints...)
	record.Request.AllowedWriteSet = append([]string(nil), record.Request.AllowedWriteSet...)
	record.Request.MetadataRefs = append([]string(nil), record.Request.MetadataRefs...)
	record.Request.ExternalWork = copyAppChangeExternalWorkV0(record.Request.ExternalWork)
	return record
}

func copyAppChangeExternalWorkV0(
	work *AppChangeExternalWorkV0,
) *AppChangeExternalWorkV0 {
	if work == nil {
		return nil
	}
	out := *work
	out.InterfaceRefs = append([]string(nil), work.InterfaceRefs...)
	out.WorkRefs = append([]string(nil), work.WorkRefs...)
	out.InputFields = copyDomainWorkFieldsFromAppChangeV0(work.InputFields)
	return &out
}

func normalizeAppChangeRecordV0(record AppChangeRecordV0) AppChangeRecordV0 {
	return AppChangeRecordV0{
		Request:     normalizeAppChangeRequestV0(record.Request),
		ReceivedAt:  strings.TrimSpace(record.ReceivedAt),
		RequestedBy: strings.TrimSpace(record.RequestedBy),
	}
}

func appChangeRecordKeyV0(record AppChangeRecordV0) string {
	request := normalizeAppChangeRequestV0(record.Request)
	return request.RunRef + "\x00" + request.ChangeRef
}
