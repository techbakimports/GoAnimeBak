# GoAnime Release Notes - Version 1.8.4

Release date: 2026-05-03

## Highlights

- **SuperFlix API Migration**: Migrated SuperFlix endpoint from `superflixapi.rest` (deprecated) to `superflixapi.online` (canonical). The legacy host performs a 301 redirect, but Go's HTTP Client downgrades POST to GET, breaking requests. Updated across all code and tests.
- **SuperFlix Content Filtering**: Implemented episode filtering by air date, removing episodes with missing, "null", or future air dates. This resolves issues where placeholder episodes appeared without available sources.
- **SuperFlix Error Handling Improvements**: Implemented `ensureJSONResponse()` to detect HTML or non-2xx responses and produce clear error messages instead of cryptic "invalid character '<'". Now performs body-sniffing instead of relying solely on Content-Type header.
- **Temporarily Disabled Sources**: FlixHQ, SFlix, and 9Anime commented out until fixes are implemented. AnimeDrive also temporarily disabled.
- **Conditional Logic Refactoring**: Replacement of complex if-else chains with switch statements for improved readability and maintainability across the codebase.
- **Version Bumped**: Updated to 1.8.4 with automatic normalization of the "v" prefix in CI workflows.

## Features

- Migrate SuperFlix API endpoint to canonical `superflixapi.online` host.
- Add content filtering for SuperFlix episodes: remove episodes with empty, "null", or future air dates.
- Implement `ensureJSONResponse()` helper for clear detection of HTTP/HTML errors in SuperFlix APIs.
- Refactor conditional chains across multiple files: replace nested if-else with switch statements for improved readability.
- Add support for non-UTC timezones in SuperFlix episode filtering (normalizes to UTC internally).
- Version normalization: automatic TrimPrefix("v") in CI builds to avoid "vv1.8.4" in outputs.

## Bug Fixes

- **Critical**: Fix SuperFlix API host migration: update from `superflixapi.rest` to `superflixapi.online`. The 301 redirect from the legacy host downgrades POST to GET, breaking `/player/bootstrap`. Legacy code received HTML 404 and failed with "invalid character '<' looking for beginning of value".
- **Critical**: Fix SuperFlix episode filtering: placeholder episodes with `air_date: null` or `air_date: ""` are now removed before returning to the user. Episodes with future air dates are also filtered. This resolves issues where users saw "no servers available" for episodes that haven't yet been released.
- Fix SuperFlix error messages: implement `ensureJSONResponse()` to detect HTML or non-2xx status codes and produce clear messages instead of "invalid character '<'". Logs now indicate the endpoint, status code, and final URL for diagnosis.
- Fix SuperFlix air_date timezone handling: filtering now uses UTC internally, preventing leakage of tomorrow's episodes in timezones west of UTC.
- Temporarily disable FlixHQ, SFlix, and 9Anime source providers until fixes are ready. Sources commented out with `/* */` blocks in provider registry.
- Temporarily disable AnimeDrive scraper until Cloudflare bypass is implemented.
- Fix version normalization: strip leading "v" from injected CI version string to avoid "vv1.8.4" in outputs.
- Refactor logging: replace `log.Fatalln()` with `util.Errorf()` in error handlers of main.go, improving graceful shutdown.
- Remove Nix configuration files (default.nix, flake.nix, flake.lock, gomod2nix.toml) from repository.

## Improvements

- Refactor conditional logic: replace if-else chains with switch statements across multiple modules (`allanime_enhanced.go`, `player.go`, `playvideo.go`, `scraper.go`, `download.go`, `appflow/anime_data.go`, `util/util.go`, `util/perf.go`, `scraper/flixhq.go`, `scraper/sflix.go`, `scraper/superflix.go`) for improved readability and maintainability.
- Add helper function `filterEpisodesByAirDate()` in SuperFlix scraper to centralize episode date filtering logic.
- Improve SuperFlix error diagnostics: enrich error messages with player URL, content ID, and HTTP status for easier triage.
- Add `ErrSuperFlixNoServers` typed error to distinguish content-unavailability (placeholder episodes) from system errors.
- Rename `isFlixHQSourcePlayer()` to `isMovieOrTVSourcePlayer()` in player.go to reflect that both SuperFlix and FlixHQ share the same stream extraction route.
- Add comprehensive regression test suites:
  - `runspinner_regression_test.go`: tests for spinner race condition where action completed after runner returned.
  - `source_routing_regression_test.go`: tests to validate correct routing of SuperFlix and FlixHQ through enhanced API.
  - `allanime_ctr_regression_test.go`: improvements in slice allocation to avoid inefficient append operations.
  - `superflix_test.go`: tests for air_date filtering, JSON response validation, HTML error detection, and timezone handling.
  - `version_regression_test.go`: tests for version string normalization without "v" prefix.
- Update go.mod/go.sum: dependency bump (klauspost/compress v1.18.5 → v1.18.6).
- Minor code style improvements: replace `baseURL = baseURL + "/"` with `baseURL += "/"` and `timePerFrame = timePerFrame / 2` with `timePerFrame /= 2`.

---

