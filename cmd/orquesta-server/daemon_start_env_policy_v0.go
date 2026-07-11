package main

import (
	"sort"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	daemonStartEnvProfileSettingV0    = "ORQUESTA_SERVER_DAEMON_START_ENV_PROFILE"
	daemonStartEnvCategoriesSettingV0 = "ORQUESTA_SERVER_DAEMON_START_ENV_CATEGORIES"
	daemonStartEnvCountsSettingV0     = "ORQUESTA_SERVER_DAEMON_START_ENV_COUNTS"
	daemonStartEnvIssuesSettingV0     = "ORQUESTA_SERVER_DAEMON_START_ENV_ISSUES"
	daemonAutoprogrammingEnvPrefixV0  = "ORQUESTA_" + "AUTOPROGRAMMING_"
)

type daemonStartEnvPolicyV0 struct {
	Profile    string
	Categories []string
	Projected  int
	Explicit   int
	Defaulted  int
	Derived    int
	Issues     []string
}

type daemonStartEnvBuilderV0 struct {
	values     map[string]string
	sources    map[string]string
	categories map[string]bool
	parentKeys map[string]bool
}

func serverDaemonStartEnvironmentV0(parent []string, config orquestaserver.ConfigV0) []string {
	builder := newDaemonStartEnvBuilderV0()
	builder.addParentV0(parent)
	builder.addDerivedV0(config)
	return builder.envV0()
}

func serverDaemonStartEnvPolicyV0(parent []string, config orquestaserver.ConfigV0) daemonStartEnvPolicyV0 {
	builder := newDaemonStartEnvBuilderV0()
	builder.addParentV0(parent)
	builder.addDerivedV0(config)
	return builder.policyV0()
}

func daemonStartEnvSettingsV0(policy daemonStartEnvPolicyV0) []orquestaserver.ServerConfigSettingV0 {
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingV0(daemonStartEnvProfileSettingV0, policy.Profile, "daemon_start_env", "Perfil env daemon", "Perfil allowlist usado por start para el daemon residente."),
		serverConfigSettingV0(daemonStartEnvCategoriesSettingV0, strings.Join(policy.Categories, ","), "daemon_start_env", "Categorias env daemon", "Categorias de entorno proyectadas sin valores crudos."),
		serverConfigSettingV0(daemonStartEnvCountsSettingV0, daemonStartEnvCountsV0(policy), "daemon_start_env", "Conteos env daemon", "Conteos por origen explicit defaulted derived sin valores crudos."),
		serverConfigSettingV0(daemonStartEnvIssuesSettingV0, strings.Join(policy.Issues, ","), "daemon_start_env", "Issues env daemon", "Issues publicos accionables de variables requeridas."),
	}
}

func newDaemonStartEnvBuilderV0() daemonStartEnvBuilderV0 {
	return daemonStartEnvBuilderV0{
		values:     map[string]string{},
		sources:    map[string]string{},
		categories: map[string]bool{},
		parentKeys: map[string]bool{},
	}
}

func (builder daemonStartEnvBuilderV0) addParentV0(parent []string) {
	for _, item := range parent {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		builder.parentKeys[key] = true
		category, allowed := daemonStartEnvCategoryV0(key)
		if !allowed {
			continue
		}
		builder.setV0(key, value, "explicit", category, false)
	}
}

func (builder daemonStartEnvBuilderV0) addDerivedV0(config orquestaserver.ConfigV0) {
	config = orquestaserver.NormalizeConfigV0(config)
	projectConfig := projectConfigFromServerConfigBestEffortV0(config)
	startupCleanup := startupCleanupConfigFromProjectConfigFileV0(projectConfig)
	builder.setConfigV0(envServerAddrV0, config.Addr, "defaulted", "server_bootstrap", true)
	builder.setConfigV0(envServerStateDirV0, config.StateDir, "derived", "server_bootstrap", true)
	builder.setV0(envServerWorktreeV0, serverWorktreeDirFromEnvV0(config.ProjectWorkDir), "derived", "server_bootstrap", false)
	builder.setConfigV0(envServerAuditFileV0, config.AuditFile, "defaulted", "server_bootstrap", false)
	builder.setConfigV0(envServerAutonomyEnabledV0, strconv.FormatBool(config.ResidentDirectorEnabled), "derived", "server", true)
	builder.setConfigV0(envServerResidentDirectorEnabledV0, strconv.FormatBool(config.ResidentDirectorEnabled), "derived", "server", true)
	builder.setConfigV0(envServerResidentDirectorMaxActionsV0, strconv.Itoa(config.ResidentDirectorMaxActions), "defaulted", "server", false)
	builder.setConfigV0(envCodexProjectWorkDirV0, config.ProjectWorkDir, "derived", "codex_runtime", true)
	builder.setConfigV0(envCodexRuntimeWorkDirV0, config.RuntimeWorkDir, "derived", "codex_runtime", true)
	builder.setV0(envCodexCommandV0, codexCommandPathV0(), "derived", "codex_runtime", false)
	builder.setV0(envCodexHomeV0, homeDirV0(), "derived", "codex_runtime", false)
	builder.setV0(envCodexCodeHomeV0, codeHomeDirV0(), "derived", "codex_runtime", false)
	builder.setV0(envCodexPathV0, envOrDefaultV0(envCodexPathV0, getenvRawV0("PATH")), "derived", "codex_runtime", false)
	builder.setConfigV0(envSecurityModeV0, serverSecurityModeEffectiveValueFromProjectConfigFileV0(projectConfig), "defaulted", "rails", false)
	builder.setConfigV0(envRailsModeV0, serverRailsModeEffectiveValueFromProjectConfigFileV0(projectConfig), "defaulted", "rails", false)
	builder.setConfigV0(detailProhibitedRailsEnvV0, serverDetailRailsEffectiveValueFromProjectConfigFileV0(projectConfig), "defaulted", "rails", false)
	builder.setConfigV0(detailProhibitedRailsScopeEnvV0, serverDetailRailsScopeEffectiveValueFromProjectConfigFileV0(projectConfig), "defaulted", "rails", false)
	builder.setConfigV0(envStartupCleanupModeV0, startupCleanup.Mode, "defaulted", "startup", false)
	builder.setConfigV0(envStartupCleanupScopeRefsV0, strings.Join(startupCleanup.ScopeRefs, ","), "defaulted", "startup", false)
	builder.setConfigV0(envStartupQueueLimitV0, strconv.Itoa(startupCleanup.QueueLimit), "defaulted", "startup", false)
	builder.setConfigV0(envServerDaemonLogMaxBytesV0, strconv.FormatInt(config.DaemonLogPolicy.MaxBytes, 10), "defaulted", "daemon_logs", false)
	builder.setConfigV0(envServerDaemonLogMaxRotatedV0, strconv.Itoa(config.DaemonLogPolicy.MaxRotatedFiles), "defaulted", "daemon_logs", false)
	builder.setConfigV0(envServerDaemonLogRetentionDaysV0, strconv.Itoa(config.DaemonLogPolicy.RetentionDays), "defaulted", "daemon_logs", false)
}

func (builder daemonStartEnvBuilderV0) setConfigV0(key string, value string, fallbackSource string, category string, override bool) {
	source := fallbackSource
	if builder.parentKeys[strings.TrimSpace(key)] {
		source = "explicit"
	}
	builder.setV0(key, value, source, category, override)
}

func (builder daemonStartEnvBuilderV0) setV0(key string, value string, source string, category string, override bool) {
	key = strings.TrimSpace(key)
	if key == "" || strings.TrimSpace(value) == "" {
		return
	}
	if _, exists := builder.values[key]; exists && !override {
		return
	}
	builder.values[key] = value
	builder.sources[key] = source
	builder.categories[category] = true
}

func (builder daemonStartEnvBuilderV0) envV0() []string {
	keys := make([]string, 0, len(builder.values))
	for key := range builder.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+builder.values[key])
	}
	return out
}

func (builder daemonStartEnvBuilderV0) policyV0() daemonStartEnvPolicyV0 {
	policy := daemonStartEnvPolicyV0{Profile: "allowlist"}
	for category := range builder.categories {
		policy.Categories = append(policy.Categories, category)
	}
	sort.Strings(policy.Categories)
	for _, source := range builder.sources {
		policy.Projected++
		switch source {
		case "explicit":
			policy.Explicit++
		case "defaulted":
			policy.Defaulted++
		case "derived":
			policy.Derived++
		}
	}
	if strings.TrimSpace(builder.values[envCodexCommandV0]) == "" {
		policy.Issues = append(policy.Issues, "codex_command_unavailable")
	}
	return policy
}

func daemonStartEnvCountsV0(policy daemonStartEnvPolicyV0) string {
	return "projected=" + strconv.Itoa(policy.Projected) +
		",explicit=" + strconv.Itoa(policy.Explicit) +
		",defaulted=" + strconv.Itoa(policy.Defaulted) +
		",derived=" + strconv.Itoa(policy.Derived)
}

func daemonStartEnvCategoryV0(key string) (string, bool) {
	if _, registered := serverEffectiveEnvRegistryV0[key]; !registered {
		return "", false
	}
	if key == envOPESBaseURLLegacyV0 {
		return "opes_bridge", true
	}
	if daemonStartEnvBlockedKeyV0(key) {
		return "", false
	}
	for _, rule := range daemonStartEnvPrefixRulesV0() {
		if strings.HasPrefix(key, rule.prefix) {
			return rule.category, true
		}
	}
	return "", false
}

type daemonStartEnvPrefixRuleV0 struct {
	prefix   string
	category string
}

func daemonStartEnvPrefixRulesV0() []daemonStartEnvPrefixRuleV0 {
	return []daemonStartEnvPrefixRuleV0{
		{"ORQUESTA_SERVER_", "server"},
		{"ORQUESTA_STARTUP_", "startup"},
		{"ORQUESTA_RAILS_", "rails"},
		{"ORQUESTA_DETAIL_", "rails"},
		{"ORQUESTA_CODEX_", "codex_runtime"},
		{daemonAutoprogrammingEnvPrefixV0, "autoprogramming"},
		{"ORQUESTA_CAPACITY_", "capacity"},
		{"ORQUESTA_DIRECTOR_", "director"},
		{"ORQUESTA_OPES_", "opes_bridge"},
		{"ORQUESTA_DOMAIN_WORK_", "domain_work"},
		{"ORQUESTA_EGRESS_", "egress_sanitizer"},
		{"ORQUESTA_REQUIRED_TEST_", "required_tests"},
		{"ORQUESTA_REVIEW_", "review"},
	}
}

func daemonStartEnvBlockedKeyV0(key string) bool {
	if key == envServerControlTokenV0 {
		return false
	}
	if key == envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0 {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case envCodexHomeV0, envCodexCodeHomeV0, envCodexPathV0:
		return true
	}
	upper := strings.ToUpper(strings.TrimSpace(key))
	for _, fragment := range []string{"TOKEN", "SECRET", "PASSWORD", "API_KEY", "AUTHORIZATION", "OAUTH"} {
		if strings.Contains(upper, fragment) {
			return true
		}
	}
	return false
}

func getenvRawV0(key string) string {
	for _, item := range []string{key} {
		if value := strings.TrimSpace(envOrDefaultV0(item, "")); value != "" {
			return value
		}
	}
	return ""
}
