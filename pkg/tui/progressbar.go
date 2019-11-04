// Utils for text-based UIs
package tui

func ProgressBar(pct int, barLength int, theme ProgressBarTheme) string {
	r := make([]rune, barLength)

	ratio := float64(barLength) * float64(pct) / 100.0

	for i := 0; i < barLength; i++ {
		ch := theme.Vacant
		if float64(i+1) <= ratio {
			ch = theme.Filled
		}

		r[i] = ch
	}

	return string(r)
}

type ProgressBarTheme struct {
	Filled rune
	Vacant rune
}

func ProgressBarDefaultTheme() ProgressBarTheme {
	return ProgressBarTheme{'█', '░'}
}

func ProgressBarCirclesTheme() ProgressBarTheme {
	return ProgressBarTheme{'⬤', '○'}
}
