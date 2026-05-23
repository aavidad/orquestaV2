package orquestamcp

import (
	"strings"
	"testing"
)

func TestMCPAutoprogrammingSuperviseDescriptorV0ExponeModoResidente(t *testing.T) {
	descriptor := MCPAutoprogrammingSuperviseDescriptorV0()
	if !containsMCPStringPartForTestV0(descriptor.Invariantes, "resident_mode") ||
		!containsMCPStringPartForTestV0([]string{descriptor.InputSchema}, "resident_mode") {
		t.Fatalf("descriptor=%+v", descriptor)
	}
}

func containsMCPStringPartForTestV0(values []string, part string) bool {
	for _, value := range values {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}
