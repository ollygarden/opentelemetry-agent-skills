package otelagenttools

import (
	"context"
	"strings"
	"testing"
)

func useGitHubServer(t *testing.T, routes map[string]string) {
	t.Helper()
	server := newJSONServer(routes)
	t.Cleanup(server.Close)
	api, raw := githubAPIBase, githubRawBase
	githubAPIBase, githubRawBase = server.URL, server.URL+"/raw"
	t.Cleanup(func() { githubAPIBase, githubRawBase = api, raw })
}

func TestLatestGitHubReleaseFiltersByTagPrefix(t *testing.T) {
	useGitHubServer(t, map[string]string{
		"/repos/o/r/releases": `[
			{"tag_name":"cmd/opampsupervisor/v0.3.0","name":"x"},
			{"tag_name":"v0.3.0-rc.1","name":"rc","prerelease":true},
			{"tag_name":"cmd/builder/v0.2.0","name":"cmd/builder/v0.2.0"},
			{"tag_name":"v0.2.0","name":"v1.8.0/v0.2.0"}
		]`,
	})

	release, err := LatestGitHubRelease(context.Background(), "o/r", "")
	if err != nil || release.Tag != "v0.2.0" {
		t.Fatalf("got %+v, %v; want tag v0.2.0", release, err)
	}
	builder, err := FetchComponent(context.Background(), ComponentSource{Component: "OCB", Repo: "o/r", TagPrefix: "cmd/builder/"})
	if err != nil || len(builder.Versions) != 1 || builder.Versions[0] != "0.2.0" {
		t.Fatalf("got %+v, %v; want version 0.2.0", builder, err)
	}
	if builder.ReleaseURL != "https://github.com/o/r/releases/tag/cmd/builder/v0.2.0" {
		t.Fatalf("release URL %q", builder.ReleaseURL)
	}
}

func TestLatestGitHubReleasePicksHighestVersion(t *testing.T) {
	useGitHubServer(t, map[string]string{
		"/repos/o/r/releases": `[{"tag_name":"v1.41.1","name":"backport"},{"tag_name":"v1.42.0","name":"v1.42.0"}]`,
	})
	release, err := LatestGitHubRelease(context.Background(), "o/r", "")
	if err != nil || release.Tag != "v1.42.0" {
		t.Fatalf("got %+v, %v; want tag v1.42.0", release, err)
	}
}

func TestLatestGitHubReleaseKeepsBothErrors(t *testing.T) {
	useGitHubServer(t, map[string]string{})
	_, err := LatestGitHubRelease(context.Background(), "o/r", "")
	if err == nil || !strings.Contains(err.Error(), "/releases") || !strings.Contains(err.Error(), "/tags") {
		t.Fatalf("want both request errors, got %v", err)
	}
}

func TestLatestGitHubReleaseFallsBackToTags(t *testing.T) {
	useGitHubServer(t, map[string]string{
		"/repos/o/r/releases": `[]`,
		"/repos/o/r/tags":     `[{"name":"v1.9.0"},{"name":"v1.10.0"},{"name":"other"}]`,
	})
	release, err := LatestGitHubRelease(context.Background(), "o/r", "")
	if err != nil || release.Tag != "v1.10.0" {
		t.Fatalf("got %+v, %v; want tag v1.10.0", release, err)
	}
}

func TestFetchComponentReleaseNameVersions(t *testing.T) {
	useGitHubServer(t, map[string]string{
		"/repos/o/core/releases":  `[{"tag_name":"v0.163.0","name":"v1.69.0/v0.163.0"}]`,
		"/repos/o/other/releases": `[{"tag_name":"v0.28.0","name":"0.28.0 - 2026-10-14"}]`,
	})

	core, err := FetchComponent(context.Background(), ComponentSource{Component: "core", Repo: "o/core", ReleaseNameVersions: true})
	if err != nil || len(core.Versions) != 2 || core.Versions[0] != "1.69.0" || core.Versions[1] != "0.163.0" {
		t.Fatalf("got %+v, %v", core, err)
	}
	other, err := FetchComponent(context.Background(), ComponentSource{Component: "other", Repo: "o/other", ReleaseNameVersions: true})
	if err != nil || len(other.Versions) != 1 || other.Versions[0] != "0.28.0" {
		t.Fatalf("non-version release name should fall back to tag: %+v, %v", other, err)
	}
}

func TestCheckPin(t *testing.T) {
	useGitHubServer(t, map[string]string{
		"/repos/o/r/releases":              `[{"tag_name":"v1.2.0"}]`,
		"/raw/o/r/v1.2.0/build.gradle.kts": `val configurationTag = "1.3.0"`,
	})
	source := PinSource{Name: "pin", Repo: "o/r", Path: "build.gradle.kts", Pattern: `configurationTag = "([^"]+)"`, DocPattern: `configurationTag = "([^"]+)"`}

	result, err := CheckPin(context.Background(), source, "`configurationTag = \"1.2.0\"`")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Drifted() || result.Upstream != "1.3.0" || result.Recorded != "1.2.0" || result.Tag != "v1.2.0" {
		t.Fatalf("got %+v", result)
	}

	result, err = CheckPin(context.Background(), source, "`configurationTag = \"1.3.0\"`")
	if err != nil || result.Drifted() {
		t.Fatalf("got %+v, %v; want no drift", result, err)
	}

	if _, err := CheckPin(context.Background(), source, "no pin here"); err == nil {
		t.Fatal("want error when the document does not record the pin")
	}
}
