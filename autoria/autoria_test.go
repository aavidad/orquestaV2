package autoria

import "testing"

func TestMarkdownNotice(t *testing.T) {
	if got := MarkdownNotice("es"); got != "> "+AttributionES {
		t.Fatalf("notice es=%q", got)
	}
	if got := MarkdownNotice("en"); got != "> "+AttributionEN {
		t.Fatalf("notice en=%q", got)
	}
}

func TestCodeHeaderLine(t *testing.T) {
	if got := CodeHeaderLine("//", "es"); got != "// "+AttributionES {
		t.Fatalf("header=%q", got)
	}
	if got := CodeHeaderLine("#", "en"); got != "# "+AttributionEN {
		t.Fatalf("header=%q", got)
	}
}
