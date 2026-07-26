package catalog

import "testing"

func TestBuiltInV1SemanticDigestGolden(t *testing.T) {
	const want = "71aad1a4be1b2684355e78084adfaa3b7b38d6a9af7b7e1c181cdb2932714606"
	if got := SemanticDigest(BuiltInV1()); got != want {
		t.Fatalf("BuiltInV1 semantic digest=%q want=%q", got, want)
	}
}
