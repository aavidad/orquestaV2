package main

import "strings"

var autoprogrammingPromotionGuardianConfigFailureCodesV0 = []string{
	"guardian_config_invalid_bool",
	"guardian_config_invalid_duration",
	"guardian_config_invalid_int",
	"guardian_config_invalid_budget",
	"guardian_config_invalid_allowlist",
	"guardian_config_invalid_flag",
	"guardian_config_path_outside_allowed_root",
	"guardian_config_path_symlink_blocked",
	"guardian_config_project_root_invalid",
	"guardian_promotion_lease_busy",
	"guardian_promotion_lease_lost",
}

func autoprogrammingPromotionGuardianFailureCodeV0(output string) string {
	for _, code := range autoprogrammingPromotionGuardianConfigFailureCodesV0 {
		if strings.Contains(output, code) {
			return code
		}
	}
	return ""
}
