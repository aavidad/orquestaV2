package orquestadataingestion

const (
	DataIngestionContractSchemaVersionV0 = "data_ingestion_contract.v0"
	DataIngestionReceiptSchemaVersionV0  = "data_ingestion_receipt.v0"
)

type DataSourceKindV0 string

const (
	DataSourceKindCSVV0      DataSourceKindV0 = "csv"
	DataSourceKindXLSXV0     DataSourceKindV0 = "xlsx"
	DataSourceKindODSV0      DataSourceKindV0 = "ods"
	DataSourceKindJSONV0     DataSourceKindV0 = "json"
	DataSourceKindParquetV0  DataSourceKindV0 = "parquet"
	DataSourceKindSQLiteV0   DataSourceKindV0 = "sqlite"
	DataSourceKindMySQLV0    DataSourceKindV0 = "mysql"
	DataSourceKindPostgresV0 DataSourceKindV0 = "postgres"
)

type DataAdapterIdentityV0 struct {
	AdapterRef string `json:"adapter_ref"`
	Version    string `json:"version"`
}

type DataSourceMaterialV0 struct {
	DatasetRef    string           `json:"dataset_ref"`
	SourceRef     string           `json:"source_ref"`
	SourceKind    DataSourceKindV0 `json:"source_kind"`
	ContentHash   string           `json:"content_hash"`
	SnapshotRef   string           `json:"snapshot_ref"`
	CursorRef     string           `json:"cursor_ref,omitempty"`
	ProvenanceRef string           `json:"provenance_ref"`
}

type DataColumnV0 struct {
	ColumnRef string `json:"column_ref"`
	Name      string `json:"name"`
	DataType  string `json:"data_type"`
	Nullable  bool   `json:"nullable"`
}

type DataDatasetProfileV0 struct {
	DatasetRef string         `json:"dataset_ref"`
	ProfileRef string         `json:"profile_ref"`
	Columns    []DataColumnV0 `json:"columns"`
	RowCount   int64          `json:"row_count,omitempty"`
}

type DataFieldMappingV0 struct {
	FieldRef  string `json:"field_ref"`
	ColumnRef string `json:"column_ref"`
}

type DataMappingV0 struct {
	MappingRef string               `json:"mapping_ref"`
	Fields     []DataFieldMappingV0 `json:"fields"`
}

type DataValidationResultV0 struct {
	ValidationRef string   `json:"validation_ref"`
	Accepted      bool     `json:"accepted"`
	IssueRefs     []string `json:"issue_refs,omitempty"`
}

type DataIngestionRequestV0 struct {
	SourceRef         string   `json:"source_ref"`
	SchemaRef         string   `json:"schema_ref"`
	ConfigurationRef  string   `json:"configuration_ref"`
	ConfigurationHash string   `json:"configuration_hash"`
	Locale            string   `json:"locale"`
	RequiredFieldRefs []string `json:"required_field_refs"`
}

type DataIngestionReceiptV0 struct {
	SchemaVersion     string                  `json:"schema_version"`
	ContractVersion   string                  `json:"contract_version"`
	ReceiptRef        string                  `json:"receipt_ref,omitempty"`
	DatasetRef        string                  `json:"dataset_ref"`
	SourceRef         string                  `json:"source_ref"`
	SourceKind        DataSourceKindV0        `json:"source_kind"`
	ContentHash       string                  `json:"content_hash"`
	SnapshotRef       string                  `json:"snapshot_ref"`
	SchemaRef         string                  `json:"schema_ref"`
	ConfigurationRef  string                  `json:"configuration_ref"`
	ConfigurationHash string                  `json:"configuration_hash"`
	ProfileRef        string                  `json:"profile_ref"`
	MappingRef        string                  `json:"mapping_ref"`
	ValidationRef     string                  `json:"validation_ref"`
	Adapters          []DataAdapterIdentityV0 `json:"adapters"`
	Status            string                  `json:"status"`
}

type DataIngestionResultV0 struct {
	Source     DataSourceMaterialV0   `json:"source"`
	Profile    DataDatasetProfileV0   `json:"profile"`
	Mapping    DataMappingV0          `json:"mapping"`
	Validation DataValidationResultV0 `json:"validation"`
	Receipt    DataIngestionReceiptV0 `json:"receipt"`
}
