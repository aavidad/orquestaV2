package orquestadataingestion

import "context"

type DataSourcePortV0 interface {
	AdapterIdentityV0() DataAdapterIdentityV0
	ResolveDataSourceV0(context.Context, string) (DataSourceMaterialV0, error)
}

type DataProfilerPortV0 interface {
	AdapterIdentityV0() DataAdapterIdentityV0
	ProfileDataSetV0(context.Context, DataSourceMaterialV0) (DataDatasetProfileV0, error)
}

type DataMappingPortV0 interface {
	AdapterIdentityV0() DataAdapterIdentityV0
	MapDataSetV0(context.Context, DataDatasetProfileV0, string, []string, string) (DataMappingV0, error)
}

type DataValidatorPortV0 interface {
	AdapterIdentityV0() DataAdapterIdentityV0
	ValidateDataMappingV0(context.Context, DataDatasetProfileV0, DataMappingV0) (DataValidationResultV0, error)
}

type DataReceiptStorePortV0 interface {
	AdapterIdentityV0() DataAdapterIdentityV0
	StoreDataIngestionReceiptV0(context.Context, DataIngestionReceiptV0) (string, error)
}

type DataIngestionPortsV0 struct {
	Source       DataSourcePortV0
	Profiler     DataProfilerPortV0
	Mapper       DataMappingPortV0
	Validator    DataValidatorPortV0
	ReceiptStore DataReceiptStorePortV0
}
