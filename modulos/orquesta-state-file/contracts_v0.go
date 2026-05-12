package orquestastatefile

import orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"

var _ orquestacionnucleoapp.RunStorePortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.EventSinkPortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.WorkflowTaskStorePortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.WorkflowTaskWriterPortV0 = (*StoreV0)(nil)
var _ orquestacionnucleoapp.AgentProcessRegistryPortV0 = (*StoreV0)(nil)
