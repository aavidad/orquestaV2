package orquestaruntime

func validateProcessRuntimeLaunchReceiptV0(
	receipt *ProcessRuntimeLaunchReceiptV0,
) []ProcessRuntimeErrorV0 {
	if receipt == nil {
		return nil
	}
	var issues []ProcessRuntimeErrorV0
	if receipt.SchemaVersion != ProcessRuntimeLaunchReceiptSchemaVersionV0 {
		issues = append(issues, processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "launch_receipt.schema_version"))
	}
	refs := []struct {
		field    string
		value    string
		required bool
	}{
		{"launch_receipt.receipt_ref", receipt.ReceiptRef, true},
		{"launch_receipt.command_ref", receipt.CommandRef, true},
		{"launch_receipt.executable_ref", receipt.ExecutableRef, true},
		{"launch_receipt.working_dir_ref", receipt.WorkingDirRef, false},
		{"launch_receipt.policy_ref", receipt.PolicyRef, true},
	}
	for _, ref := range refs {
		issues = append(issues, validateProcessRuntimeSnapshotRefV0(ref.field, ref.value, ref.required)...)
	}
	for _, ref := range receipt.ArgRefs {
		issues = append(issues, validateProcessRuntimeSnapshotRefV0("launch_receipt.arg_refs", ref, true)...)
	}
	for _, ref := range receipt.EnvRefs {
		issues = append(issues, validateProcessRuntimeSnapshotRefV0("launch_receipt.env_refs", ref, true)...)
	}
	if receipt.PolicyHash == "" {
		issues = append(issues, processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "launch_receipt.policy_hash"))
	}
	if receipt.EnvPolicy != ProcessRuntimeLaunchEnvPolicyAllowlistV0 {
		issues = append(issues, processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "launch_receipt.env_policy"))
	}
	if receipt.IO.StdoutPolicy != ProcessRuntimeLaunchIOPolicyDiscardV0 ||
		receipt.IO.StderrPolicy != ProcessRuntimeLaunchIOPolicyDiscardV0 ||
		!receipt.IO.OutputRedacted {
		issues = append(issues, processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "launch_receipt.io"))
	}
	return issues
}
