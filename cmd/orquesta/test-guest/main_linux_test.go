//go:build linux

package main

import "testing"

func TestGuestMainRejectsExecutionOutsidePID1(t *testing.T) {
	if status := run(nil); status != 2 {
		t.Fatalf("status=%d", status)
	}
}
