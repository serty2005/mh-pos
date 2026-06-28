package escpos

import (
	"bytes"
	"fmt"
	"strings"

	"mh-pos-platform/receipt/ir"
	"mh-pos-platform/receipt/layout"
)

const (
	paperCutFull    = "full"
	paperCutPartial = "partial"
	paperCutNone    = "none"
)

// CodepageCP437 задаёт кодовую страницу PC437 (USA/Standard Europe) — default для ESC/POS.
const CodepageCP437 = "cp437"

// CodepageCP866 задаёт кодовую страницу PC866 (Cyrillic) — для русскоязычных развёртываний.
const CodepageCP866 = "cp866"

// RenderOptions задают физические ограничения ESC/POS-рендера.
// Codepage: "" или "cp437" = CP437 (default, ESC t 0); "cp866" = CP866 (ESC t 17).
type RenderOptions struct {
	CPL          int
	RasterOnly   bool
	PaperCutType string
	Codepage     string
}

// Render преобразует IR в ESC/POS-команды для печати raw-принтером.
func Render(blocks []ir.Block, opts RenderOptions) ([]byte, error) {
	r := renderer{
		cpl:          layout.NormalizeCPL(opts.CPL),
		paperCutType: normalizeCut(opts.PaperCutType),
		rasterOnly:   opts.RasterOnly,
		codepage:     normalizeCodepage(opts.Codepage),
	}
	return r.render(blocks)
}

type renderer struct {
	buf          bytes.Buffer
	cpl          int
	paperCutType string
	rasterOnly   bool
	codepage     string
}

func normalizeCodepage(cp string) string {
	if strings.TrimSpace(strings.ToLower(cp)) == CodepageCP866 {
		return CodepageCP866
	}
	return CodepageCP437
}

func (r *renderer) encodeText(text string) ([]byte, error) {
	if r.codepage == CodepageCP866 {
		return EncodeCP866(text)
	}
	return EncodeCP437(text)
}

func (r *renderer) render(blocks []ir.Block) ([]byte, error) {
	r.writeBytes(0x1b, '@')
	if r.codepage == CodepageCP866 {
		r.writeBytes(0x1b, 't', 17) // PC866 Cyrillic
	} else {
		r.writeBytes(0x1b, 't', 0) // PC437 USA/Standard Europe (default)
	}
	if err := r.renderBlocks(blocks); err != nil {
		return nil, err
	}
	return r.buf.Bytes(), nil
}

func (r *renderer) renderBlocks(blocks []ir.Block) error {
	for _, block := range blocks {
		switch b := block.(type) {
		case ir.TextBlock:
			if err := r.text(b); err != nil {
				return err
			}
		case ir.RuleBlock:
			if err := r.line(strings.Repeat("-", r.cpl), ir.AlignLeft, ir.FontNormal, false); err != nil {
				return err
			}
		case ir.SpaceBlock:
			for range max(b.Lines, 1) {
				r.writeBytes('\n')
			}
		case ir.QRBlock:
			if err := r.qr(b); err != nil {
				return err
			}
		case ir.BarcodeBlock:
			if err := r.barcode(b); err != nil {
				return err
			}
		case ir.ImageBlock:
			if err := r.image(b); err != nil {
				return err
			}
		case ir.CutBlock:
			r.cut(b)
		case ir.DrawerBlock:
			r.writeBytes(0x1b, 'p', 0, 25, 250)
		case ir.IfBlock:
			if err := r.renderBlocks(b.Blocks); err != nil {
				return err
			}
		case ir.EachBlock:
			if err := r.renderBlocks(b.Blocks); err != nil {
				return err
			}
		default:
			return fmt.Errorf("escpos: unsupported block %T", block)
		}
	}
	return nil
}

func (r *renderer) text(block ir.TextBlock) error {
	cpl := layout.TextCPL(r.cpl, block.Font)
	for _, line := range block.Lines {
		rendered := layout.RenderColumns(line.Columns, cpl)
		if len(line.Columns) == 1 {
			rendered = layout.Align(rendered, cpl, block.Alignment)
		}
		if err := r.line(rendered, block.Alignment, block.Font, block.Bold); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) line(text string, alignMode ir.Alignment, font ir.Font, bold bool) error {
	if r.rasterOnly {
		return fmt.Errorf("escpos: raster-only text rendering is outside POS-69 primitive scope")
	}
	raw, err := r.encodeText(text)
	if err != nil {
		return err
	}
	r.writeBytes(0x1b, 'a', escposAlign(alignMode))
	r.writeBytes(0x1b, 'E', boolByte(bold))
	r.writeBytes(0x1d, '!', fontSize(font))
	r.buf.Write(raw)
	r.writeBytes('\n', 0x1d, '!', 0, 0x1b, 'E', 0, 0x1b, 'a', 0)
	return nil
}

func (r *renderer) qr(block ir.QRBlock) error {
	payload, err := r.encodeText(block.Payload)
	if err != nil {
		return err
	}
	size := block.Size
	if size <= 0 {
		size = 6
	}
	size = min(max(size, 1), 8)
	model := byte(50)
	if block.Model == 1 {
		model = 49
	}
	r.writeBytes(0x1b, 'a', 1)
	r.writeBytes(0x1d, '(', 'k', 4, 0, 49, 65, model, 0)
	r.writeBytes(0x1d, '(', 'k', 3, 0, 49, 67, byte(size))
	r.writeBytes(0x1d, '(', 'k', 3, 0, 49, 69, 48)
	r.writeStoreQR(payload)
	r.writeBytes(0x1d, '(', 'k', 3, 0, 49, 81, 48)
	r.writeBytes(0x1b, 'a', 0)
	return nil
}

func (r *renderer) writeStoreQR(payload []byte) {
	length := len(payload) + 3
	r.writeBytes(0x1d, '(', 'k', byte(length%256), byte(length/256), 49, 80, 48)
	r.buf.Write(payload)
}

func (r *renderer) barcode(block ir.BarcodeBlock) error {
	typ, ok := barcodeType(block.Type)
	if !ok {
		return fmt.Errorf("escpos: unsupported barcode type %q", block.Type)
	}
	data, err := r.encodeText(block.Data)
	if err != nil {
		return err
	}
	if len(data) > 255 {
		return fmt.Errorf("escpos: barcode data too long")
	}
	r.writeBytes(0x1d, 'H', boolHRI(block.HRI), 0x1d, 'h', 80, 0x1d, 'k', typ, byte(len(data)))
	r.buf.Write(data)
	r.writeBytes('\n')
	return nil
}

func (r *renderer) image(block ir.ImageBlock) error {
	if block.Width <= 0 || block.Height <= 0 {
		return fmt.Errorf("escpos: image width and height are required")
	}
	if block.Width%8 != 0 {
		return fmt.Errorf("escpos: image width must be aligned to 8 pixels")
	}
	rowBytes := block.Width / 8
	want := rowBytes * block.Height
	if len(block.Data) != want {
		return fmt.Errorf("escpos: image data size mismatch")
	}
	r.writeBytes(0x1d, 'v', '0', 0, byte(rowBytes%256), byte(rowBytes/256), byte(block.Height%256), byte(block.Height/256))
	r.buf.Write(block.Data)
	r.writeBytes('\n')
	return nil
}

func (r *renderer) cut(block ir.CutBlock) {
	if r.paperCutType == paperCutNone {
		return
	}
	partial := block.Partial || r.paperCutType == paperCutPartial
	mode := byte(0)
	if partial {
		mode = 1
	}
	r.writeBytes(0x1d, 'V', mode)
}

func (r *renderer) writeBytes(values ...byte) {
	r.buf.Write(values)
}

func escposAlign(a ir.Alignment) byte {
	switch a {
	case ir.AlignCenter:
		return 1
	case ir.AlignRight:
		return 2
	default:
		return 0
	}
}

func fontSize(font ir.Font) byte {
	switch font {
	case ir.FontDouble:
		return 0x11
	case ir.FontSmaller:
		return 0x01
	default:
		return 0
	}
}

func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}

func boolHRI(v bool) byte {
	if v {
		return 2
	}
	return 0
}

func barcodeType(typ string) (byte, bool) {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "upca":
		return 65, true
	case "upce":
		return 66, true
	case "ean13":
		return 67, true
	case "ean8":
		return 68, true
	case "code39":
		return 69, true
	case "itf":
		return 70, true
	case "codabar":
		return 71, true
	default:
		return 0, false
	}
}

func normalizeCut(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case paperCutPartial:
		return paperCutPartial
	case paperCutNone:
		return paperCutNone
	default:
		return paperCutFull
	}
}
