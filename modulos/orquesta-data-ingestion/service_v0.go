package orquestadataingestion

import (
	"context"
	"fmt"
	"strings"
)

func IngestDataV0(ctx context.Context, request DataIngestionRequestV0, ports DataIngestionPortsV0) (DataIngestionResultV0, error) {
	if err := validateDataIngestionRequestV0(request); err != nil {
		return DataIngestionResultV0{}, err
	}
	if ports.Source == nil || ports.Profiler == nil || ports.Mapper == nil || ports.Validator == nil || ports.ReceiptStore == nil {
		return DataIngestionResultV0{}, fmt.Errorf("data_ingestion_port_required")
	}
	source, err := ports.Source.ResolveDataSourceV0(ctx, request.SourceRef)
	if err != nil {
		return DataIngestionResultV0{}, err
	}
	if source.SourceRef != request.SourceRef || !validDataSourceKindV0(source.SourceKind) || blankDataValueV0(source.DatasetRef) || blankDataValueV0(source.ContentHash) || blankDataValueV0(source.SnapshotRef) || blankDataValueV0(source.ProvenanceRef) {
		return DataIngestionResultV0{}, fmt.Errorf("data_source_material_invalid")
	}
	profile, err := ports.Profiler.ProfileDataSetV0(ctx, source)
	if err != nil {
		return DataIngestionResultV0{}, err
	}
	if profile.DatasetRef != source.DatasetRef || blankDataValueV0(profile.ProfileRef) || len(profile.Columns) == 0 {
		return DataIngestionResultV0{}, fmt.Errorf("data_profile_invalid")
	}
	mapping, err := ports.Mapper.MapDataSetV0(ctx, profile, request.SchemaRef, request.RequiredFieldRefs, request.Locale)
	if err != nil {
		return DataIngestionResultV0{}, err
	}
	if blankDataValueV0(mapping.MappingRef) {
		return DataIngestionResultV0{}, fmt.Errorf("data_mapping_invalid")
	}
	validation, err := ports.Validator.ValidateDataMappingV0(ctx, profile, mapping)
	if err != nil {
		return DataIngestionResultV0{}, err
	}
	if blankDataValueV0(validation.ValidationRef) {
		return DataIngestionResultV0{}, fmt.Errorf("data_validation_invalid")
	}
	receipt := DataIngestionReceiptV0{SchemaVersion: DataIngestionReceiptSchemaVersionV0, ContractVersion: DataIngestionContractSchemaVersionV0, DatasetRef: source.DatasetRef, SourceRef: source.SourceRef, SourceKind: source.SourceKind, ContentHash: source.ContentHash, SnapshotRef: source.SnapshotRef, SchemaRef: request.SchemaRef, ConfigurationRef: request.ConfigurationRef, ConfigurationHash: request.ConfigurationHash, ProfileRef: profile.ProfileRef, MappingRef: mapping.MappingRef, ValidationRef: validation.ValidationRef, Adapters: []DataAdapterIdentityV0{ports.Source.AdapterIdentityV0(), ports.Profiler.AdapterIdentityV0(), ports.Mapper.AdapterIdentityV0(), ports.Validator.AdapterIdentityV0(), ports.ReceiptStore.AdapterIdentityV0()}, Status: "completed"}
	if !validation.Accepted {
		receipt.Status = "completed_with_issues"
	}
	receiptRef, err := ports.ReceiptStore.StoreDataIngestionReceiptV0(ctx, receipt)
	if err != nil {
		return DataIngestionResultV0{}, err
	}
	receipt.ReceiptRef = receiptRef
	return DataIngestionResultV0{Source: source, Profile: profile, Mapping: mapping, Validation: validation, Receipt: receipt}, nil
}

func validateDataIngestionRequestV0(request DataIngestionRequestV0) error {
	if blankDataValueV0(request.SourceRef) || blankDataValueV0(request.SchemaRef) || blankDataValueV0(request.ConfigurationRef) || blankDataValueV0(request.ConfigurationHash) || blankDataValueV0(request.Locale) {
		return fmt.Errorf("data_ingestion_request_invalid")
	}
	return nil
}

func validDataSourceKindV0(kind DataSourceKindV0) bool {
	switch kind {
	case DataSourceKindCSVV0, DataSourceKindXLSXV0, DataSourceKindODSV0, DataSourceKindJSONV0, DataSourceKindParquetV0, DataSourceKindSQLiteV0, DataSourceKindMySQLV0, DataSourceKindPostgresV0:
		return true
	}
	return false
}

func blankDataValueV0(value string) bool { return strings.TrimSpace(value) == "" }
