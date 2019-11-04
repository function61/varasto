package tui

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestProgressBar(t *testing.T) {
	assert.EqualString(t, ProgressBar(0, 20, ProgressBarDefaultTheme()), "░░░░░░░░░░░░░░░░░░░░")
	assert.EqualString(t, ProgressBar(50, 20, ProgressBarDefaultTheme()), "██████████░░░░░░░░░░")
	assert.EqualString(t, ProgressBar(100, 20, ProgressBarDefaultTheme()), "████████████████████")
}

func TestProgressBarThemes(t *testing.T) {
	assert.EqualString(t, ProgressBar(13, 20, ProgressBarDefaultTheme()), "██░░░░░░░░░░░░░░░░░░")
	assert.EqualString(t, ProgressBar(13, 20, ProgressBarCirclesTheme()), "⬤⬤○○○○○○○○○○○○○○○○○○")
}
