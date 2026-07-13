package orquestapresentationextractionopenxml

import "testing"

func TestDefaultLimitsV0MatchPresentationSecurityContractV0(t *testing.T) {
	limits := defaultLimitsV0(ConfigV0{})
	if limits.archive != 32<<20 || limits.entries != 256 || limits.xml != 4<<20 || limits.expanded != 64<<20 || limits.slides != 200 || limits.text != 4<<20 || limits.depth != 64 {
		t.Fatalf("unexpected security limits: %#v", limits)
	}
}
