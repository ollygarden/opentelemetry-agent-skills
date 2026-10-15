package otelagenttools

import (
	"context"
	"fmt"
	"strings"
)

// FetchComponent resolves the latest stable release of a component.
func FetchComponent(ctx context.Context, source ComponentSource) (ComponentEntry, error) {
	release, err := LatestGitHubRelease(ctx, source.Repo, source.TagPrefix)
	if err != nil {
		return ComponentEntry{}, err
	}
	versions := []string{normalizeVersion(strings.TrimPrefix(release.Tag, source.TagPrefix))}
	if source.ReleaseNameVersions {
		if fromName := releaseNameVersions(release.Name); fromName != nil {
			versions = fromName
		}
	}
	return ComponentEntry{
		Component:  source.Component,
		Versions:   versions,
		ReleaseURL: fmt.Sprintf("https://github.com/%s/releases/tag/%s", source.Repo, release.Tag),
		Notes:      source.Notes,
	}, nil
}

// releaseNameVersions splits a release name such as "v1.69.0/v0.163.0" into
// its versions, or returns nil when the name is not such a list.
func releaseNameVersions(name string) []string {
	parts := strings.Split(name, "/")
	versions := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !githubTagPattern.MatchString(part) {
			return nil
		}
		versions = append(versions, normalizeVersion(part))
	}
	return versions
}
