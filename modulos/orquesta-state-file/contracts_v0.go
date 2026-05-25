package orquestastatefile

import (
	orquestacore "orquesta/modulos/orquesta-core"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

var _ orquestacionnucleoapp.RunStorePortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.EventSinkPortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.RunEventReaderPortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.WorkflowTaskStorePortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.WorkflowTaskByParentStorePortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.WorkflowTaskWriterPortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.WorkflowTaskWaitStateWriterPortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.WorkflowTaskWaitStateStorePortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.RequiredTestEvidenceStorePortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.OperationalDirectorPlanStateWriterPortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.AgentProcessRegistryPortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.AgentProcessRegistryListPortV0 = (*StoreV0)(nil)
var _ orquestacore.FunctionContractReadIndexPortV0 = (*StoreV0)(nil)
