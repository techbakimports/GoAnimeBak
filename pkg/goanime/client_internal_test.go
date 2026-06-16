package goanime

import (
	"testing"

	"github.com/alvarorichard/Goanime/pkg/goanime/types"
	"github.com/stretchr/testify/assert"
)

func TestDefaultStreamOptions(t *testing.T) {
	opts := DefaultStreamOptions()
	assert.Equal(t, "best", opts.Quality)
	assert.Equal(t, "sub", opts.Mode)
}

func TestGetAvailableSources_AllSix(t *testing.T) {
	client := NewClient()
	sources := client.GetAvailableSources()

	assert.Len(t, sources, 6)
	assert.Contains(t, sources, types.SourceAllAnime)
	assert.Contains(t, sources, types.SourceAnimeFire)
	assert.Contains(t, sources, types.SourceGoyabu)
	assert.Contains(t, sources, types.SourceGogoAnime)
	assert.Contains(t, sources, types.SourceAnimesOnlineCC)
	assert.Contains(t, sources, types.SourceAnimeHeaven)
}

func TestIsPlayableStreamURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"direct mp4", "https://cdn.example.com/video.mp4", true},
		{"hls master playlist", "https://cdn.example.com/master.m3u8", true},
		{"webm stream", "https://cdn.example.com/video.webm", true},
		{"googlevideo CDN", "https://r1---sn-abc.googlevideo.com/videoplayback?id=1", true},
		{"unresolved blogger embed", "https://www.blogger.com/video.g?token=abc", false},
		{"unresolved blogspot embed", "https://example.blogspot.com/video.g?token=abc", false},
		{"megacloud embed page", "https://megacloud.tv/embed/abc123", false},
		{"megacloud club embed page", "https://megacloud.club/embed/abc123", false},
		{"playtaku page", "https://playtaku.com/watch/abc", false},
		{"streaming.php page", "https://example.com/streaming.php?id=1", false},
		{"javascript file", "https://example.com/player.js", false},
		{"html page", "https://example.com/watch.html", false},
		{"css file", "https://example.com/style.css", false},
		{"image file", "https://example.com/poster.png", false},
		{"unknown CDN path is allowed by default", "https://cdn.example.com/stream/abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isPlayableStreamURL(tt.url))
		})
	}
}
