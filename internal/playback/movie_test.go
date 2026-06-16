package playback

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestCreateUpdater_DiscordDisabled(t *testing.T) {
	anime := &models.Anime{Name: "Test Anime"}
	isPaused := false
	var mu sync.Mutex

	updater := createUpdater(anime, &isPaused, &mu, time.Minute, false)

	assert.Nil(t, updater, "no updater should be created when Discord is disabled")
}

func TestGetSocketPath(t *testing.T) {
	path := getSocketPath()

	if runtime.GOOS == "windows" {
		assert.Equal(t, `\\.\pipe\goanime_mpvsocket`, path)
	} else {
		assert.Contains(t, path, "mpvsocket")
	}
}
