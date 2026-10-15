package otelagenttools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type SDKSource struct {
	Language    string `json:"language"`
	Package     string `json:"package"`
	SourceKind  string `json:"source_kind"`
	SourceLabel string `json:"source_label"`
	SourceURL   string `json:"source_url"`
	Target      string `json:"target"`
	DocsURL     string `json:"docs_url"`
	ExamplesURL string `json:"examples_url"`
}

func LoadSources(path string) ([]SDKSource, error) {
	var sources []SDKSource
	if err := loadJSON(path, &sources); err != nil {
		return nil, err
	}

	seenKeys := map[string]struct{}{}
	for i := range sources {
		sources[i].Language = strings.TrimSpace(sources[i].Language)
		sources[i].Package = strings.TrimSpace(sources[i].Package)
		sources[i].SourceKind = strings.TrimSpace(sources[i].SourceKind)
		sources[i].SourceLabel = strings.TrimSpace(sources[i].SourceLabel)
		sources[i].SourceURL = strings.TrimSpace(sources[i].SourceURL)
		sources[i].Target = strings.TrimSpace(sources[i].Target)
		sources[i].DocsURL = strings.TrimSpace(sources[i].DocsURL)
		sources[i].ExamplesURL = strings.TrimSpace(sources[i].ExamplesURL)

		if sources[i].Language == "" || sources[i].Package == "" || sources[i].SourceKind == "" || sources[i].SourceLabel == "" || sources[i].SourceURL == "" || sources[i].Target == "" {
			return nil, fmt.Errorf("source %d has empty required fields", i)
		}
		key := strings.Join([]string{
			sources[i].Language,
			sources[i].Package,
			sources[i].SourceKind,
			sources[i].Target,
		}, "\x00")
		if _, exists := seenKeys[key]; exists {
			return nil, fmt.Errorf("duplicate source entry for language=%s package=%s target=%s", sources[i].Language, sources[i].Package, sources[i].Target)
		}
		seenKeys[key] = struct{}{}
	}

	return sources, nil
}

// ComponentSource is a GitHub-released OpenTelemetry component listed in the
// components table of the version index.
type ComponentSource struct {
	Component string `json:"component"`
	Repo      string `json:"repo"`
	TagPrefix string `json:"tag_prefix"`
	// ReleaseNameVersions reads the version list from a release name such as
	// "v1.69.0/v0.163.0" instead of from the tag.
	ReleaseNameVersions bool   `json:"release_name_versions"`
	Notes               string `json:"notes"`
}

// LoadComponentSources reads and validates component-sources.json.
func LoadComponentSources(path string) ([]ComponentSource, error) {
	var sources []ComponentSource
	if err := loadJSON(path, &sources); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for i, source := range sources {
		if source.Component == "" || source.Repo == "" {
			return nil, fmt.Errorf("component source %d has empty required fields", i)
		}
		if _, exists := seen[source.Component]; exists {
			return nil, fmt.Errorf("duplicate component source %q", source.Component)
		}
		seen[source.Component] = struct{}{}
	}
	return sources, nil
}

// PinSource names a value that an upstream release pins in a source file and
// that a reference document records, so the two can be compared.
type PinSource struct {
	Name      string `json:"name"`
	Repo      string `json:"repo"`
	TagPrefix string `json:"tag_prefix"`
	Path      string `json:"path"`
	// Pattern captures the pinned value in group 1 from the upstream file;
	// DocPattern does the same in the document and defaults to Pattern.
	Pattern    string `json:"pattern"`
	DocPattern string `json:"doc_pattern"`
}

// LoadPinSources reads and validates a pin-source JSON file.
func LoadPinSources(path string) ([]PinSource, error) {
	var sources []PinSource
	if err := loadJSON(path, &sources); err != nil {
		return nil, err
	}
	for i, source := range sources {
		if source.Name == "" || source.Repo == "" || source.Path == "" || source.Pattern == "" {
			return nil, fmt.Errorf("pin source %d has empty required fields", i)
		}
		if source.DocPattern == "" {
			sources[i].DocPattern = source.Pattern
		}
	}
	return sources, nil
}

func loadJSON[T any](path string, dst *[]T) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if len(*dst) == 0 {
		return fmt.Errorf("%s: no entries configured", path)
	}
	return nil
}
