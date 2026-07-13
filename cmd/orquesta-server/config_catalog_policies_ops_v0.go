package main

import orquestaconfig "orquesta/modulos/orquesta-config"

// serverProjectConfigOpsPolicyTableV0 names every admitted JSON leaf.
var serverProjectConfigOpsPolicyTableV0 = map[string]serverProjectConfigPolicySpecV0{
	"/codebase_broker/command": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/external_indexer_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/max_concurrent": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/project_name": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/provider_kind": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/state_dir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/timeout_ms": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/watchdog_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/watchdog_orphan_min_age_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codebase_broker/watchdog_stop_orphans": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/control_plane/permission_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/control_plane/principal": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/control_plane/public_reason": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/control_plane/remote_access_opt_in": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/control_plane/token": {
		Sensitive: true, EditabilityReason: "secret has no mutation API and must not be exposed through catalog editing",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/council/double_review_required": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/council/gate_required": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/council/members": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/daemon_logs/local_raw_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/daemon_logs/local_raw_reason": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/daemon_logs/max_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/daemon_logs/max_rotated_files": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/daemon_logs/retention_days": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/delivery_ledger_path": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/file_dir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/file_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/http_allowed_hosts": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/http_base_url": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/http_create_path": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/http_domain_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/http_egress_mode": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/http_submit_path": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/domain_work/http_timeout_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/local_model/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/local_model/evidence_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/local_model/model_path": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/local_model/model_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/local_model/runtime_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/sanitizer_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/sidecar/adapter_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/sidecar/command": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/sidecar/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/sidecar/evidence_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/sidecar/local_endpoint": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/sidecar/sidecar_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/egress_sanitizer/sidecar/transport_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/api_key_file": {
		Sensitive: false, EditabilityReason: "secret file reference has no mutation API and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/base_url": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/burst_connector_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/burst_tool": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/max_request_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/max_response_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/mcp_path": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/outbox_connector_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/outbox_tool": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/query_connector_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/query_tool": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/status_connector_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/status_tool": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/hermes_operator/timeout_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/operator_director_mailbox/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes/base_url": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes/default_max_attempts": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes/project_workdir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes/temporal_confirm": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes/timeout_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/allow_unfiltered": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/confirm": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/correlation_id": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/destination_evidence_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/dry_run": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/initial_delay_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/input_ledger_disabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/input_ledger_path": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/interval_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/job_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/job_type": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/job_type_sequence": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/limit": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/max_ticks": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/priority": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/productive_confirm": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/program_id": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/remote_qa/capability": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/remote_qa/capability_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/remote_qa/evidence_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/remote_qa/network_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/remote_qa/provider_quota_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/remote_qa/reason": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/remote_qa/tool_path_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/remote_qa_auth_state_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/require_runtime_compatibility": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/required_runtime_binary_sha256": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/required_runtime_build_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/required_runtime_commit_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis/capability": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis/capability_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis/evidence_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis/network_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis/provider_quota_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis/reason": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis/tool_path_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis_no_progress_timeout_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis_progress_heartbeat_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis_provider_timeout_ready": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis_tool_command": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis_tool_preflight_timeout_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/speech_synthesis_tool_workdir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/supervise_submitted": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/timeout_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/topic_id": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/wait_resident_interval_ms": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_bridge/wait_resident_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/batch_size": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/confirm": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/course_id": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/course_root": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/dry_run": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/interval_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/max_in_flight": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/max_ticks": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/queue_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/reconcile_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/reconcile_limit": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/registry_path": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/template_run_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_registry_finalpkg/template_topic_id": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_topic_registry/agent_id": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_topic_registry/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_topic_registry/force": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/opes_topic_registry/tool_path": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/rails_security/detail_prohibited_rails": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/rails_security/detail_prohibited_rails_scope": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/rails_security/rails_mode": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/rails_security/security_mode": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/schema_version": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server/addr": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server/audit_disabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server/audit_file": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server/state_dir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_http/control_body_max_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_http/idle_timeout_ms": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_http/max_header_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_http/read_header_timeout_ms": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_http/read_timeout_ms": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_http/write_timeout_ms": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/acceptance": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/after_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/area": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/branch_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/compact_rules": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/context_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/daily_context_budget_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/daily_goal_budget": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/disabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/evidence_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/frozen_tests_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/goal_first_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/max_requests": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/priority_score": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/project_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/project_workdir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/required_tests": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/target_queue": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/worktree_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle/write_set": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/acceptance": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/after_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/area": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/branch_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/compact_rules": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/context_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/daily_context_budget_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/daily_goal_budget": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/disabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/evidence_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/frozen_tests_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/goal_first_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/max_requests": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/priority_score": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/project_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/project_workdir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/required_tests": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/target_queue": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/worktree_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_idle_self_improvement/write_set": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_lifecycle/shutdown_grace_ms": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_resident_director/max_actions": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_supervisor/drain_max_commands": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_supervisor/drain_max_dispatches": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_supervisor/drain_max_external_waits": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_supervisor/drain_max_outbox": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_supervisor/max_executions_per_tick": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_supervisor/max_runs_per_tick": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/server_supervisor/queue_limit": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/startup_cleanup/mode": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/startup_cleanup/queue_limit": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/startup_cleanup/scope_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/telegram_operator/authorized_chat_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/telegram_operator/bot_link_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/telegram_operator/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/telegram_operator/notification_target_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/telegram_operator/require_confirmation": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/telegram_operator/token": {
		Sensitive: true, EditabilityReason: "secret has no mutation API and must not be exposed through catalog editing",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/wizard_bot/daily_token_budget": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/wizard_bot/llm_enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/wizard_bot/model": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/wizard_bot/reasoning_effort": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/worktree_snapshot/max_file_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/worktree_snapshot/max_files": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/worktree_snapshot/max_total_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
}

func serverProjectConfigOpsPolicyV0(leaf orquestaconfig.JSONLeafV0) (orquestaconfig.LeafPolicyV0, bool) {
	return serverProjectConfigPolicyFromExactTableV0(leaf, serverProjectConfigOpsPolicyTableV0)
}
