package svg

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"mh-pos-platform/receipt/escpos"
	"mh-pos-platform/receipt/ir"
	"mh-pos-platform/receipt/layout"
	"mh-pos-platform/receipt/parser"
)

func TestRenderBasics(t *testing.T) {
	blocks := []ir.Block{
		ir.TextBlock{Lines: []ir.TextLine{{Columns: []ir.Column{{Text: "Привет", Align: ir.AlignCenter}}}}, Alignment: ir.AlignCenter, Font: ir.FontDouble, Bold: true},
		ir.RuleBlock{},
		ir.SpaceBlock{Lines: 1},
		ir.QRBlock{Payload: "MHT1:019", Size: 4, Model: 2},
		ir.BarcodeBlock{Type: "ean13", Data: "4601234567890", HRI: true},
		ir.ImageBlock{Width: 64, Height: 16, Data: []byte{0xff}},
		ir.CutBlock{Partial: true},
	}
	got, err := Render(blocks, RenderOptions{CPL: 32})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<svg `, `Привет`, `data-block="qr"`, `data-size="4"`, `data-block="barcode"`, `data-block="image"`, `data-block="cut"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestRenderCPL32And48(t *testing.T) {
	for _, cpl := range []int{32, 48} {
		got, err := Render([]ir.Block{ir.RuleBlock{}}, RenderOptions{CPL: cpl})
		if err != nil {
			t.Fatal(err)
		}
		rule := strings.Repeat("-", cpl)
		if !strings.Contains(got, ">"+rule+"<") {
			t.Fatalf("cpl %d: missing rule %q in %s", cpl, rule, got)
		}
		if !strings.Contains(got, `width="`+strconv.Itoa(24+cpl*8)+`"`) {
			t.Fatalf("svg width does not reflect cpl %d: %s", cpl, got)
		}
	}
}

func TestRenderQRSizeAffectsPreview(t *testing.T) {
	small, err := Render([]ir.Block{ir.QRBlock{Payload: "x", Size: 2}}, RenderOptions{CPL: 48})
	if err != nil {
		t.Fatal(err)
	}
	large, err := Render([]ir.Block{ir.QRBlock{Payload: "x", Size: 6}}, RenderOptions{CPL: 48})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(small, `width="42"`) || !strings.Contains(large, `width="126"`) {
		t.Fatalf("qr sizes not reflected\nsmall=%s\nlarge=%s", small, large)
	}
}

func TestLayoutEqualityWithEscposFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "escpos", "fixtures", "basic.rl"))
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := parser.Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := escpos.Render(blocks, escpos.RenderOptions{CPL: 48, Codepage: escpos.CodepageCP866}); err != nil {
		t.Fatal(err)
	}
	text := blocks[2].(ir.TextBlock)
	want := layout.RenderColumns(text.Lines[0].Columns, 48)
	gotSVG, err := Render([]ir.Block{text}, RenderOptions{CPL: 48})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotSVG, want) {
		t.Fatalf("svg did not use shared column layout %q in %s", want, gotSVG)
	}
	gotEsc, err := escpos.Render([]ir.Block{text}, escpos.RenderOptions{CPL: 48, Codepage: escpos.CodepageCP866})
	if err != nil {
		t.Fatal(err)
	}
	rawText, _ := escpos.EncodeCP866(want)
	if !bytes.Contains(gotEsc, rawText) {
		t.Fatalf("escpos did not use shared column layout %q in % x", want, gotEsc)
	}
}
