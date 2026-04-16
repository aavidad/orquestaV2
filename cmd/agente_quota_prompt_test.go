package cmd

import "testing"

func TestDebeAutoPausarPorAgotamientoDetectaApproachingRateLimits(t *testing.T) {
	if !debeAutoPausarPorAgotamiento(20, "Approaching rate limits: Switch to gpt-5.1-codex-mini") {
		t.Fatal("deberia auto-pausar ante prompt de approaching rate limits")
	}
}
