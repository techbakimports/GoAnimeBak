package gobridge

import (
	"encoding/json"
	"testing"
)

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Fatal("Version() returned empty string")
	}
	if v != "gobridge/1.0.0" {
		t.Fatalf("unexpected version: %s", v)
	}
}

func TestGetSources(t *testing.T) {
	result := GetSources()
	if result == "" {
		t.Fatal("GetSources() returned empty string")
	}

	var sources []SourceResult
	if err := json.Unmarshal([]byte(result), &sources); err != nil {
		t.Fatalf("GetSources() returned invalid JSON: %v", err)
	}

	if len(sources) == 0 {
		t.Fatal("GetSources() returned empty array")
	}

	// Verify known sources exist
	found := map[string]bool{}
	for _, s := range sources {
		found[s.ID] = true
		if s.Name == "" {
			t.Errorf("source has empty name: %+v", s)
		}
	}

	if !found["AllAnime"] {
		t.Error("AllAnime source not found")
	}
	if !found["AnimeFire"] {
		t.Error("AnimeFire source not found")
	}
}

func TestSearchAnime_EmptyQuery(t *testing.T) {
	_, err := SearchAnime("", "")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSearchAnime_InvalidSource(t *testing.T) {
	_, err := SearchAnime("naruto", "InvalidSource123")
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
}

func TestGetEpisodes_EmptyURL(t *testing.T) {
	_, err := GetEpisodes("", "AllAnime")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestGetEpisodes_EmptySource(t *testing.T) {
	_, err := GetEpisodes("some-url", "")
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestGetStreamURL_EmptyInputs(t *testing.T) {
	_, err := GetStreamURL("", "", "", "")
	if err == nil {
		t.Fatal("expected error for empty inputs")
	}
}

func TestGetStreamURL_InvalidAnimeJSON(t *testing.T) {
	_, err := GetStreamURL("not-json", `{"number":"1","url":"x"}`, "best", "sub")
	if err == nil {
		t.Fatal("expected error for invalid anime JSON")
	}
}

func TestGetStreamURL_InvalidEpisodeJSON(t *testing.T) {
	_, err := GetStreamURL(`{"url":"x","source":"AllAnime"}`, "not-json", "best", "sub")
	if err == nil {
		t.Fatal("expected error for invalid episode JSON")
	}
}

// Integration tests — require network. Skip with -short.
func TestSearchAnime_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	result, err := SearchAnime("naruto", "")
	if err != nil {
		t.Fatalf("SearchAnime failed: %v", err)
	}

	var animes []AnimeResult
	if err := json.Unmarshal([]byte(result), &animes); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if len(animes) == 0 {
		t.Fatal("no results for 'naruto'")
	}

	// Verify structure
	for _, a := range animes {
		if a.Name == "" {
			t.Error("anime has empty name")
		}
		if a.URL == "" {
			t.Error("anime has empty URL")
		}
		if a.Source == "" {
			t.Error("anime has empty source")
		}
	}

	t.Logf("found %d results for 'naruto'", len(animes))
}
