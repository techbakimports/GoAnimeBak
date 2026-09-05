package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2/spinner"

	"github.com/alvarorichard/Goanime/internal/api/manga"
	"github.com/alvarorichard/Goanime/internal/tui"
	"github.com/alvarorichard/Goanime/internal/util"
)

// HandleMangaRequest processes --manga requests: search MangaDex, let the
// user pick a title and a chapter, download its pages, and export the
// chapter as a CBZ or PDF file.
func HandleMangaRequest() error {
	util.InitLogger()

	if util.GlobalMangaRequest == nil {
		return fmt.Errorf("manga request is nil")
	}
	req := util.GlobalMangaRequest
	ctx := context.Background()
	client := manga.NewClient()

	selectedManga, err := searchAndSelectManga(ctx, client, req.Query)
	if err != nil {
		return err
	}

	selectedChapter, err := fetchAndSelectChapter(ctx, client, selectedManga)
	if err != nil {
		return err
	}

	return downloadAndExportChapter(ctx, client, selectedManga, selectedChapter, req.Format)
}

// searchAndSelectManga searches MangaDex and lets the user pick a result.
func searchAndSelectManga(ctx context.Context, client *manga.Client, query string) (*manga.MangaResult, error) {
	var results []manga.MangaResult
	var searchErr error

	_ = tui.RunClean(func() error {
		return spinner.New().
			Title(fmt.Sprintf("Searching MangaDex for %q...", query)).
			Type(spinner.Dots).
			Action(func() {
				results, searchErr = client.SearchManga(ctx, query)
			}).
			Run()
	})
	if searchErr != nil {
		return nil, fmt.Errorf("manga search failed: %w", searchErr)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no manga found for %q", query)
	}

	items := make([]tui.MenuItem, len(results))
	for i, m := range results {
		items[i] = tui.MenuItem{Label: formatMangaLabel(m), Value: strconv.Itoa(i)}
	}

	choice := tui.RunMenu("Select a manga", items)
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 0 || idx >= len(results) {
		return nil, fmt.Errorf("manga selection cancelled")
	}
	return &results[idx], nil
}

func formatMangaLabel(m manga.MangaResult) string {
	label := m.Title
	if m.Year > 0 {
		label = fmt.Sprintf("%s (%d)", label, m.Year)
	}
	if m.Status != "" {
		label = fmt.Sprintf("%s [%s]", label, m.Status)
	}
	return label
}

// fetchAndSelectChapter fetches a manga's chapter feed and lets the user
// pick a chapter.
func fetchAndSelectChapter(ctx context.Context, client *manga.Client, m *manga.MangaResult) (*manga.ChapterResult, error) {
	var chapters []manga.ChapterResult
	var fetchErr error

	_ = tui.RunClean(func() error {
		return spinner.New().
			Title(fmt.Sprintf("Fetching chapters for %s...", m.Title)).
			Type(spinner.Dots).
			Action(func() {
				chapters, fetchErr = client.GetChapters(ctx, m.ID)
			}).
			Run()
	})
	if fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch chapters: %w", fetchErr)
	}

	items := make([]tui.MenuItem, len(chapters))
	for i, ch := range chapters {
		items[i] = tui.MenuItem{Label: formatChapterLabel(ch), Value: strconv.Itoa(i)}
	}

	choice := tui.RunMenu(fmt.Sprintf("Select a chapter (%s)", m.Title), items)
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 0 || idx >= len(chapters) {
		return nil, fmt.Errorf("chapter selection cancelled")
	}
	return &chapters[idx], nil
}

func formatChapterLabel(ch manga.ChapterResult) string {
	label := "Chapter"
	if ch.Chapter != "" {
		label = fmt.Sprintf("Chapter %s", ch.Chapter)
	}
	if ch.Title != "" {
		label = fmt.Sprintf("%s — %s", label, ch.Title)
	}
	return fmt.Sprintf("%s [%s]", label, ch.Language)
}

// downloadAndExportChapter resolves the chapter's page URLs, downloads them
// to a temporary directory (cleaned up afterward), and bundles the result
// into the requested format under util.DefaultMangaDownloadDir().
func downloadAndExportChapter(ctx context.Context, client *manga.Client, m *manga.MangaResult, ch *manga.ChapterResult, format string) error {
	pageURLs, err := client.GetChapterPageURLs(ctx, ch.ID)
	if err != nil {
		return fmt.Errorf("failed to resolve chapter pages: %w", err)
	}

	chapterLabel := util.SanitizeForFilename(chapterNumberOrID(ch))
	baseDir := filepath.Join(util.DefaultMangaDownloadDir(), util.SanitizeForFilename(m.Title))
	tmpDir := filepath.Join(baseDir, "tmp_ch_"+chapterLabel)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	util.Infof("Downloading %d page(s)...", len(pageURLs))
	pageFiles, err := downloadPagesWithProgress(ctx, pageURLs, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to download chapter pages: %w", err)
	}

	outPath := filepath.Join(baseDir, fmt.Sprintf("%s - Ch.%s.%s", util.SanitizeForFilename(m.Title), chapterLabel, format))

	util.Infof("Building %s...", format)
	if format == "pdf" {
		err = manga.BuildPDF(pageFiles, outPath)
	} else {
		err = manga.BuildCBZ(pageFiles, outPath)
	}
	if err != nil {
		return fmt.Errorf("failed to build %s: %w", format, err)
	}

	util.Infof("Saved to: %s", outPath)
	return nil
}

// chapterNumberOrID returns the chapter number for display/filenames,
// falling back to the chapter's MangaDex ID for oneshots with no number.
func chapterNumberOrID(ch *manga.ChapterResult) string {
	if ch.Chapter != "" {
		return ch.Chapter
	}
	return ch.ID
}

// --- page download progress bar ---

// mangaProgressMsg reports how many pages have finished downloading.
type mangaProgressMsg struct {
	done, total int
}

// mangaProgressModel renders a Bubble Tea progress bar for the page
// download phase, in the same spirit as internal/downloader's progressModel
// but tracking page count instead of bytes.
type mangaProgressModel struct {
	progress progress.Model
	done     int
	total    int
	peakPct  float64
	mu       sync.Mutex
}

func (m *mangaProgressModel) Init() tea.Cmd { return nil }

func (m *mangaProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case mangaProgressMsg:
		m.mu.Lock()
		m.done = msg.done
		m.total = msg.total
		pct := 0.0
		if m.total > 0 {
			pct = float64(m.done) / float64(m.total)
		}
		if pct < m.peakPct { // monotonic: never go backward
			pct = m.peakPct
		} else {
			m.peakPct = pct
		}
		cmd := m.progress.SetPercent(pct)
		m.mu.Unlock()
		return m, cmd
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *mangaProgressModel) View() tea.View {
	m.mu.Lock()
	defer m.mu.Unlock()
	return tea.NewView(fmt.Sprintf("Downloading pages (%d/%d)\n%s\n\nPress Ctrl+C to cancel",
		m.done, m.total, m.progress.View()))
}

// downloadPagesWithProgress downloads urls to destDir while driving a
// Bubble Tea progress bar, blocking until the download finishes.
func downloadPagesWithProgress(ctx context.Context, urls []string, destDir string) ([]string, error) {
	m := &mangaProgressModel{progress: progress.New(progress.WithDefaultBlend())}
	p := tui.NewProgram(m)

	var pageFiles []string
	var downloadErr error

	go func() {
		pageFiles, downloadErr = manga.DownloadPages(ctx, urls, destDir, func(done, total int) {
			p.Send(mangaProgressMsg{done: done, total: total})
		})
		time.Sleep(200 * time.Millisecond)
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		return nil, fmt.Errorf("progress display error: %w", err)
	}
	return pageFiles, downloadErr
}
