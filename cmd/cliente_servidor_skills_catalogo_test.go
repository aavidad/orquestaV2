package cmd

import "testing"

func TestCommandSupportsServerModeSkillsCatalogo(t *testing.T) {
	casos := []struct {
		args []string
		want bool
	}{
		{args: []string{"skills", "remotas", "--q", "openai"}, want: true},
		{args: []string{"skills", "borrar", "12"}, want: true},
	}
	for _, tc := range casos {
		if got := commandSupportsServerMode(tc.args); got != tc.want {
			t.Fatalf("commandSupportsServerMode(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}
