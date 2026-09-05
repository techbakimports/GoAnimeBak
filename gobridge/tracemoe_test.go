package gobridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// withMockTraceMoeServer points traceMoeSearchURL at a mock server for the
// duration of the test and restores it afterward.
func withMockTraceMoeServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	original := traceMoeSearchURL
	traceMoeSearchURL = srv.URL
	t.Cleanup(func() { traceMoeSearchURL = original })
}

func TestIdentifyAnimeFromImage_EmptyImage(t *testing.T) {
	if _, err := IdentifyAnimeFromImage(nil); err == nil {
		t.Fatal("expected an error for an empty image")
	}
	if _, err := IdentifyAnimeFromImage([]byte{}); err == nil {
		t.Fatal("expected an error for an empty image")
	}
}

func TestIdentifyAnimeFromImage_Success(t *testing.T) {
	withMockTraceMoeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"frameCount": 10,
			"error": "",
			"result": [
				{
					"anilist": {"id": 21, "idMal": 21, "title": {"romaji": "One Piece", "english": "One Piece"}},
					"episode": 1050,
					"from": 12.5,
					"to": 14,
					"similarity": 0.97,
					"video": "https://media.trace.moe/video/1",
					"image": "https://media.trace.moe/image/1"
				}
			]
		}`))
	})

	raw, err := IdentifyAnimeFromImage([]byte("fake-image-bytes"))
	if err != nil {
		t.Fatalf("IdentifyAnimeFromImage failed: %v", err)
	}

	var results []TraceMoeResult
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatalf("failed to unmarshal result JSON: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].AnilistID != 21 || results[0].TitleEnglish != "One Piece" || results[0].Episode != "1050" {
		t.Errorf("unexpected result: %+v", results[0])
	}
}

func TestIdentifyAnimeFromImage_NoMatchIsStillValidJSON(t *testing.T) {
	withMockTraceMoeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"frameCount": 0, "error": "", "result": []}`))
	})

	raw, err := IdentifyAnimeFromImage([]byte("fake-image-bytes"))
	if err != nil {
		t.Fatalf("IdentifyAnimeFromImage failed: %v", err)
	}

	var results []TraceMoeResult
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatalf("failed to unmarshal result JSON: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestIdentifyAnimeFromImage_QuotaExceeded(t *testing.T) {
	withMockTraceMoeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": "rate limited"}`))
	})

	if _, err := IdentifyAnimeFromImage([]byte("fake-image-bytes")); err == nil {
		t.Fatal("expected a quota error")
	}
}

func TestFormatTraceMoeEpisode(t *testing.T) {
	cases := map[string]string{
		`1050`:  "1050",
		`null`:  "",
		`"OVA"`: "OVA",
		``:      "",
	}
	for in, want := range cases {
		got := formatTraceMoeEpisode(json.RawMessage(in))
		if got != want {
			t.Errorf("formatTraceMoeEpisode(%q) = %q, want %q", in, got, want)
		}
	}
}
