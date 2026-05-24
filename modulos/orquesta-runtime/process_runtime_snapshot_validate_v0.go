package orquestaruntime

func ValidateProcessRuntimeSnapshotV0(snapshot ProcessRuntimeSnapshotV0) []ProcessRuntimeErrorV0 {
	return validateProcessRuntimeSnapshotV0(snapshot)
}

func validateProcessRuntimeSnapshotV0(snapshot ProcessRuntimeSnapshotV0) []ProcessRuntimeErrorV0 {
	var issues []ProcessRuntimeErrorV0
	if snapshot.SchemaVersion != ProcessRuntimeConnectorVersionV0 {
		issues = append(issues, processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "schema_version"))
	}
	issues = append(issues, validateProcessRuntimeSnapshotRefV0("process_ref", snapshot.ProcessRef, true)...)
	issues = append(issues, validateProcessRuntimeSnapshotRefV0("session_ref", snapshot.SessionRef, false)...)
	issues = append(issues, validateProcessRuntimeSnapshotRefV0("launch_ref", snapshot.LaunchRef, false)...)
	issues = append(issues, validateProcessRuntimeSnapshotRefV0("stop_ref", snapshot.StopRef, false)...)
	switch snapshot.Status {
	case ProcessRuntimeRunningV0, ProcessRuntimeStoppedV0:
	default:
		issues = append(issues, processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "status"))
	}
	return issues
}

func validateProcessRuntimeSnapshotRefV0(
	field string,
	ref string,
	required bool,
) []ProcessRuntimeErrorV0 {
	if ref == "" {
		if required {
			return []ProcessRuntimeErrorV0{processRuntimeErrorV0(ProcessRuntimeRefInvalidaV0, field)}
		}
		return nil
	}
	if !opaqueRefPatternV0.MatchString(ref) || processRuntimeUnsafeValueV0(ref) ||
		processRuntimeOperationalPathUnsafeV0(ref) {
		return []ProcessRuntimeErrorV0{processRuntimeErrorV0(ProcessRuntimeRefInvalidaV0, field)}
	}
	return nil
}
