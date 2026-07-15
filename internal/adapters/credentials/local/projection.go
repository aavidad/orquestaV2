package local

import (
	"encoding/json"

	"orquesta/internal/credentials"
)

// durableProjectionGate retains guards from both sides of a mutation. That
// keeps rotated or revoked material available until the final material-free
// durable projection has been checked.
type durableProjectionGate struct {
	guards []*credentials.LeakGuard
}

type durableDocumentProjection struct {
	SchemaVersion int                                                   `json:"schema_version"`
	DocumentType  string                                                `json:"document_type"`
	Revision      uint64                                                `json:"revision"`
	ParentDigest  string                                                `json:"parent_digest"`
	Records       map[credentials.CredentialRef]durableRecordProjection `json:"records"`
	Ledger        map[string]ledger                                     `json:"ledger"`
}

type durableRecordProjection struct {
	Metadata credentials.Metadata `json:"metadata"`
}

func newDurableProjectionGate(doc document) (*durableProjectionGate, error) {
	gate := &durableProjectionGate{}
	if err := gate.AddDocument(doc); err != nil {
		gate.Destroy()
		return nil, err
	}
	return gate, nil
}

func (gate *durableProjectionGate) AddDocument(doc document) error {
	if gate == nil {
		return credentials.NewError(credentials.ErrorStoreIO, "durable_projection_gate")
	}
	for _, item := range doc.Records {
		if len(item.Material) == 0 {
			continue
		}
		secret, err := credentials.NewSecret(item.Material)
		if err != nil {
			return err
		}
		guard, err := credentials.NewLeakGuard(secret)
		secret.Destroy()
		if err != nil {
			return err
		}
		gate.guards = append(gate.guards, guard)
	}
	return nil
}

func (gate *durableProjectionGate) Validate(doc document) error {
	if gate == nil {
		return credentials.NewError(credentials.ErrorStoreIO, "durable_projection_gate")
	}
	records := make(map[credentials.CredentialRef]durableRecordProjection, len(doc.Records))
	for ref, item := range doc.Records {
		records[ref] = durableRecordProjection{Metadata: item.Metadata}
	}
	payload, err := json.MarshalIndent(durableDocumentProjection{
		SchemaVersion: doc.SchemaVersion,
		DocumentType:  doc.DocumentType,
		Revision:      doc.Revision,
		ParentDigest:  doc.ParentDigest,
		Records:       records,
		Ledger:        doc.Ledger,
	}, "", "  ")
	if err != nil {
		return credentials.WrapError(credentials.ErrorStoreIO, "durable_projection", err)
	}
	payload = append(payload, '\n')
	defer clear(payload)
	surfaces := []credentials.LeakSurface{{Name: "durable_projection", Content: payload}}
	for _, guard := range gate.guards {
		if err := guard.Scan(surfaces); err != nil {
			return err
		}
	}
	return nil
}

func (gate *durableProjectionGate) Destroy() {
	if gate == nil {
		return
	}
	for _, guard := range gate.guards {
		guard.Destroy()
	}
	clear(gate.guards)
	gate.guards = nil
}
