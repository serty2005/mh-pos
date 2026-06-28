// Package layout содержит общую CPL-разметку для печатных рендереров.
package layout

import (
	"strings"
	"unicode/utf8"

	"mh-pos-platform/receipt/ir"
)

const defaultCPL = 48

// NormalizeCPL возвращает дефолтную ширину ленты, если CPL не задан.
func NormalizeCPL(cpl int) int {
	if cpl <= 0 {
		return defaultCPL
	}
	return cpl
}

// RenderColumns собирает логическую строку в фиксированную CPL-ширину.
func RenderColumns(columns []ir.Column, cpl int) string {
	if len(columns) == 0 {
		return ""
	}
	widths := ColumnWidths(columns, NormalizeCPL(cpl))
	var b strings.Builder
	for i, col := range columns {
		b.WriteString(Align(Fit(col.Text, widths[i]), widths[i], col.Align))
	}
	return b.String()
}

// ColumnWidths делит CPL между fixed и auto колонками так же для SVG и ESC/POS.
func ColumnWidths(columns []ir.Column, cpl int) []int {
	widths := make([]int, len(columns))
	fixed, auto := 0, 0
	for i, col := range columns {
		if col.Width > 0 {
			widths[i] = col.Width
			fixed += col.Width
			continue
		}
		auto++
	}
	remaining := max(NormalizeCPL(cpl)-fixed, auto)
	if auto == 0 {
		return widths
	}
	base, extra := remaining/auto, remaining%auto
	for i := range widths {
		if widths[i] != 0 {
			continue
		}
		widths[i] = base
		if extra > 0 {
			widths[i]++
			extra--
		}
	}
	return widths
}

// Align дополняет строку пробелами до нужной ширины.
func Align(s string, width int, mode ir.Alignment) string {
	padding := width - utf8.RuneCountInString(s)
	if padding <= 0 {
		return s
	}
	switch mode {
	case ir.AlignRight:
		return strings.Repeat(" ", padding) + s
	case ir.AlignCenter:
		left := padding / 2
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", padding-left)
	default:
		return s + strings.Repeat(" ", padding)
	}
}

// Fit обрезает строку по rune count, сохраняя UTF-8.
func Fit(s string, width int) string {
	if width <= 0 || utf8.RuneCountInString(s) <= width {
		return s
	}
	out := make([]rune, 0, width)
	for _, r := range s {
		if len(out) == width {
			break
		}
		out = append(out, r)
	}
	return string(out)
}
