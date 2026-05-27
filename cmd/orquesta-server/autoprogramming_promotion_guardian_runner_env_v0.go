package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const autoprogrammingPromotionGuardianRunnerEnvPolicyV0 = "minimal_allowlist"

func defaultAutoprogrammingPromotionGuardianRunnerEnvAllowlistV0() []string {
	if runtime.GOOS == "windows" {
		return []string{"PATH", "TEMP", "TMP", "SystemRoot", "ComSpec", "PATHEXT", "WINDIR"}
	}
	return []string{"PATH", "TMPDIR", "TEMP", "TMP"}
}

func autoprogrammingPromotionGuardianRunnerEnvV0(
	parent []string,
	request serverAutoprogrammingPromotionGuardianRequestV0,
	allowlist []string,
) []string {
	allow := autoprogrammingPromotionGuardianRunnerEnvAllowlistV0(allowlist)
	env := autoprogrammingPromotionGuardianInheritedEnvV0(parent, allow, request)
	env = autoprogrammingPromotionGuardianEnvV0(env, request)
	env = append(env, "ORQUESTA_GUARDIAN_RUNNER_ENV_POLICY="+autoprogrammingPromotionGuardianRunnerEnvPolicyV0)
	env = append(env, autoprogrammingPromotionGuardianRunnerCacheEnvV0(request)...)
	env = autoprogrammingPromotionGuardianRunnerUniqueEnvV0(env)
	sort.Strings(env)
	return env
}

func autoprogrammingPromotionGuardianRunnerEnvAllowlistV0(allowlist []string) map[string]bool {
	if len(allowlist) == 0 {
		allowlist = defaultAutoprogrammingPromotionGuardianRunnerEnvAllowlistV0()
	}
	allow := make(map[string]bool, len(allowlist))
	for _, key := range allowlist {
		key = strings.TrimSpace(key)
		if key != "" {
			allow[key] = true
		}
	}
	return allow
}

func autoprogrammingPromotionGuardianInheritedEnvV0(
	parent []string,
	allow map[string]bool,
	request serverAutoprogrammingPromotionGuardianRequestV0,
) []string {
	env := make([]string, 0, len(allow))
	for _, item := range parent {
		key, _, ok := strings.Cut(item, "=")
		key = strings.TrimSpace(key)
		if ok && allow[key] &&
			!autoprogrammingPromotionGuardianRunnerExplicitEnvV0(key) &&
			autoprogrammingPromotionGuardianRunnerSensitiveEnvAllowedV0(key, request) {
			env = append(env, item)
		}
	}
	return env
}

func autoprogrammingPromotionGuardianRunnerExplicitEnvV0(key string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(key)), "ORQUESTA_GUARDIAN_")
}

func autoprogrammingPromotionGuardianRunnerSensitiveEnvAllowedV0(
	key string,
	request serverAutoprogrammingPromotionGuardianRequestV0,
) bool {
	if !autoprogrammingPromotionGuardianRunnerSensitiveEnvV0(key) {
		return true
	}
	return len(compactStringsV0(append(
		append([]string(nil), request.EvidenceRefs...),
		request.CommandEffectEvidenceRefs...,
	))) > 0
}

func autoprogrammingPromotionGuardianRunnerSensitiveEnvV0(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" {
		return false
	}
	if upper == "HOME" || strings.HasPrefix(upper, "ORQUESTA_CODEX_") ||
		strings.HasPrefix(upper, "GIT_") || strings.HasSuffix(upper, "_PROXY") {
		return true
	}
	for _, fragment := range []string{
		"TOKEN",
		"SECRET",
		"PASSWORD",
		"API_KEY",
		"AUTH",
		"OAUTH",
		"OPENAI",
		"ANTHROPIC",
		"AZURE",
		"GOOGLE",
	} {
		if strings.Contains(upper, fragment) {
			return true
		}
	}
	return false
}

func autoprogrammingPromotionGuardianRunnerUniqueEnvV0(env []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(env))
	for index := len(env) - 1; index >= 0; index-- {
		key, _, ok := strings.Cut(env[index], "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, env[index])
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

func autoprogrammingPromotionGuardianRunnerCacheEnvV0(
	request serverAutoprogrammingPromotionGuardianRequestV0,
) []string {
	stateDir := strings.TrimSpace(request.StateDir)
	if stateDir == "" {
		stateDir = strings.TrimSpace(request.ProjectDir)
	}
	if stateDir == "" {
		return nil
	}
	cacheRoot := filepath.Join(stateDir, "guardian-runner-cache")
	return []string{
		"GOCACHE=" + filepath.Join(cacheRoot, "go-build"),
		"XDG_CACHE_HOME=" + cacheRoot,
	}
}

func autoprogrammingPromotionGuardianRunnerEnvAllowlistFromEnvV0() []string {
	return csvEnvOrDefaultV0(
		envServerAutoprogrammingPromotionGuardianRunnerEnvAllowlistV0,
		defaultAutoprogrammingPromotionGuardianRunnerEnvAllowlistV0(),
	)
}

func autoprogrammingPromotionGuardianRunnerParentEnvV0() []string {
	return os.Environ()
}
