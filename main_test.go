package main

import (
	"context"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// Production runs on a minimal alpine image with no /usr/share/zoneinfo, so the binary must embed
// the zone database. time.LoadLocation cannot prove that hermetically (it always prefers system
// sources and only then falls back to the embedded copy), so assert the dependency instead.
func TestMainEmbedsTimeZoneDatabase(t *testing.T) {
	out, err := exec.CommandContext(context.Background(), "go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	if slices.Contains(strings.Fields(string(out)), "time/tzdata") {
		return
	}
	t.Fatal("main must import time/tzdata so IANA zones resolve on images without system zoneinfo")
}
