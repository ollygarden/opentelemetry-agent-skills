package otelagenttools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchDirectString(t *testing.T) {
	server := newJSONServer(map[string]string{
		"/npm": `{"version":"1.2.3"}`,
	})
	defer server.Close()

	version, err := fetchDirectString(context.Background(), server.URL+"/npm", "version")
	if err != nil {
		t.Fatalf("fetchDirectString returned error: %v", err)
	}
	if version != "1.2.3" {
		t.Fatalf("fetchDirectString returned %q, want %q", version, "1.2.3")
	}
}

func TestFetchNestedString(t *testing.T) {
	server := newJSONServer(map[string]string{
		"/pypi": `{"info":{"version":"2.3.4"}}`,
	})
	defer server.Close()

	version, err := fetchNestedString(context.Background(), server.URL+"/pypi", "info", "version")
	if err != nil {
		t.Fatalf("fetchNestedString returned error: %v", err)
	}
	if version != "2.3.4" {
		t.Fatalf("fetchNestedString returned %q, want %q", version, "2.3.4")
	}
}

func TestFetchFirstStringFromKeyedArray(t *testing.T) {
	server := newJSONServer(map[string]string{
		"/packagist": `{"packages":{"demo/pkg":[{"version":"3.4.5"}]}}`,
	})
	defer server.Close()

	version, err := fetchFirstStringFromKeyedArray(context.Background(), server.URL+"/packagist", "demo/pkg", []string{"packages"}, "version")
	if err != nil {
		t.Fatalf("fetchFirstStringFromKeyedArray returned error: %v", err)
	}
	if version != "3.4.5" {
		t.Fatalf("fetchFirstStringFromKeyedArray returned %q, want %q", version, "3.4.5")
	}
}

func TestFetchLastStableString(t *testing.T) {
	server := newJSONServer(map[string]string{
		"/nuget": `{"versions":["1.0.0-beta.1","1.0.0","1.1.0-rc.1","1.1.0"]}`,
	})
	defer server.Close()

	version, err := fetchLastStableString(context.Background(), server.URL+"/nuget", "versions")
	if err != nil {
		t.Fatalf("fetchLastStableString returned error: %v", err)
	}
	if version != "1.1.0" {
		t.Fatalf("fetchLastStableString returned %q, want %q", version, "1.1.0")
	}
}

func TestFetchMavenMetadataRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <versioning>
    <latest>1.61.0</latest>
    <release>1.61.0</release>
    <versions>
      <version>1.59.0</version>
      <version>1.60.0</version>
      <version>1.61.0</version>
    </versions>
  </versioning>
</metadata>`))
	}))
	defer server.Close()

	version, err := fetchMavenMetadataRelease(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetchMavenMetadataRelease returned error: %v", err)
	}
	if version != "1.61.0" {
		t.Fatalf("fetchMavenMetadataRelease returned %q, want %q", version, "1.61.0")
	}
}

func TestNormalizeVersion(t *testing.T) {
	if got := normalizeVersion(" v1.2.3 "); got != "1.2.3" {
		t.Fatalf("normalizeVersion returned %q, want %q", got, "1.2.3")
	}
}

func TestFetchVersionUnsupportedSource(t *testing.T) {
	_, err := FetchVersion(context.Background(), "does-not-exist", "demo")
	if err == nil {
		t.Fatal("FetchVersion returned nil error for unsupported source")
	}
	if err.Error() != "unsupported source kind: does-not-exist" {
		t.Fatalf("FetchVersion returned %q", err.Error())
	}
}

func newJSONServer(routes map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
}
