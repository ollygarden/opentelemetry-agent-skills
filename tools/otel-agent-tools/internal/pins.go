package otelagenttools

import (
	"context"
	"fmt"
	"regexp"
)

// PinResult compares a value pinned at an upstream release with the value a
// reference document records for it.
type PinResult struct {
	Name     string
	Tag      string
	Upstream string
	Recorded string
}

// Drifted reports whether the document no longer matches the release.
func (r PinResult) Drifted() bool {
	return r.Upstream != r.Recorded
}

// CheckPin reads the pinned value from source.Path at the latest release of
// source.Repo and the recorded value from doc.
func CheckPin(ctx context.Context, source PinSource, doc string) (PinResult, error) {
	recorded, err := captureFirst(source.DocPattern, doc)
	if err != nil {
		return PinResult{}, fmt.Errorf("%s: document: %w", source.Name, err)
	}
	release, err := LatestGitHubRelease(ctx, source.Repo, source.TagPrefix)
	if err != nil {
		return PinResult{}, fmt.Errorf("%s: %w", source.Name, err)
	}
	content, err := FetchGitHubFile(ctx, source.Repo, release.Tag, source.Path)
	if err != nil {
		return PinResult{}, fmt.Errorf("%s: %w", source.Name, err)
	}
	upstream, err := captureFirst(source.Pattern, content)
	if err != nil {
		return PinResult{}, fmt.Errorf("%s: %s@%s:%s: %w", source.Name, source.Repo, release.Tag, source.Path, err)
	}
	return PinResult{Name: source.Name, Tag: release.Tag, Upstream: upstream, Recorded: recorded}, nil
}

func captureFirst(pattern, text string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("compile %q: %w", pattern, err)
	}
	if re.NumSubexp() < 1 {
		return "", fmt.Errorf("pattern %q has no capture group", pattern)
	}
	m := re.FindStringSubmatch(text)
	if m == nil {
		return "", fmt.Errorf("pattern %q not found", pattern)
	}
	return m[1], nil
}
