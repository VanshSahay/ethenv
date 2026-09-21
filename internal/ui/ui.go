package ui

import (
	"fmt"
	"io"
	"strings"
)

const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
)

func OK(msg string) string { return Green + "✔ " + Reset + msg }

func Fail(msg string) string { return Red + "✘ " + Reset + msg }

func Warn(msg string) string { return Yellow + "! " + Reset + msg }

func Header(title string) string { return Bold + title + Reset }

func Table(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	line := func(cells []string) {
		parts := make([]string, len(cells))
		for i, c := range cells {
			if i < len(widths) {
				parts[i] = c + strings.Repeat(" ", widths[i]-len(c))
			}
		}
		fmt.Fprintln(w, "  "+strings.Join(parts, "  "))
	}

	line(headers)
	sep := make([]string, len(headers))
	for i, wd := range widths {
		sep[i] = strings.Repeat("-", wd)
	}
	fmt.Fprintln(w, "  "+strings.Join(sep, "  "))
	for _, row := range rows {
		line(row)
	}
}
