package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func serverProjectConfigEffectiveValueMatchesEnvV0(config serverProjectConfigFileV0, key string) bool {
	envValue := strings.TrimSpace(os.Getenv(key))
	if envValue == "" {
		return false
	}
	value, ok := serverProjectConfigEffectiveStringValueForEnvKeyV0(config, key)
	return ok && strings.TrimSpace(value) == envValue
}

func serverProjectConfigEffectiveStringValueForEnvKeyV0(config serverProjectConfigFileV0, key string) (string, bool) {
	if value, ok := serverProjectConfigControlPlaneStringValueForEnvKeyV0(config.ControlPlane, key); ok {
		return value, true
	}
	switch key {
	case envStartupCleanupModeV0:
		if configStringPointerHasValueV0(config.StartupCleanup.Mode) {
			return strings.ToLower(strings.TrimSpace(*config.StartupCleanup.Mode)), true
		}
	case envStartupCleanupScopeRefsV0:
		if config.StartupCleanup.ScopeRefs != nil && len(*config.StartupCleanup.ScopeRefs) > 0 {
			return strings.Join(compactServerStringsV0(*config.StartupCleanup.ScopeRefs), ","), true
		}
	case envStartupQueueLimitV0:
		if configIntPointerPositiveV0(config.StartupCleanup.QueueLimit) {
			return strconv.Itoa(*config.StartupCleanup.QueueLimit), true
		}
	case envSecurityModeV0:
		if configStringPointerHasValueV0(config.RailsSecurity.SecurityMode) {
			return serverSecurityModeEffectiveValueFromProjectConfigFileV0(config), true
		}
	case envRailsModeV0:
		if configStringPointerHasValueV0(config.RailsSecurity.RailsMode) {
			return serverRailsModeEffectiveValueFromProjectConfigFileV0(config), true
		}
	case envDetailProhibitedRailsV0:
		if configStringPointerHasValueV0(config.RailsSecurity.DetailProhibitedRails) {
			return serverDetailRailsEffectiveValueFromProjectConfigFileV0(config), true
		}
	case envDetailProhibitedRailsScopeV0:
		if configStringPointerHasValueV0(config.RailsSecurity.DetailProhibitedRailsScope) {
			return serverDetailRailsScopeEffectiveValueFromProjectConfigFileV0(config), true
		}
	}
	return "", false
}

func serverProjectConfigControlPlaneStringValueForEnvKeyV0(config serverProjectConfigControlPlaneV0, key string) (string, bool) {
	switch key {
	case envServerRemoteControlPlaneConfirmV0:
		if config.RemoteAccessOptIn != nil {
			return fmt.Sprintf("%t", *config.RemoteAccessOptIn), true
		}
	case envServerControlTokenV0:
		if configStringPointerHasValueV0(config.Token) {
			return strings.TrimSpace(*config.Token), true
		}
	case envServerControlPrincipalV0:
		if configStringPointerHasValueV0(config.Principal) {
			return strings.TrimSpace(*config.Principal), true
		}
	case envServerControlPermissionRefV0:
		if configStringPointerHasValueV0(config.PermissionRef) {
			return strings.TrimSpace(*config.PermissionRef), true
		}
	case envServerControlPublicReasonV0:
		if configStringPointerHasValueV0(config.PublicReason) {
			return strings.TrimSpace(*config.PublicReason), true
		}
	}
	return "", false
}
