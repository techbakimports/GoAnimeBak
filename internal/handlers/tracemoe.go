package handlers

import (
	"context"
	"fmt"

	"github.com/alvarorichard/Goanime/internal/api/tracemoe"
	"github.com/alvarorichard/Goanime/internal/util"
)

// maxTraceMoeResults caps how many candidate matches are printed to the user.
const maxTraceMoeResults = 3

// HandleTraceMoeRequest processes --identify requests: uploads a screenshot
// to trace.moe and prints the best-matching anime scenes.
func HandleTraceMoeRequest() error {
	util.InitLogger()

	if util.GlobalTraceMoeRequest == nil {
		return fmt.Errorf("identify request is nil")
	}
	req := util.GlobalTraceMoeRequest

	util.Infof("Identifying scene from: %s", req.ImagePath)

	matches, err := tracemoe.NewClient().Search(context.Background(), req.ImagePath)
	if err != nil {
		return fmt.Errorf("scene identification failed: %w", err)
	}

	limit := maxTraceMoeResults
	if len(matches) < limit {
		limit = len(matches)
	}

	for i := 0; i < limit; i++ {
		printTraceMoeMatch(i+1, matches[i])
	}

	return nil
}

// printTraceMoeMatch prints one ranked match in a human-readable line.
func printTraceMoeMatch(rank int, m tracemoe.Match) {
	title := m.TitleEnglish
	if title == "" {
		title = m.TitleRomaji
	}
	if title == "" {
		title = m.TitleNative
	}
	if title == "" {
		title = "Unknown title"
	}

	episodeInfo := ""
	if m.Episode != "" {
		episodeInfo = fmt.Sprintf(", episode %s", m.Episode)
	}

	util.Infof("%d. %s%s — %s (similarity: %.1f%%)",
		rank, title, episodeInfo,
		tracemoe.FormatTimestamp(m.From), m.Similarity*100)
}
