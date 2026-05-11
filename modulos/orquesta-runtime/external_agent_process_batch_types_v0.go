package orquestaruntime

type ExternalAgentProcessBatchItemV0 struct {
	ItemRef  string                                `json:"item_ref,omitempty"`
	Spec     ExternalAgentLaunchSpecV0             `json:"spec"`
	Resolver ExternalAgentProcessCommandResolverV0 `json:"-"`
	Runtime  ExternalAgentProcessRuntimePortV0     `json:"-"`
}

type ExternalAgentProcessBatchResultV0 struct {
	Index         int                                `json:"index"`
	ItemRef       string                             `json:"item_ref,omitempty"`
	RequestID     string                             `json:"request_id,omitempty"`
	CorrelationID string                             `json:"correlation_id,omitempty"`
	ProfileRef    string                             `json:"profile_ref,omitempty"`
	ConnectorRef  string                             `json:"connector_ref,omitempty"`
	RuntimeKind   string                             `json:"runtime_kind,omitempty"`
	Result        ExternalAgentProcessLaunchResultV0 `json:"result"`
}
