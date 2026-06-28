package main

import "testing"

func TestServerOPESAutomationContextFromEnvV0IgnoraBanderasOPESFalseV0(t *testing.T) {
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv(envOPESBaseURLLegacyV0, "")
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envOPESProjectWorkDirV0, "")
	t.Setenv(envOPESBridgeEnabledV0, "false")
	t.Setenv(envOPESRegistryFinalPkgEnabledV0, "false")
	t.Setenv(envOPESTopicRegistryEnabledV0, "false")
	t.Setenv(envOPESTopicRegistryToolPathV0, "")

	if serverOPESAutomationContextFromEnvV0() {
		t.Fatalf("banderas OPES false no deben activar contexto OPES")
	}
}

func TestServerOPESAutomationContextFromEnvV0DetectaBanderasOPESTrueV0(t *testing.T) {
	t.Setenv(envOPESBridgeEnabledV0, "true")

	if !serverOPESAutomationContextFromEnvV0() {
		t.Fatalf("bandera OPES true debe activar contexto OPES")
	}
}
