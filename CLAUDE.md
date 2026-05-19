# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

GoAnime is a CLI text-based user interface (TUI) written in Go that allows users to search for anime and play/download episodes directly in mpv. The application scrapes data from multiple streaming sources (AllAnime, AnimeFire, SuperFlix, FlixHQ, and others) with automatic fallback support.

**Repository**: github.com/alvarorichard/Goanime  
**Language**: Go 1.26.2  
**Lines of Code**: ~57,000  
**Test Coverage**: 77 test files across the codebase

## Building & Running

### Development Build
```bash
# Build the application
go build -o bin/goanime ./cmd/goanime

# Run locally
./bin/goanime "Naruto"

# Run with search term
./bin/goanime "Attack on Titan"
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/scraper/...
go test ./internal/player/...

# Run a single test
go test -run TestFunctionName ./internal/package/...

# Run tests with verbose output
go test -v ./...

# Run tests excluding integration tests (no external API calls)
go test -short ./...

# Run macOS-specific socket tests (macOS only)
go test -race -v ./internal/player/test/... -run "Socket|MacOS"
```

### Code Quality & Linting
```bash
# Format all Go code (required before commits)
go fmt ./...

# Run static analysis (built-in)
go vet ./...

# Run advanced static analysis
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...

# Run security scanner
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# Check for known vulnerabilities in dependencies
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Run comprehensive linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run

# Run all checks (pre-commit checklist)
go fmt ./... && go vet ./... && staticcheck ./... && gosec ./... && govulncheck ./... && golangci-lint run
```

### Platform-Specific Builds
```bash
# Linux
./build/buildlinux.sh
./build/buildlinux-with-sqlite.sh

# macOS
./build/buildmacos.sh

# Windows
./build/buildwindows.sh
```

### Dependencies
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Check for updates
go list -u -m all
```

## Architecture Overview

### High-Level Data Flow

```
User Input (CLI)
    ↓
[handlers] - Routes requests to appropriate workflow
    ↓
[appflow] - Manages orchestration of data and playback
    ↓
[api] - Fetches anime/episode data and metadata
    ↓
[scraper] - Multi-source web scraping (AllAnime, AnimeFire, SuperFlix, etc.)
    ↓
[models] - Core data structures (Anime, Episode, URLs)
    ↓
[player] - Platform-specific video playback (mpv integration)
    ↓
[discord] - Discord Rich Presence integration
    ↓
[tracking] - Watch history and progress persistence
```

### Core Packages

#### cmd/goanime/
- **main.go**: Application entry point. Handles:
  - Terminal state preservation and restoration
  - Signal handling for graceful shutdown
  - Performance timing and profiling
  - Pre-warming HTTP clients and scrapers
  - Routing to handlers based on CLI flags (playback, download, update, upscale)

#### internal/handlers/
Orchestrates high-level workflows:
- **playback.go**: Main playback workflow (search → select anime/episode → play)
- **media.go**: Download workflow management
- **download.go**: Media download orchestration
- **update.go**: Auto-update functionality
- **upscale.go**: Anime4K upscaling integration

#### internal/appflow/
Application flow management and orchestration:
- **playback.go**: Coordinates playback logic across modules
- **media.go**: Handles media-specific workflows (movies vs. TV shows)
- **anime_data.go**: Data enrichment and aggregation

#### internal/api/
Fetches and manages anime/episode metadata from external APIs:
- **api.go**: SSRF protection, TLS configuration, network utilities
- **anime.go**, **episodes.go**: Core anime/episode data fetching
- **episode_providers.go**: Multi-source provider resolution (routes to correct scraper)
- **episode_providers_test.go**: Provider selection logic tests
- **enhanced.go**: Enhanced API with multi-source fallback support
- **allanime_smart.go**, **allanime_enhanced.go**: AllAnime API integration
- **aniskip.go**: Skippable OP/ED timestamp resolution
- **movie/**: Movie enrichment APIs (TMDB, OMDB metadata)
- **providers/**: Source provider registry and metadata management

#### internal/scraper/
Multi-source web scraping implementations:
- **unified.go**: Unified scraper interface (UnifiedScraper) that all sources implement
- **allanime.go**: AllAnime.day scraper (GraphQL API-based)
- **animefire.go**: Animefire.io scraper (HTML parsing)
- **superflix.go**: SuperFlix scraper (movie/TV sources)
- **flixhq.go**, **sflix.go**: FlixHQ and SFlix implementations
- **nineanime.go**: 9Anime scraper
- **animedrive.go**: AnimeDrive scraper
- **goyabu.go**: GoYabu scraper
- **media_manager.go**: Manages movie vs. TV show scraping logic
- **source_circuit.go**: Circuit breaker pattern for source health management
- **source_health.go**: Tracks source availability and fallback logic
- **source_diagnostic.go**: Debugging and diagnostics for source issues
- **ssrf.go**: SSRF attack prevention

#### internal/models/
Core data structures:
- **anime.go**: Anime type (name, URL, episodes, metadata)
- **skip.go**: Skip times for opening/ending themes
- **tmdb.go**: The Movie Database types
- **urls.go**: URL utilities and validation
- **media.go**: Movie/TV show distinction types

#### internal/player/
Video playback abstraction:
- **player.go**: Platform-agnostic player interface
- **unix.go**, **windows.go**: Platform-specific mpv integration
- **tracker.go**: Watch progress and resume functionality
- **test/**: Platform-specific tests (socket handling on macOS, etc.)

#### internal/discord/
Discord Rich Presence integration:
- Shows what anime/episode is currently playing

#### internal/downloader/ & internal/download/
Episode download functionality:
- **hls/**: HLS/m3u8 stream downloading

#### internal/util/
Utility functions:
- **util.go**: Core utilities (flag parsing, error handling, logging)
- **logger.go**: Structured logging and debug output
- **httpclient.go**: Reusable HTTP client with Chrome TLS, timeouts, retries
- **help.go**: CLI help text generation
- **perf.go**: Performance profiling and timing utilities
- **ytdlp.go**: yt-dlp integration for stream downloading

#### pkg/goanime/
Public library API exposing core scraping/searching functionality:
- **client.go**: Public client for external projects to use GoAnime as a library
- **types/**: Public type definitions (Anime, Episode, Source enum)
- **examples/**: Example usage (search, episodes, stream URL retrieval, source-specific search)

## Key Architectural Patterns

### Multi-Source Provider System
Sources can be temporarily disabled (commented out with /* */ blocks) in the provider registry. The source_circuit.go and source_health.go modules track source availability and implement fallback logic. When a source fails, the system automatically tries the next available source.

**Provider Registry**: internal/api/providers/registry.go contains the authoritative list of enabled sources.

### Unified Scraper Interface
All scrapers implement the UnifiedScraper interface defined in internal/scraper/unified.go. This allows seamless substitution and fallback between sources.

### Movie vs. TV Show Logic
The scraper distinguishes between movies and TV shows via internal/scraper/media_manager.go. Some sources (SuperFlix, FlixHQ, SFlix) primarily serve movies/TV shows, while others (AllAnime, AnimeFire) focus on anime. The media manager routes requests to the appropriate scraper.

### Terminal State Management
main.go preserves terminal state before entering raw mode (used by TUI libraries like bubbletea and go-fuzzyfinder). On exit or interrupt, the terminal is restored to prevent user shells from being left in a broken state.

### Performance Pre-Warming
main.go pre-warms:
- HTTP clients (Chrome TLS handshake is expensive)
- mpv binary path lookup
- Scraper manager initialization
- Database connections

This minimizes latency when the user first searches.

## Development Workflow & Branching

**Branching Strategy:**
- **main**: Production-ready, always stable
- **dev**: Integration branch for all features
- **feature/***: Feature branches (create from dev)
- **bugfix/***: Bug fix branches (create from dev)
- **hotfix/***: Critical fixes that go directly to main

**Rules:**
- Never commit directly to main
- All changes go through dev first
- Create feature/bugfix branches from dev
- Use conventional commit messages: type(scope): description (e.g., feat(scraper): add FlixHQ support)

**Commit Types:**
- feat: New feature
- fix: Bug fix
- docs: Documentation
- style: Code style
- refactor: Refactoring
- test: Tests
- chore: Maintenance

## Testing Strategy

### Test Organization
- Test files are named *_test.go and live in the same package as tested code
- Use table-driven tests for multiple test cases
- Integration tests that require external APIs can be skipped with -short flag

### Test Coverage Areas
- **API layer** (internal/api/*_test.go): API response parsing, GraphQL queries
- **Scraper layer** (internal/scraper/*_test.go): HTML parsing, source resolution
- **Player integration** (internal/player/test/): Platform-specific socket handling
- **Regression tests**: Named files like *_regression_test.go test for previously fixed bugs

### Example Regression Tests
- runspinner_regression_test.go: Tests spinner race conditions
- source_routing_regression_test.go: Tests correct scraper routing
- superflix_test.go: Tests SuperFlix API migration and error handling
- allanime_ctr_regression_test.go: Tests AllAnime provider resolution

## CI/CD Pipeline

GitHub Actions (.github/workflows/ci.yml):
1. Checks out code
2. Sets up Go 1.26.2
3. Installs platform-specific dependencies (mpv)
4. Runs golangci-lint (linting)
5. Runs gosec (security scanning)
6. Runs govulncheck (dependency vulnerability check)
7. Executes full test suite across Ubuntu, Windows, and macOS

**Status Badge**: Build status is visible in README.md

## Important Notes

### Known Limitations & Workarounds
- **Temporarily Disabled Sources**: FlixHQ, SFlix, and 9Anime are commented out in the provider registry (v1.8.4). AnimeDrive is also disabled pending Cloudflare bypass implementation.
- **SuperFlix API Migration**: Uses superflixapi.online (not deprecated superflixapi.rest). Filtering by air_date removes placeholder episodes.
- **Terminal Restoration**: On abnormal exit, terminal may not restore properly if cleanup handlers fail. The application attempts to reset ANSI attributes and restore cursor visibility.

### Common File Locations
- **Build scripts**: ./build/buildlinux.sh, ./build/buildmacos.sh, ./build/buildwindows.sh
- **Configuration**: .golangci.yml (linting rules), .deepsource.toml (code quality)
- **Release notes**: CHANGELOG.md (detailed version history)
- **Development guide**: docs/Development.md (comprehensive contributor guide)
- **Scraping guide**: docs/SCRAPING_INTEGRATION.md (multi-source architecture)
- **Library usage**: docs/LIBRARY_USAGE.md (using GoAnime as a Go library)

## Debugging & Troubleshooting

### Debug Logging
The application uses util.Errorf() and util.LogDebug() for logging. Debug output is written to a debug log file. Use util.Logger for structured logging.

### Performance Profiling
main.go uses util.StartTimer() to measure total execution time. Utilities in internal/util/perf.go provide profiling helpers.

### Testing Edge Cases
- **macOS Socket Paths**: Use -run "Socket|MacOS" to run platform-specific tests
- **Race Conditions**: Use go test -race to detect concurrent access issues
- **HTTP Mocking**: Tests that call external APIs should mock responses or use short tests (-short flag)

## Contributing Guidelines

All contributions must pass:
1. go fmt formatting check
2. go vet static analysis
3. staticcheck advanced analysis
4. gosec security scanning
5. govulncheck dependency vulnerability check
6. golangci-lint comprehensive linting
7. Full test suite (go test ./...)

Features require unit tests covering main functionality and edge cases. Bug fixes require a test that reproduces the bug and verifies the fix. Documentation-only changes do not require tests.

See docs/Development.md for the complete development guide with code style examples and documentation standards.
