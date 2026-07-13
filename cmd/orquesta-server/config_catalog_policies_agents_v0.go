package main

import orquestaconfig "orquesta/modulos/orquesta-config"

// serverProjectConfigAgentPolicyTableV0 names every admitted JSON leaf.
var serverProjectConfigAgentPolicyTableV0 = map[string]serverProjectConfigPolicySpecV0{
	"/claude_model_routing/aliases": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_model_routing/policy_ref": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_model_routing/strict": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_model_routing/task_routes": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/command_path": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/enabled": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/extra_args": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/home_dir": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/output_format": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/path": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/permission_mode": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/project_work_dir": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/claude_runtime/runtime_work_dir": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_director/max_subagents_per_agent": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_director/recursive_agent_budget": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_director/wave_agents": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_model_routing/aliases": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_model_routing/policy_ref": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_model_routing/strict": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_model_routing/task_routes": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_runtime/execution_mode": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_runtime/max_batch_ready": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_runtime/max_concurrency": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_runtime/max_expected_seconds": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_runtime/reasoning_effort": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_runtime/runtime_work_dir": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_usage_accounting/log_max_bytes": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_usage_accounting/mode": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/agents": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/allow_unmanaged_launch": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/approval_policy": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/extra_args": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/isolate_home": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/model": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/path": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/profile": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/project_memories": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/projection_max_file_bytes": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/projection_max_files": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/projection_max_total_bytes": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/purge_runtime": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/purge_runtime_confirm": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/purge_runtime_report": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/reasoning_effort": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/runtime_workdir": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/sandbox": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/source_code_home": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/stop_confirm": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/stop_force": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/strict_credential_projection": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/tail_reason": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/unmanaged_launch_confirm": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/unmanaged_launch_reason": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/codex_wave/wave_ref": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/approval_mode": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/command_path": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/enabled": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/extra_args": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/home_dir": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/model": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/output_format": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/path": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/project_work_dir": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/gemini_runtime/runtime_work_dir": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/goal_backend/allow_app_server_proxy_diagnostic": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/goal_backend/kind": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/goal_backend/preflight_timeout_ms": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/goal_backend/prompt_locale": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/goal_backend/timeout_ms": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/runtime_models/allowed_models": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/runtime_models/base_url": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/runtime_models/bearer_token_file": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/runtime_models/enabled": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/runtime_models/timeout_seconds": {
		Sensitive: false, EditabilityReason: "agent runtime configuration has no live setter and is read from the canonical document at startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
}

func serverProjectConfigAgentsPolicyV0(leaf orquestaconfig.JSONLeafV0) (orquestaconfig.LeafPolicyV0, bool) {
	return serverProjectConfigPolicyFromExactTableV0(leaf, serverProjectConfigAgentPolicyTableV0)
}
