package otelagenttools

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type fetcher func(context.Context, string) (string, error)

var githubTagPattern = regexp.MustCompile(`^v?[0-9]+(\.[0-9]+){1,3}$`)

var httpClient = &http.Client{
	Timeout: 20 * time.Second,
}

var fetchers = map[string]fetcher{
	"crates":    fetchCratesVersion,
	"github":    fetchGitHubVersion,
	"goproxy":   fetchGoProxyVersion,
	"hex":       fetchHexVersion,
	"maven":     fetchMavenVersion,
	"npm":       fetchNPMVersion,
	"nuget":     fetchNuGetVersion,
	"packagist": fetchPackagistVersion,
	"pypi":      fetchPyPIVersion,
	"rubygems":  fetchRubyGemsVersion,
}

func FetchVersion(ctx context.Context, sourceKind, target string) (string, error) {
	fetch, ok := fetchers[sourceKind]
	if !ok {
		return "", fmt.Errorf("unsupported source kind: %s", sourceKind)
	}

	version, err := fetch(ctx, target)
	if err != nil {
		return "", err
	}
	return normalizeVersion(version), nil
}

func fetchJSON(ctx context.Context, rawURL string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "otel-agent-tools/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("request %s: unexpected status %d: %s", rawURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

func fetchJSONDocument(ctx context.Context, rawURL string) (map[string]any, error) {
	var doc map[string]any
	if err := fetchJSON(ctx, rawURL, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func fetchDirectString(ctx context.Context, rawURL string, path ...string) (string, error) {
	doc, err := fetchJSONDocument(ctx, rawURL)
	if err != nil {
		return "", err
	}
	value, err := extractStringAtPath(doc, path...)
	if err != nil {
		return "", fmt.Errorf("%s: %w", rawURL, err)
	}
	return value, nil
}

func fetchNestedString(ctx context.Context, rawURL string, path ...string) (string, error) {
	return fetchDirectString(ctx, rawURL, path...)
}

func fetchLastStableString(ctx context.Context, rawURL string, path ...string) (string, error) {
	doc, err := fetchJSONDocument(ctx, rawURL)
	if err != nil {
		return "", err
	}
	values, err := extractStringSliceAtPath(doc, path...)
	if err != nil {
		return "", fmt.Errorf("%s: %w", rawURL, err)
	}
	for i := len(values) - 1; i >= 0; i-- {
		if !strings.Contains(values[i], "-") {
			return values[i], nil
		}
	}
	return "", fmt.Errorf("%s: no stable version found", rawURL)
}

func fetchFirstStringFromKeyedArray(ctx context.Context, rawURL string, key string, mapPath []string, valuePath ...string) (string, error) {
	doc, err := fetchJSONDocument(ctx, rawURL)
	if err != nil {
		return "", err
	}
	value, err := extractFirstStringFromKeyedArray(doc, key, mapPath, valuePath...)
	if err != nil {
		return "", fmt.Errorf("%s: %w", rawURL, err)
	}
	return value, nil
}

func extractStringAtPath(doc map[string]any, path ...string) (string, error) {
	value, err := extractValueAtPath(doc, path...)
	if err != nil {
		return "", err
	}
	s, ok := value.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("missing string at %s", strings.Join(path, "."))
	}
	return s, nil
}

func extractStringSliceAtPath(doc map[string]any, path ...string) ([]string, error) {
	value, err := extractValueAtPath(doc, path...)
	if err != nil {
		return nil, err
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array at %s", strings.Join(path, "."))
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("expected string array at %s", strings.Join(path, "."))
		}
		result = append(result, s)
	}
	return result, nil
}

func extractFirstStringFromKeyedArray(doc map[string]any, key string, mapPath []string, valuePath ...string) (string, error) {
	value, err := extractValueAtPath(doc, mapPath...)
	if err != nil {
		return "", err
	}
	m, ok := value.(map[string]any)
	if !ok {
		return "", fmt.Errorf("expected object at %s", strings.Join(mapPath, "."))
	}

	itemsValue, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing key %q at %s", key, strings.Join(mapPath, "."))
	}
	items, ok := itemsValue.([]any)
	if !ok {
		return "", fmt.Errorf("expected array for key %q at %s", key, strings.Join(mapPath, "."))
	}
	if len(items) == 0 {
		return "", fmt.Errorf("empty array for key %q at %s", key, strings.Join(mapPath, "."))
	}

	first, ok := items[0].(map[string]any)
	if !ok {
		return "", fmt.Errorf("expected object items for key %q at %s", key, strings.Join(mapPath, "."))
	}
	return extractStringAtPath(first, valuePath...)
}

func extractValueAtPath(doc map[string]any, path ...string) (any, error) {
	var current any = doc
	for _, part := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected object at %s", part)
		}
		next, ok := m[part]
		if !ok {
			return nil, fmt.Errorf("missing field %s", strings.Join(path, "."))
		}
		current = next
	}
	return current, nil
}

func fetchNPMVersion(ctx context.Context, target string) (string, error) {
	u := fmt.Sprintf("https://registry.npmjs.org/%s/latest", url.PathEscape(target))
	return fetchDirectString(ctx, u, "version")
}

func fetchNuGetVersion(ctx context.Context, target string) (string, error) {
	u := fmt.Sprintf("https://api.nuget.org/v3-flatcontainer/%s/index.json", strings.ToLower(target))
	return fetchLastStableString(ctx, u, "versions")
}

func fetchMavenVersion(ctx context.Context, target string) (string, error) {
	parts := strings.SplitN(target, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("maven target must be group:artifact: %s", target)
	}

	groupPath := strings.ReplaceAll(parts[0], ".", "/")
	u := fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/maven-metadata.xml",
		groupPath, url.PathEscape(parts[1]))
	return fetchMavenMetadataRelease(ctx, u)
}

func fetchMavenMetadataRelease(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("User-Agent", "otel-agent-tools/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("request %s: unexpected status %d: %s", rawURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var meta struct {
		Versioning struct {
			Release  string   `xml:"release"`
			Latest   string   `xml:"latest"`
			Versions []string `xml:"versions>version"`
		} `xml:"versioning"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", fmt.Errorf("decode %s: %w", rawURL, err)
	}

	if v := strings.TrimSpace(meta.Versioning.Release); v != "" && !strings.Contains(v, "-") {
		return v, nil
	}
	for i := len(meta.Versioning.Versions) - 1; i >= 0; i-- {
		v := strings.TrimSpace(meta.Versioning.Versions[i])
		if v != "" && !strings.Contains(v, "-") {
			return v, nil
		}
	}
	return "", fmt.Errorf("%s: no stable version found", rawURL)
}

func fetchGitHubVersion(ctx context.Context, target string) (string, error) {
	var releases []struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}

	releasesURL := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20", target)
	if err := fetchJSON(ctx, releasesURL, &releases); err == nil {
		for _, release := range releases {
			if release.Draft || release.Prerelease || release.TagName == "" {
				continue
			}
			return release.TagName, nil
		}
	}

	var tags []struct {
		Name string `json:"name"`
	}
	tagsURL := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=100", target)
	if err := fetchJSON(ctx, tagsURL, &tags); err != nil {
		return "", err
	}

	var versions []string
	for _, tag := range tags {
		if githubTagPattern.MatchString(tag.Name) {
			versions = append(versions, tag.Name)
		}
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("github: no release or semver tag found for %s", target)
	}

	slices.SortFunc(versions, compareVersions)
	return versions[len(versions)-1], nil
}

func fetchGoProxyVersion(ctx context.Context, target string) (string, error) {
	u := fmt.Sprintf("https://proxy.golang.org/%s/@latest", target)
	return fetchDirectString(ctx, u, "Version")
}

func fetchPyPIVersion(ctx context.Context, target string) (string, error) {
	u := fmt.Sprintf("https://pypi.org/pypi/%s/json", target)
	return fetchNestedString(ctx, u, "info", "version")
}

func fetchPackagistVersion(ctx context.Context, target string) (string, error) {
	u := fmt.Sprintf("https://repo.packagist.org/p2/%s.json", target)
	return fetchFirstStringFromKeyedArray(ctx, u, target, []string{"packages"}, "version")
}

func fetchRubyGemsVersion(ctx context.Context, target string) (string, error) {
	u := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", target)
	return fetchDirectString(ctx, u, "version")
}

func fetchCratesVersion(ctx context.Context, target string) (string, error) {
	u := fmt.Sprintf("https://crates.io/api/v1/crates/%s", target)
	return fetchNestedString(ctx, u, "crate", "max_stable_version")
}

func fetchHexVersion(ctx context.Context, target string) (string, error) {
	u := fmt.Sprintf("https://hex.pm/api/packages/%s", target)
	return fetchDirectString(ctx, u, "latest_stable_version")
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func compareVersions(a, b string) int {
	aParts := parseVersionParts(a)
	bParts := parseVersionParts(b)
	for i := 0; i < max(len(aParts), len(bParts)); i++ {
		var aPart, bPart int
		if i < len(aParts) {
			aPart = aParts[i]
		}
		if i < len(bParts) {
			bPart = bParts[i]
		}
		if aPart < bPart {
			return -1
		}
		if aPart > bPart {
			return 1
		}
	}
	return 0
}

func parseVersionParts(version string) []int {
	cleaned := normalizeVersion(version)
	segments := strings.Split(cleaned, ".")
	parts := make([]int, 0, len(segments))
	for _, segment := range segments {
		value, err := strconv.Atoi(segment)
		if err != nil {
			return []int{0}
		}
		parts = append(parts, value)
	}
	return parts
}
