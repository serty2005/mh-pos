// Package svg рендерит печатный IR в SVG-предпросмотр.
package svg

import (
	"fmt"
	"html"
	"strings"

	"mh-pos-platform/receipt/ir"
	"mh-pos-platform/receipt/layout"
)

const (
	charWidth   = 8
	lineHeight  = 18
	padding     = 12
	defaultQR   = 6
	qrModuleCnt = 21
)

// RenderOptions задают ширину предпросмотра в символах на строку.
type RenderOptions struct {
	CPL int
}

// Render преобразует IR в автономный SVG-документ.
func Render(blocks []ir.Block, opts RenderOptions) (string, error) {
	r := renderer{cpl: layout.NormalizeCPL(opts.CPL)}
	if err := r.renderBlocks(blocks); err != nil {
		return "", err
	}
	width := padding*2 + r.cpl*charWidth
	height := max(r.y+padding, padding*2+lineHeight)
	body := strings.Join(r.parts, "")
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img"><rect width="100%%" height="100%%" fill="#fff"/><g font-family="ui-monospace,SFMono-Regular,Menlo,Consolas,monospace" fill="#111">%s</g></svg>`, width, height, width, height, body), nil
}

type renderer struct {
	parts []string
	cpl   int
	y     int
}

func (r *renderer) renderBlocks(blocks []ir.Block) error {
	for _, block := range blocks {
		switch b := block.(type) {
		case ir.TextBlock:
			r.text(b)
		case ir.RuleBlock:
			r.textLine(strings.Repeat("-", r.cpl), ir.FontNormal, false)
		case ir.SpaceBlock:
			r.y += max(b.Lines, 1) * lineHeight
		case ir.QRBlock:
			r.qr(b)
		case ir.BarcodeBlock:
			r.barcode(b)
		case ir.ImageBlock:
			r.image(b)
		case ir.CutBlock:
			r.cut()
		case ir.DrawerBlock:
			// Импульс ящика не печатает ничего на бумаге.
		case ir.IfBlock:
			if err := r.renderBlocks(b.Blocks); err != nil {
				return err
			}
		case ir.EachBlock:
			if err := r.renderBlocks(b.Blocks); err != nil {
				return err
			}
		default:
			return fmt.Errorf("svg: unsupported block %T", block)
		}
	}
	return nil
}

func (r *renderer) text(block ir.TextBlock) {
	for _, line := range block.Lines {
		rendered := layout.RenderColumns(line.Columns, r.cpl)
		if len(line.Columns) == 1 {
			rendered = layout.Align(rendered, r.cpl, block.Alignment)
		}
		r.textLine(rendered, block.Font, block.Bold)
	}
}

func (r *renderer) textLine(text string, font ir.Font, bold bool) {
	size := fontSize(font)
	weight := "400"
	if bold {
		weight = "700"
	}
	r.y += lineHeight
	r.parts = append(r.parts, fmt.Sprintf(`<text x="%d" y="%d" font-size="%d" font-weight="%s" xml:space="preserve">%s</text>`, padding, r.y, size, weight, html.EscapeString(text)))
}

func (r *renderer) qr(block ir.QRBlock) {
	size := block.Size
	if size <= 0 {
		size = defaultQR
	}
	size = min(max(size, 1), 8)
	side := qrModuleCnt * size
	x := padding + (r.cpl*charWidth-side)/2
	if x < padding {
		x = padding
	}
	r.y += lineHeight
	r.parts = append(r.parts, fmt.Sprintf(`<g data-block="qr" data-size="%d" data-payload="%s"><rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="#111" stroke-width="2"/><path d="%s" stroke="#111" stroke-width="%d"/></g>`, size, html.EscapeString(block.Payload), x, r.y, side, side, qrPath(x, r.y, side), size))
	r.y += side
}

func (r *renderer) barcode(block ir.BarcodeBlock) {
	width := min(r.cpl*charWidth, 240)
	height := 44
	x := padding + (r.cpl*charWidth-width)/2
	r.y += lineHeight
	r.parts = append(r.parts, fmt.Sprintf(`<g data-block="barcode" data-type="%s" data-value="%s"><rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="#111"/><line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#111" stroke-width="3"/></g>`, html.EscapeString(block.Type), html.EscapeString(block.Data), x, r.y, width, height, x+8, r.y+6, x+width-8, r.y+height-6))
	r.y += height
	if block.HRI {
		r.textLine(block.Data, ir.FontSmaller, false)
	}
}

func (r *renderer) image(block ir.ImageBlock) {
	width, height := block.Width, block.Height
	if width <= 0 {
		width = r.cpl * charWidth
	}
	if height <= 0 {
		height = 32
	}
	width = min(width, r.cpl*charWidth)
	r.y += lineHeight
	r.parts = append(r.parts, fmt.Sprintf(`<rect data-block="image" x="%d" y="%d" width="%d" height="%d" fill="#f5f5f5" stroke="#111" stroke-dasharray="4 3"/>`, padding, r.y, width, height))
	r.y += height
}

func (r *renderer) cut() {
	r.y += lineHeight
	r.parts = append(r.parts, fmt.Sprintf(`<line data-block="cut" x1="%d" y1="%d" x2="%d" y2="%d" stroke="#777" stroke-dasharray="6 4"/>`, padding, r.y, padding+r.cpl*charWidth, r.y))
}

func fontSize(font ir.Font) int {
	switch font {
	case ir.FontDouble:
		return 22
	case ir.FontSmaller:
		return 11
	default:
		return 14
	}
}

func qrPath(x, y, side int) string {
	mid := side / 2
	return fmt.Sprintf("M%d %dH%dM%d %dV%dM%d %dL%d %d", x+side/4, y+mid, x+side*3/4, x+mid, y+side/4, y+side*3/4, x+side/4, y+side/4, x+side*3/4, y+side*3/4)
}
