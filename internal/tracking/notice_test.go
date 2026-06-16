package tracking

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read pipe: %v", err)
	}
	return string(out)
}

func TestHandleTrackingNotice_CgoDisabled(t *testing.T) {
	original := IsCgoEnabled
	defer func() { IsCgoEnabled = original }()

	IsCgoEnabled = false

	output := captureStdout(t, HandleTrackingNotice)

	assert.True(t, strings.Contains(output, "tracking disabled"),
		"expected a disabled-tracking notice, got: %q", output)
}

func TestHandleTrackingNotice_CgoEnabled(t *testing.T) {
	original := IsCgoEnabled
	defer func() { IsCgoEnabled = original }()

	IsCgoEnabled = true

	output := captureStdout(t, HandleTrackingNotice)

	assert.Empty(t, output, "no notice should be printed when CGO/SQLite tracking is available")
}
