package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	otelagenttools "otel-agent-tools/internal"
)

const (
	defaultConfigFilename     = "sdk-sources.json"
	defaultComponentsFilename = "component-sources.json"
	defaultPinsFilename       = "declarative-config-pins.json"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing command")
	}

	switch args[0] {
	case "reconcile-versions":
		return runReconcileVersions(args[1:])
	case "reconcile":
		if len(args) < 2 {
			return fmt.Errorf("missing reconcile target")
		}
		switch args[1] {
		case "versions":
			return runReconcileVersions(args[2:])
		default:
			return fmt.Errorf("unknown reconcile target: %s", args[1])
		}
	case "check":
		if len(args) < 2 || args[1] != "pins" {
			return fmt.Errorf("usage: check pins --doc path [--config path]")
		}
		return runCheckPins(args[2:])
	case "-h", "--help", "help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runReconcileVersions(args []string) error {
	fs := flag.NewFlagSet("reconcile-versions", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	configPath := fs.String("config", resolveConfigPath(defaultConfigFilename), "Path to sdk-sources.json")
	componentsPath := fs.String("components", resolveConfigPath(defaultComponentsFilename), "Path to component-sources.json")
	outputPath := fs.String("out", "", "Path to generated markdown index")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outputPath == "" {
		return fmt.Errorf("missing required --out path")
	}

	sources, err := otelagenttools.LoadSources(*configPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	entries := make([]otelagenttools.VersionEntry, 0, len(sources))
	for _, source := range sources {
		version, err := otelagenttools.FetchVersion(ctx, source.SourceKind, source.Target)
		if err != nil {
			return fmt.Errorf("%s: %w", source.Language, err)
		}
		entries = append(entries, otelagenttools.VersionEntry{
			Language:    source.Language,
			Package:     source.Package,
			Version:     version,
			SourceLabel: source.SourceLabel,
			SourceURL:   source.SourceURL,
			DocsURL:     source.DocsURL,
			ExamplesURL: source.ExamplesURL,
		})
	}

	componentSources, err := otelagenttools.LoadComponentSources(*componentsPath)
	if err != nil {
		return err
	}
	components := make([]otelagenttools.ComponentEntry, 0, len(componentSources))
	for _, source := range componentSources {
		component, err := otelagenttools.FetchComponent(ctx, source)
		if err != nil {
			return fmt.Errorf("%s: %w", source.Component, err)
		}
		components = append(components, component)
	}

	previous, err := os.ReadFile(*outputPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read output: %w", err)
	}
	today := time.Now().UTC().Format(time.DateOnly)
	rendered, err := otelagenttools.RenderVersionIndex(previous, entries, components, today)
	if err != nil {
		return err
	}

	unchanged, err := writeIfChanged(*outputPath, rendered)
	if err != nil {
		return err
	}
	if unchanged {
		fmt.Printf("up to date: %s\n", *outputPath)
		return nil
	}

	fmt.Printf("wrote: %s\n", *outputPath)
	return nil
}

func writeIfChanged(path string, next []byte) (bool, error) {
	current, err := os.ReadFile(path)
	if err == nil && bytes.Equal(current, next) {
		return true, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read output: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(path, next, 0o644); err != nil {
		return false, fmt.Errorf("write output: %w", err)
	}
	return false, nil
}

func runCheckPins(args []string) error {
	fs := flag.NewFlagSet("check-pins", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	configPath := fs.String("config", resolveConfigPath(defaultPinsFilename), "Path to declarative-config-pins.json")
	docPath := fs.String("doc", "", "Path to the document that records the pins")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *docPath == "" {
		return fmt.Errorf("missing required --doc path")
	}

	sources, err := otelagenttools.LoadPinSources(*configPath)
	if err != nil {
		return err
	}
	doc, err := os.ReadFile(*docPath)
	if err != nil {
		return fmt.Errorf("read doc: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	drifted := 0
	for _, source := range sources {
		result, err := otelagenttools.CheckPin(ctx, source, string(doc))
		if err != nil {
			return err
		}
		if result.Drifted() {
			drifted++
			fmt.Printf("::warning file=%s::%s: %s pins %q, the record says %q\n", *docPath, result.Name, result.Tag, result.Upstream, result.Recorded)
			continue
		}
		fmt.Printf("ok: %s: %s pins %q\n", result.Name, result.Tag, result.Upstream)
	}
	if drifted > 0 {
		return fmt.Errorf("%d pin(s) drifted; re-verify %s", drifted, *docPath)
	}
	return nil
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  otel-agent-tools reconcile versions --out path [--config path] [--components path]")
	fmt.Println("  otel-agent-tools reconcile-versions --out path [--config path] [--components path]")
	fmt.Println("  otel-agent-tools check pins --doc path [--config path]")
}

func resolveConfigPath(filename string) string {
	candidates := []string{
		filename,
		filepath.Join("tools", "otel-agent-tools", filename),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filename
}
