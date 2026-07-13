package main

import orquestaconfig "orquesta/modulos/orquesta-config"

// serverProjectConfigCorePolicyTableV0 names every admitted JSON leaf.
var serverProjectConfigCorePolicyTableV0 = map[string]serverProjectConfigPolicySpecV0{
	"/autoprogramming/checkpoint_only_high_consumption_tokens": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/checkpoint_only_max_wait_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/no_checkpoint_warning_max_wait_seconds": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/app_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/archive_dir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/commit_message": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/artifact_max_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/artifact_root": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/build_command": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/candidate_bin": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/command": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/command_effect_evidence_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/command_timeout": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/current_bin": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/health_timeout": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/last_good_bin": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_allow_broad_sandbox": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_branch_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_promotion_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_reasoning_effort": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_required_tests": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_run_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_runtime_dir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_sandbox": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_sandbox_evidence_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_worktree_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_codex_write_set": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/repair_command": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/runner_env_allowlist": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/skip_health_evidence_refs": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/state_dir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/guardian/test_commands": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/promotion/repo_ref": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/autoprogramming/required_test_attestation_config_file": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/required_test_runner/allowed_commands": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/required_test_runner/enabled": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/required_test_runner/environment": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/required_test_runner/go_command": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/required_test_runner/max_artifacts": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/required_test_runner/max_output_bytes": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
	"/required_test_runner/output_dir": {
		Sensitive: false, EditabilityReason: "the server has no field-level configuration setter; edit the canonical document before startup",
		RestartBehavior: orquestaconfig.RestartBehaviorRestartV0,
	},
}

func serverProjectConfigCorePolicyV0(leaf orquestaconfig.JSONLeafV0) (orquestaconfig.LeafPolicyV0, bool) {
	return serverProjectConfigPolicyFromExactTableV0(leaf, serverProjectConfigCorePolicyTableV0)
}
