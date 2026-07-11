package main

type serverProjectConfigServerV0 struct {
	Addr          *string `json:"addr,omitempty"`
	StateDir      *string `json:"state_dir,omitempty"`
	AuditFile     *string `json:"audit_file,omitempty"`
	AuditDisabled *bool   `json:"audit_disabled,omitempty"`
}

type serverProjectConfigServerHTTPV0 struct {
	ReadHeaderTimeoutMS *int `json:"read_header_timeout_ms,omitempty"`
	ReadTimeoutMS       *int `json:"read_timeout_ms,omitempty"`
	WriteTimeoutMS      *int `json:"write_timeout_ms,omitempty"`
	IdleTimeoutMS       *int `json:"idle_timeout_ms,omitempty"`
	MaxHeaderBytes      *int `json:"max_header_bytes,omitempty"`
	ControlBodyMaxBytes *int `json:"control_body_max_bytes,omitempty"`
}

type serverProjectConfigServerLifecycleV0 struct {
	ShutdownGraceMS *int `json:"shutdown_grace_ms,omitempty"`
}
