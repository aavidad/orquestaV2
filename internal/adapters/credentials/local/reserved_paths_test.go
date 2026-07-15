package local

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestReservedPathsReportExactRecoveryNamespace(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{name: "empty", source: "", want: []string{}},
		{name: "blank", source: "  ", want: []string{}},
		{name: "absolute", source: filepath.Join(string(filepath.Separator), "private", "credentials.json"),
			want: []string{filepath.Join(string(filepath.Separator), "private", "credentials.json") + ".next"}},
		{name: "cleaned", source: filepath.Join("var", "staging", "..", "secrets", "credentials.json") + string(filepath.Separator) + ".",
			want: []string{filepath.Join("var", "secrets", "credentials.json") + ".next"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ReservedPaths(test.source); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ReservedPaths(%q) = %#v, want %#v", test.source, got, test.want)
			}
		})
	}

	first := ReservedPaths("credentials.json")
	first[0] = "mutated"
	if got := ReservedPaths("credentials.json"); !reflect.DeepEqual(got, []string{"credentials.json.next"}) {
		t.Fatalf("reserved path result leaked caller mutation: %#v", got)
	}
}
