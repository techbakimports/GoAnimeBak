package types

import (
	"fmt"
	"strings"

	"github.com/alvarorichard/Goanime/internal/scraper"
)

// Source represents an anime scraper source
type Source int

const (
	// SourceAllAnime represents the AllAnime source
	SourceAllAnime Source = iota
	// SourceAnimeFire represents the AnimeFire source
	SourceAnimeFire
	// SourceGoyabu represents the Goyabu source (PT-BR)
	SourceGoyabu
	// SourceGogoAnime represents the GogoAnime source (English)
	SourceGogoAnime
	// SourceAnimesOnlineCC represents the AnimesOnlineCC source (PT-BR)
	SourceAnimesOnlineCC
	// SourceAnimeHeaven represents the AnimeHeaven source (English)
	SourceAnimeHeaven
)

// String returns the string representation of the source
func (s Source) String() string {
	switch s {
	case SourceAllAnime:
		return "AllAnime"
	case SourceAnimeFire:
		return "AnimeFire"
	case SourceGoyabu:
		return "Goyabu"
	case SourceGogoAnime:
		return "GogoAnime"
	case SourceAnimesOnlineCC:
		return "AnimesOnlineCC"
	case SourceAnimeHeaven:
		return "AnimeHeaven"
	default:
		return "Unknown"
	}
}

// ToScraperType converts the public Source type to internal ScraperType
func (s Source) ToScraperType() scraper.ScraperType {
	switch s {
	case SourceAllAnime:
		return scraper.AllAnimeType
	case SourceAnimeFire:
		return scraper.AnimefireType
	case SourceGoyabu:
		return scraper.GoyabuType
	case SourceGogoAnime:
		return scraper.GogoAnimeType
	case SourceAnimesOnlineCC:
		return scraper.AnimesOnlineCCType
	case SourceAnimeHeaven:
		return scraper.AnimeHeavenType
	default:
		return scraper.AllAnimeType
	}
}

// ParseSource parses a string into a Source type.
// Accepts both canonical names ("AllAnime") and display names ("Animefire.io").
func ParseSource(s string) (Source, error) {
	lower := strings.ToLower(s)
	switch {
	case lower == "allanime" || lower == "all":
		return SourceAllAnime, nil
	case lower == "animefire" || lower == "fire" || lower == "animefire.io":
		return SourceAnimeFire, nil
	case lower == "goyabu":
		return SourceGoyabu, nil
	case lower == "gogoanime":
		return SourceGogoAnime, nil
	case lower == "animesonlinecc" || lower == "animescc":
		return SourceAnimesOnlineCC, nil
	case lower == "animeheaven" || lower == "heaven":
		return SourceAnimeHeaven, nil
	default:
		return SourceAllAnime, fmt.Errorf("unknown source: %s", s)
	}
}
