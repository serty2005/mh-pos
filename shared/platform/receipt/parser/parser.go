// Package parser разбирает ReceiptLine Level 1 в печатный IR.
package parser

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"mh-pos-platform/receipt/ir"
)

var barcodeTypes = map[string]bool{
	"code39":  true,
	"ean13":   true,
	"ean8":    true,
	"upca":    true,
	"upce":    true,
	"itf":     true,
	"codabar": true,
}

// Parse разбирает ReceiptLine Level 1 markup и возвращает width-independent IR.
func Parse(markup string) ([]ir.Block, error) {
	p := &documentParser{}
	normalized := strings.TrimSuffix(strings.ReplaceAll(markup, "\r\n", "\n"), "\n")
	for lineNo, line := range strings.Split(normalized, "\n") {
		if err := p.parseLine(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
		}
	}
	if len(p.stack) != 0 {
		frame := p.stack[len(p.stack)-1]
		return nil, fmt.Errorf("unclosed %s %q", frame.kind, frame.expr)
	}
	return p.root, nil
}

type frame struct {
	kind   string
	expr   string
	blocks []ir.Block
}

type documentParser struct {
	root  []ir.Block
	stack []frame
}

func (p *documentParser) parseLine(line string) error {
	starts, middle, closes, err := splitControlWrappers(line)
	if err != nil {
		return err
	}
	for _, start := range starts {
		p.stack = append(p.stack, frame{kind: start.kind, expr: start.expr})
	}
	if strings.TrimSpace(middle) != "" || len(starts) == 0 && len(closes) == 0 {
		block, err := parseBlock(middle)
		if err != nil {
			return err
		}
		p.append(block)
	}
	for _, close := range closes {
		if err := p.close(close); err != nil {
			return err
		}
	}
	return nil
}

func (p *documentParser) append(block ir.Block) {
	if len(p.stack) == 0 {
		p.root = append(p.root, block)
		return
	}
	top := &p.stack[len(p.stack)-1]
	top.blocks = append(top.blocks, block)
}

func (p *documentParser) close(kind string) error {
	if len(p.stack) == 0 {
		return fmt.Errorf("unexpected closing %s", kind)
	}
	top := p.stack[len(p.stack)-1]
	if top.kind != kind {
		return fmt.Errorf("unexpected closing %s, expected %s", kind, top.kind)
	}
	p.stack = p.stack[:len(p.stack)-1]
	if kind == "if" {
		p.append(ir.IfBlock{Expr: top.expr, Blocks: top.blocks})
		return nil
	}
	p.append(ir.EachBlock{Key: top.expr, Blocks: top.blocks})
	return nil
}

type controlToken struct {
	kind string
	expr string
}

func splitControlWrappers(line string) ([]controlToken, string, []string, error) {
	rest := strings.TrimSpace(line)
	var starts []controlToken
	for {
		token, tail, ok, err := cutControlStart(rest)
		if err != nil {
			return nil, "", nil, err
		}
		if !ok {
			break
		}
		starts = append(starts, token)
		rest = strings.TrimSpace(tail)
	}

	var closes []string
	for {
		next, ok := cutControlEnd(rest)
		if !ok {
			break
		}
		closes = append([]string{next.kind}, closes...)
		rest = strings.TrimSpace(next.expr)
	}
	return starts, rest, closes, nil
}

func cutControlStart(s string) (controlToken, string, bool, error) {
	for _, kind := range []string{"if", "each"} {
		prefix := "{" + kind + ":"
		if !strings.HasPrefix(s, prefix) {
			continue
		}
		end := strings.IndexByte(s, '}')
		if end < 0 {
			return controlToken{}, "", false, fmt.Errorf("unterminated %s directive", kind)
		}
		expr := strings.TrimSpace(s[len(prefix):end])
		if expr == "" {
			return controlToken{}, "", false, fmt.Errorf("empty %s directive", kind)
		}
		return controlToken{kind: kind, expr: expr}, s[end+1:], true, nil
	}
	return controlToken{}, s, false, nil
}

func cutControlEnd(s string) (controlToken, bool) {
	for _, kind := range []string{"if", "each"} {
		suffix := "{/" + kind + "}"
		if strings.HasSuffix(s, suffix) {
			return controlToken{kind: kind, expr: strings.TrimSpace(strings.TrimSuffix(s, suffix))}, true
		}
	}
	return controlToken{}, false
}

type lineStyle struct {
	aligns []ir.Alignment
	widths []int
	font   ir.Font
	bold   bool
}

func parseBlock(line string) (ir.Block, error) {
	style := lineStyle{font: ir.FontNormal}
	rest := strings.TrimSpace(line)
	var err error
	for {
		var changed bool
		style, rest, changed, err = cutStyleDirective(style, rest)
		if err != nil {
			return nil, err
		}
		if !changed {
			break
		}
		rest = strings.TrimSpace(rest)
	}

	switch {
	case rest == "":
		return ir.SpaceBlock{Lines: 1}, nil
	case rest == "---":
		return ir.RuleBlock{}, nil
	case rest == "{cut}":
		return ir.CutBlock{}, nil
	case rest == "{cut:partial}":
		return ir.CutBlock{Partial: true}, nil
	case rest == "{drawer}":
		return ir.DrawerBlock{}, nil
	}
	if payload, ok, err := cutWholeDirective(rest, "s"); ok || err != nil {
		if err != nil {
			return nil, err
		}
		lines, err := strconv.Atoi(strings.TrimSpace(payload))
		if err != nil || lines <= 0 {
			return nil, fmt.Errorf("invalid space count %q", payload)
		}
		return ir.SpaceBlock{Lines: lines}, nil
	}
	if payload, ok, err := cutWholeDirective(rest, "qr"); ok || err != nil {
		if err != nil {
			return nil, err
		}
		return parseQR(payload)
	}
	if payload, ok, err := cutWholeDirective(rest, "barcode"); ok || err != nil {
		if err != nil {
			return nil, err
		}
		return parseBarcode(payload)
	}
	if payload, ok, err := cutWholeDirective(rest, "image"); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(payload) == "" {
			return nil, fmt.Errorf("empty image payload")
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			data = []byte(payload)
		}
		return ir.ImageBlock{Data: data}, nil
	}
	if strings.HasPrefix(rest, "{") && !strings.HasPrefix(rest, "{{") {
		return nil, fmt.Errorf("unknown directive")
	}

	text, bold, err := stripBold(rest)
	if err != nil {
		return nil, err
	}
	style.bold = style.bold || bold
	columns := strings.Split(text, "\t")
	lineOut := ir.TextLine{Columns: make([]ir.Column, len(columns))}
	for i, col := range columns {
		lineOut.Columns[i] = ir.Column{Text: col, Width: widthAt(style.widths, i), Align: alignAt(style.aligns, i)}
	}
	return ir.TextBlock{
		Lines:     []ir.TextLine{lineOut},
		Alignment: alignAt(style.aligns, 0),
		Font:      style.font,
		Bold:      style.bold,
	}, nil
}

func cutStyleDirective(style lineStyle, s string) (lineStyle, string, bool, error) {
	if !strings.HasPrefix(s, "{") || strings.HasPrefix(s, "{{") {
		return style, s, false, nil
	}
	end := strings.IndexByte(s, '}')
	if end < 0 {
		return style, s, false, fmt.Errorf("unterminated directive")
	}
	name, value, ok := strings.Cut(s[1:end], ":")
	if !ok {
		name = s[1:end]
	}
	switch name {
	case "a":
		aligns, err := parseAlignments(value)
		if err != nil {
			return style, s, false, err
		}
		style.aligns = aligns
	case "w":
		widths, err := parseWidths(value)
		if err != nil {
			return style, s, false, err
		}
		style.widths = widths
	case "f":
		font, err := parseFont(value)
		if err != nil {
			return style, s, false, err
		}
		style.font = font
	case "b":
		style.bold = true
	default:
		return style, s, false, nil
	}
	return style, s[end+1:], true, nil
}

func parseAlignments(value string) ([]ir.Alignment, error) {
	parts := strings.Split(value, ",")
	aligns := make([]ir.Alignment, len(parts))
	for i, part := range parts {
		switch strings.TrimSpace(part) {
		case "left":
			aligns[i] = ir.AlignLeft
		case "center":
			aligns[i] = ir.AlignCenter
		case "right":
			aligns[i] = ir.AlignRight
		default:
			return nil, fmt.Errorf("invalid alignment %q", part)
		}
	}
	return aligns, nil
}

func parseWidths(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	widths := make([]int, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "auto" {
			continue
		}
		width, err := strconv.Atoi(part)
		if err != nil || width <= 0 {
			return nil, fmt.Errorf("invalid width %q", part)
		}
		widths[i] = width
	}
	return widths, nil
}

func parseFont(value string) (ir.Font, error) {
	switch strings.TrimSpace(value) {
	case "normal":
		return ir.FontNormal, nil
	case "double":
		return ir.FontDouble, nil
	case "smaller":
		return ir.FontSmaller, nil
	default:
		return "", fmt.Errorf("invalid font %q", value)
	}
}

func cutWholeDirective(s, name string) (string, bool, error) {
	prefix := "{" + name + ":"
	if !strings.HasPrefix(s, prefix) {
		return "", false, nil
	}
	end, err := directiveEnd(s)
	if err != nil {
		return "", false, err
	}
	if end != len(s)-1 {
		return "", false, nil
	}
	return s[len(prefix):end], true, nil
}

func directiveEnd(s string) (int, error) {
	for i := 1; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '{' && s[i+1] == '{' {
			end := strings.Index(s[i+2:], "}}")
			if end < 0 {
				return 0, fmt.Errorf("unterminated template expression")
			}
			i += end + 3
			continue
		}
		if s[i] == '}' {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unterminated directive")
}

func parseBarcode(payload string) (ir.BarcodeBlock, error) {
	typ, data, ok := strings.Cut(payload, ":")
	if !ok {
		return ir.BarcodeBlock{}, fmt.Errorf("invalid barcode directive")
	}
	typ = strings.ToLower(strings.TrimSpace(typ))
	if !barcodeTypes[typ] {
		return ir.BarcodeBlock{}, fmt.Errorf("unsupported barcode type %q", typ)
	}
	data = strings.TrimSpace(data)
	if data == "" {
		return ir.BarcodeBlock{}, fmt.Errorf("empty barcode data")
	}
	return ir.BarcodeBlock{Type: typ, Data: data, HRI: true}, nil
}

func parseQR(payload string) (ir.QRBlock, error) {
	if strings.TrimSpace(payload) == "" {
		return ir.QRBlock{}, fmt.Errorf("empty qr payload")
	}
	if !strings.HasPrefix(payload, "size=") {
		return ir.QRBlock{Payload: payload, Model: 2}, nil
	}
	rawSize, body, ok := strings.Cut(strings.TrimPrefix(payload, "size="), ":")
	if !ok {
		return ir.QRBlock{}, fmt.Errorf("invalid qr size directive")
	}
	size, err := strconv.Atoi(strings.TrimSpace(rawSize))
	if err != nil || size < 1 || size > 8 {
		return ir.QRBlock{}, fmt.Errorf("invalid qr size %q", rawSize)
	}
	if strings.TrimSpace(body) == "" {
		return ir.QRBlock{}, fmt.Errorf("empty qr payload")
	}
	return ir.QRBlock{Payload: body, Size: size, Model: 2}, nil
}

func stripBold(s string) (string, bool, error) {
	if !strings.Contains(s, "**") {
		return s, false, nil
	}
	if strings.Count(s, "**")%2 != 0 {
		return "", false, fmt.Errorf("unbalanced bold marker")
	}
	return strings.ReplaceAll(s, "**", ""), true, nil
}

func alignAt(aligns []ir.Alignment, idx int) ir.Alignment {
	if idx < len(aligns) {
		return aligns[idx]
	}
	if len(aligns) != 0 {
		return aligns[len(aligns)-1]
	}
	return ir.AlignLeft
}

func widthAt(widths []int, idx int) int {
	if idx < len(widths) {
		return widths[idx]
	}
	return 0
}
