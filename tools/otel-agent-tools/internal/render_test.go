package otelagenttools

import (
	"strings"
	"testing"
)

func TestRenderVersionIndexKeepsDateWhenRowsUnchanged(t *testing.T) {
	sdks := []VersionEntry{{Language: "Go", Package: "go.opentelemetry.io/otel/sdk", Version: "1.0.0", SourceLabel: "Go proxy", SourceURL: "https://example.com/go"}}
	components := []ComponentEntry{{Component: "Weaver", Versions: []string{"0.1.0"}, ReleaseURL: "https://example.com/weaver"}}

	first, err := RenderVersionIndex(nil, sdks, components, "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), lastChangedPrefix+"2026-01-01 ") {
		t.Fatalf("first render missing date:\n%s", first)
	}

	same, err := RenderVersionIndex(first, sdks, components, "2026-02-01")
	if err != nil {
		t.Fatal(err)
	}
	if string(same) != string(first) {
		t.Fatalf("unchanged rows moved the date:\n%s", same)
	}

	components[0].Versions = []string{"0.2.0"}
	bumped, err := RenderVersionIndex(first, sdks, components, "2026-02-01")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bumped), lastChangedPrefix+"2026-02-01 ") || !strings.Contains(string(bumped), "`0.2.0`") {
		t.Fatalf("changed row did not bump the date:\n%s", bumped)
	}
}

func TestRenderVersionIndexComponentsTable(t *testing.T) {
	components := []ComponentEntry{{Component: "Collector core", Versions: []string{"1.69.0", "0.163.0"}, ReleaseURL: "https://example.com/r"}}
	out, err := RenderVersionIndex(nil, nil, components, "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	want := "| Collector core | `1.69.0` / `0.163.0` | [Notes](https://example.com/r) | - |"
	if !strings.Contains(string(out), want) {
		t.Fatalf("missing row %q in:\n%s", want, out)
	}
}
