package playback

import (
	"fmt"
	"testing"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestSelectEpisodeWithFuzzy_EmptyList verifies that passing an empty episode
// list returns an error instead of calling log.Fatal (which was the old behavior).
func TestSelectEpisodeWithFuzzy_EmptyList(t *testing.T) {
	_, _, _, err := SelectEpisodeWithFuzzy([]models.Episode{})

	assert.Error(t, err, "expected error for empty episode list")
	t.Logf("Got expected error: %v", err)
}

// TestFindEpisodeByNumber_NotFound verifies that searching for a non-existent
// episode number delegates to the fallback selector and propagates its error
// instead of calling os.Exit.
func TestFindEpisodeByNumber_NotFound(t *testing.T) {
	// Stub out the interactive fallback so the test never opens a TUI.
	orig := episodeFallback
	episodeFallback = func(_ []models.Episode) (string, string, int, error) {
		return "", "", 0, fmt.Errorf("episode not found (stub fallback)")
	}
	defer func() { episodeFallback = orig }()

	episodes := []models.Episode{
		{URL: "https://example.com/ep1", Number: "1"},
		{URL: "https://example.com/ep2", Number: "2"},
	}

	// Episode 999 doesn't exist — FindEpisodeByNumber should call the
	// fallback, which returns our stub error.
	_, _, _, err := FindEpisodeByNumber(episodes, 999)

	assert.Error(t, err, "expected error for non-existent episode number")
	t.Logf("Got expected error: %v", err)
}
