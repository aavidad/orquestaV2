package orquestaautoprogramming

func autoprogrammingReviewGateCanonicalStemV0(stem string) string {
	switch stem {
	case "outside_write_set",
		"outside_writeset",
		"outside_of_write_set",
		"out_of_write_set",
		"outside_allowed_write_set",
		"outside_write_scope",
		"outside_scope",
		"out_of_scope",
		"fuera_write_set",
		"fuera_de_write_set",
		"fuera_del_write_set",
		"fuera_alcance",
		"fuera_de_alcance",
		"fuera_del_alcance",
		"file_outside_writeset",
		"file_outside_write_sets",
		"files_outside_write_set",
		"files_outside_write_sets",
		"file_outside_allowed_write_set",
		"file_outside_scope",
		"files_outside_scope",
		"fichero_fuera_write_set",
		"fichero_fuera_de_write_set",
		"fichero_fuera_alcance",
		"archivo_fuera_write_set",
		"archivo_fuera_de_write_set",
		"archivo_fuera_alcance",
		"artifact_outside_write_set",
		"artifact_outside_writeset",
		"artifacts_outside_write_set",
		"artifact_out_of_write_set",
		"artifact_path_outside_write_set",
		"artifact_path_out_of_write_set",
		"artifact_paths_outside_write_set",
		"path_outside_scope",
		"path_outside_write_set",
		"paths_outside_write_set":
		return "file_outside_write_set"
	case "missing_write_set_target",
		"write_set_missing_target",
		"missing_write_set_targets",
		"write_set_missing_targets",
		"write_set_target_absent",
		"write_set_targets_absent",
		"missing_write_set_destination",
		"write_set_destination_missing",
		"missing_write_set_path",
		"write_set_path_missing",
		"missing_write_set_entry",
		"write_set_entry_missing",
		"missing_target",
		"target_missing",
		"targets_missing",
		"target_missing_from_write_set",
		"destino_write_set_faltante",
		"destino_faltante",
		"ruta_write_set_faltante":
		return "write_set_target_missing"
	case "too_large_file",
		"too_large_files",
		"large_file",
		"large_files",
		"file_large",
		"files_large",
		"file_too_big",
		"files_too_big",
		"file_too_long",
		"files_too_long",
		"file_exceeds_limit",
		"file_size_over_limit",
		"file_over_line_limit",
		"file_line_limit_exceeded",
		"files_over_line_limit",
		"line_limit_exceeded",
		"max_lines_exceeded",
		"max_line_limit_exceeded",
		"line_count_exceeded",
		"too_many_lines",
		"oversized_file",
		"oversized_files",
		"fichero_demasiado_grande",
		"archivo_demasiado_grande":
		return "file_too_large"
	case "ack_pending_rail",
		"pending_rail",
		"rail_pendiente",
		"pending_rail_ack",
		"rail_pending",
		"rail_pending_ack",
		"ack_rail_pending",
		"pending_ack_rail",
		"pending_sensitive_rail",
		"sensitive_rail_pending",
		"soft_rail",
		"soft_rails",
		"rail_blando",
		"rails_blandos",
		"rail_dudoso",
		"rails_dudosos",
		"doubtful_rail",
		"suspect_rail",
		"detector_dudoso",
		"false_positive",
		"falso_positivo":
		return "ack_pending_rail"
	case "ack_sensitive_detail_redacted",
		"sensitive_detail_redacted",
		"sensitive_detail",
		"forbidden_sensitive_detail",
		"detalle_sensible",
		"detalle_sensible_redactado",
		"ack_detalle_prohibido",
		"sensitive_detail_requires_human_review":
		return "sensitive_detail_requires_review"
	case "artifact_without_ack_requires_human_review",
		"artifact_without_ack_requires_review",
		"artifact_without_ack",
		"artifacto_sin_ack",
		"artefacto_sin_ack":
		return "artifact_without_ack_requires_review"
	case "missing_required_test":
		return "required_test_missing"
	case "failed_required_test":
		return "required_test_failed"
	case "failed_test":
		return "test_failed"
	case "ack_absent":
		return "ack_missing"
	case "ack_not_complete":
		return "ack_not_completed"
	default:
		if destructive := autoprogrammingReviewGateDestructiveCanonicalStemV0(stem); destructive != "" {
			return destructive
		}
		return stem
	}
}
