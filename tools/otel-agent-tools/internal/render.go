package otelagenttools

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"text/template"
)

type VersionEntry struct {
	Language    string
	Package     string
	Version     string
	SourceLabel string
	SourceURL   string
	DocsURL     string
	ExamplesURL string
}

// ComponentEntry is a rendered row of the components table.
type ComponentEntry struct {
	Component  string
	Versions   []string
	ReleaseURL string
	Notes      string
}

type versionIndexView struct {
	LastChanged string
	Rows        []versionIndexRow
	Components  []componentRow
}

type versionIndexRow struct {
	Language     string
	PackageCell  string
	VersionCell  string
	SourceCell   string
	DocsCell     string
	ExamplesCell string
}

type componentRow struct {
	Component   string
	VersionCell string
	ReleaseCell string
	Notes       string
}

const lastChangedPrefix = "Versions last changed: "

var lastChangedPattern = regexp.MustCompile(`(?m)^` + lastChangedPrefix + `(\d{4}-\d{2}-\d{2})\b`)

var versionIndexTemplate = template.Must(template.New("version-index").Parse(`# OpenTelemetry Version Index

` + lastChangedPrefix + `{{ .LastChanged }} (UTC). Each version below was the latest stable release on that date; newer releases may exist.

## SDKs

| Language | Package/Repo | Latest | Release Source | Setup Docs | Examples |
|---|---|---|---|---|---|
{{- range .Rows }}
| {{ .Language }} | {{ .PackageCell }} | {{ .VersionCell }} | {{ .SourceCell }} | {{ .DocsCell }} | {{ .ExamplesCell }} |
{{- end }}
{{- if .Components }}

## Components

| Component | Latest | Release | Notes |
|---|---|---|---|
{{- range .Components }}
| {{ .Component }} | {{ .VersionCell }} | {{ .ReleaseCell }} | {{ .Notes }} |
{{- end }}
{{- end }}
`))

// RenderVersionIndex renders the index stamped with today, unless rendering
// with the date already in previous reproduces previous exactly: then no row
// changed and previous is returned, so an unchanged index keeps its date.
func RenderVersionIndex(previous []byte, sdks []VersionEntry, components []ComponentEntry, today string) ([]byte, error) {
	if m := lastChangedPattern.FindSubmatch(previous); m != nil {
		unchanged, err := renderVersionIndex(sdks, components, string(m[1]))
		if err != nil {
			return nil, err
		}
		if bytes.Equal(unchanged, previous) {
			return previous, nil
		}
	}
	return renderVersionIndex(sdks, components, today)
}

func renderVersionIndex(sdks []VersionEntry, components []ComponentEntry, lastChanged string) ([]byte, error) {
	rows := make([]versionIndexRow, 0, len(sdks))
	for _, entry := range sdks {
		rows = append(rows, versionIndexRow{
			Language:     entry.Language,
			PackageCell:  fmt.Sprintf("`%s`", entry.Package),
			VersionCell:  fmt.Sprintf("`%s`", entry.Version),
			SourceCell:   markdownLink(entry.SourceLabel, entry.SourceURL),
			DocsCell:     markdownLink("Docs", entry.DocsURL),
			ExamplesCell: optionalMarkdownLink("Examples", entry.ExamplesURL),
		})
	}

	slices.SortFunc(rows, func(a, b versionIndexRow) int {
		if cmp := strings.Compare(a.Language, b.Language); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.PackageCell, b.PackageCell)
	})

	componentRows := make([]componentRow, 0, len(components))
	for _, entry := range components {
		versions := make([]string, 0, len(entry.Versions))
		for _, v := range entry.Versions {
			versions = append(versions, fmt.Sprintf("`%s`", v))
		}
		componentRows = append(componentRows, componentRow{
			Component:   entry.Component,
			VersionCell: strings.Join(versions, " / "),
			ReleaseCell: markdownLink("Notes", entry.ReleaseURL),
			Notes:       optionalText(entry.Notes),
		})
	}

	view := versionIndexView{LastChanged: lastChanged, Rows: rows, Components: componentRows}

	var buf bytes.Buffer
	if err := versionIndexTemplate.Execute(&buf, view); err != nil {
		return nil, fmt.Errorf("render version index: %w", err)
	}
	return buf.Bytes(), nil
}

func markdownLink(label, rawURL string) string {
	return fmt.Sprintf("[%s](%s)", label, rawURL)
}

func optionalMarkdownLink(label, rawURL string) string {
	if rawURL == "" {
		return "-"
	}
	return markdownLink(label, rawURL)
}

func optionalText(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
