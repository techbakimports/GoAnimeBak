package handlers

import (
	"testing"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestIsImageExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".png", true},
		{".jpg", true},
		{".jpeg", true},
		{".gif", true},
		{".bmp", true},
		{".tiff", true},
		{".webp", true},
		{".mp4", false},
		{".mkv", false},
		{".avi", false},
		{"", false},
		{".PNG", false}, // caller is responsible for lowercasing first
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			assert.Equal(t, tt.expected, isImageExtension(tt.ext))
		})
	}
}

func TestNeedsInteractiveEpisodes(t *testing.T) {
	tests := []struct {
		name      string
		mediaType models.MediaType
		expected  bool
	}{
		{"movie requires sequential fetch", models.MediaTypeMovie, true},
		{"tv requires sequential fetch", models.MediaTypeTV, true},
		{"anime can fetch in parallel", models.MediaTypeAnime, false},
		{"unknown media type defaults to parallel", models.MediaType("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, needsInteractiveEpisodes(tt.mediaType))
		})
	}
}

func TestIsSeriesPlayback(t *testing.T) {
	tests := []struct {
		name          string
		anime         *models.Anime
		totalEpisodes int
		expected      bool
	}{
		{
			name:          "anime with multiple episodes is a series",
			anime:         &models.Anime{MediaType: models.MediaTypeAnime},
			totalEpisodes: 12,
			expected:      true,
		},
		{
			name:          "anime with a single episode is treated as a movie",
			anime:         &models.Anime{MediaType: models.MediaTypeAnime},
			totalEpisodes: 1,
			expected:      false,
		},
		{
			name:          "explicit movie media type is never a series, even with >1 episode entries",
			anime:         &models.Anime{MediaType: models.MediaTypeMovie},
			totalEpisodes: 3,
			expected:      false,
		},
		{
			name:          "tv show with multiple episodes is a series",
			anime:         &models.Anime{MediaType: models.MediaTypeTV},
			totalEpisodes: 24,
			expected:      true,
		},
		{
			name:          "zero episodes is not a series",
			anime:         &models.Anime{MediaType: models.MediaTypeAnime},
			totalEpisodes: 0,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isSeriesPlayback(tt.anime, tt.totalEpisodes))
		})
	}
}

func TestHandleDownloadRequest_NilRequest(t *testing.T) {
	original := util.GlobalDownloadRequest
	defer func() { util.GlobalDownloadRequest = original }()

	util.GlobalDownloadRequest = nil

	err := HandleDownloadRequest()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download request is nil")
}

func TestHandleUpscaleRequest_NilRequest(t *testing.T) {
	original := util.GlobalUpscaleRequest
	defer func() { util.GlobalUpscaleRequest = original }()

	util.GlobalUpscaleRequest = nil

	err := HandleUpscaleRequest()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upscale request is nil")
}

func TestHandleTraceMoeRequest_NilRequest(t *testing.T) {
	original := util.GlobalTraceMoeRequest
	defer func() { util.GlobalTraceMoeRequest = original }()

	util.GlobalTraceMoeRequest = nil

	err := HandleTraceMoeRequest()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "identify request is nil")
}

func TestHandleMangaRequest_NilRequest(t *testing.T) {
	original := util.GlobalMangaRequest
	defer func() { util.GlobalMangaRequest = original }()

	util.GlobalMangaRequest = nil

	err := HandleMangaRequest()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manga request is nil")
}
