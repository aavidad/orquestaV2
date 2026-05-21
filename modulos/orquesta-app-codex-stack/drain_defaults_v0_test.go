package orquestaappcodexstack

import "testing"

func TestNormalizeDrainRunRequestV0DaMargenRealALosAgentes(t *testing.T) {
	request := normalizeDrainRunRequestV0(DrainRunRequestV0{})

	if request.MaxExternalWaits != defaultDrainRunMaxExternalWaitsV0 {
		t.Fatalf("max_external_waits=%d want=%d", request.MaxExternalWaits, defaultDrainRunMaxExternalWaitsV0)
	}
	if request.MaxExternalWaits < 120 {
		t.Fatalf("max_external_waits=%d no da margen de minutos", request.MaxExternalWaits)
	}
}
