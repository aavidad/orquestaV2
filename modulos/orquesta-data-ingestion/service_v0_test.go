package orquestadataingestion

import (
	"context"
	"testing"
)

func TestIngestDataV0PreservesSourceKindAndReceipt(t *testing.T) {
	identity := DataAdapterIdentityV0{AdapterRef: "fake", Version: "1"}
	result, err := IngestDataV0(context.Background(), DataIngestionRequestV0{SourceRef: "source:people", SchemaRef: "schema:person", ConfigurationRef: "config:1", ConfigurationHash: "sha256:config", Locale: "es", RequiredFieldRefs: []string{"field:name"}}, DataIngestionPortsV0{
		Source:       dataSourceFakeV0{identity: identity, source: DataSourceMaterialV0{DatasetRef: "dataset:people", SourceRef: "source:people", SourceKind: DataSourceKindXLSXV0, ContentHash: "sha256:data", SnapshotRef: "snapshot:people", ProvenanceRef: "provenance:source"}},
		Profiler:     dataProfilerFakeV0{identity: identity, profile: DataDatasetProfileV0{DatasetRef: "dataset:people", ProfileRef: "profile:people", Columns: []DataColumnV0{{ColumnRef: "column:name", Name: "name", DataType: "string"}}}},
		Mapper:       dataMapperFakeV0{identity: identity, mapping: DataMappingV0{MappingRef: "mapping:person", Fields: []DataFieldMappingV0{{FieldRef: "field:name", ColumnRef: "column:name"}}}},
		Validator:    dataValidatorFakeV0{identity: identity, result: DataValidationResultV0{ValidationRef: "validation:person", Accepted: true}},
		ReceiptStore: dataReceiptStoreFakeV0{identity: identity},
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if result.Receipt.ReceiptRef != "receipt:data" || result.Receipt.SourceKind != DataSourceKindXLSXV0 || result.Receipt.Status != "completed" {
		t.Fatalf("receipt=%+v", result.Receipt)
	}
}

func TestIngestDataV0RejectsInvalidSourceKind(t *testing.T) {
	identity := DataAdapterIdentityV0{AdapterRef: "fake", Version: "1"}
	_, err := IngestDataV0(context.Background(), DataIngestionRequestV0{SourceRef: "source:bad", SchemaRef: "schema:1", ConfigurationRef: "config:1", ConfigurationHash: "hash", Locale: "es"}, DataIngestionPortsV0{Source: dataSourceFakeV0{identity: identity, source: DataSourceMaterialV0{DatasetRef: "dataset:bad", SourceRef: "source:bad", SourceKind: "unknown", ContentHash: "hash", SnapshotRef: "snapshot", ProvenanceRef: "provenance"}}, Profiler: dataProfilerFakeV0{identity: identity}, Mapper: dataMapperFakeV0{identity: identity}, Validator: dataValidatorFakeV0{identity: identity}, ReceiptStore: dataReceiptStoreFakeV0{identity: identity}})
	if err == nil || err.Error() != "data_source_material_invalid" {
		t.Fatalf("err=%v", err)
	}
}

type dataSourceFakeV0 struct {
	identity DataAdapterIdentityV0
	source   DataSourceMaterialV0
}

func (fake dataSourceFakeV0) AdapterIdentityV0() DataAdapterIdentityV0 { return fake.identity }
func (fake dataSourceFakeV0) ResolveDataSourceV0(context.Context, string) (DataSourceMaterialV0, error) {
	return fake.source, nil
}

type dataProfilerFakeV0 struct {
	identity DataAdapterIdentityV0
	profile  DataDatasetProfileV0
}

func (fake dataProfilerFakeV0) AdapterIdentityV0() DataAdapterIdentityV0 { return fake.identity }
func (fake dataProfilerFakeV0) ProfileDataSetV0(context.Context, DataSourceMaterialV0) (DataDatasetProfileV0, error) {
	return fake.profile, nil
}

type dataMapperFakeV0 struct {
	identity DataAdapterIdentityV0
	mapping  DataMappingV0
}

func (fake dataMapperFakeV0) AdapterIdentityV0() DataAdapterIdentityV0 { return fake.identity }
func (fake dataMapperFakeV0) MapDataSetV0(context.Context, DataDatasetProfileV0, string, []string, string) (DataMappingV0, error) {
	return fake.mapping, nil
}

type dataValidatorFakeV0 struct {
	identity DataAdapterIdentityV0
	result   DataValidationResultV0
}

func (fake dataValidatorFakeV0) AdapterIdentityV0() DataAdapterIdentityV0 { return fake.identity }
func (fake dataValidatorFakeV0) ValidateDataMappingV0(context.Context, DataDatasetProfileV0, DataMappingV0) (DataValidationResultV0, error) {
	return fake.result, nil
}

type dataReceiptStoreFakeV0 struct{ identity DataAdapterIdentityV0 }

func (fake dataReceiptStoreFakeV0) AdapterIdentityV0() DataAdapterIdentityV0 { return fake.identity }
func (fake dataReceiptStoreFakeV0) StoreDataIngestionReceiptV0(context.Context, DataIngestionReceiptV0) (string, error) {
	return "receipt:data", nil
}
