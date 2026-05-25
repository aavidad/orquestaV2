package orquestaautoprogramming

func autoprogrammingReviewGateDestructiveCanonicalStemV0(stem string) string {
	switch stem {
	case "removed",
		"file_removed",
		"deleted_path":
		return "removed_path"
	case "truncated",
		"strong_truncate":
		return "truncated_path"
	case "renamed",
		"moved_path",
		"renamed_or_moved":
		return "renamed_or_moved_path"
	case "large_delta_replacement",
		"massive_replacement":
		return "replaced_large_delta"
	default:
		return ""
	}
}
