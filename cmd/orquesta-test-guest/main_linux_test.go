//go:build linux

package main

import "testing"

func TestScratchMountContractAllowsNobodyTraversalAndCapsRAM(t *testing.T) {
	if scratchMode != 0o711 || scratchOptions != "mode=0711,size=75%" {
		t.Fatalf("mode=%#o options=%q", scratchMode, scratchOptions)
	}
}
