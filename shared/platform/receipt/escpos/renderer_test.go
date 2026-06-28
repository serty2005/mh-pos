package escpos

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"mh-pos-platform/receipt/ir"
	"mh-pos-platform/receipt/parser"
)

// TestRenderBasicsCP437 проверяет CP437 (default) рендер с ASCII-текстом и ESC/POS командами.
func TestRenderBasicsCP437(t *testing.T) {
	blocks := []ir.Block{
		ir.TextBlock{
			Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "Hello", Align: ir.AlignCenter}}}},
			Alignment: ir.AlignCenter,
			Font:      ir.FontDouble,
			Bold:      true,
		},
		ir.RuleBlock{},
		ir.SpaceBlock{Lines: 1},
		ir.QRBlock{Payload: "MHT1:019", Size: 4, Model: 2},
		ir.BarcodeBlock{Type: "ean13", Data: "4601234567890", HRI: true},
		ir.ImageBlock{Width: 8, Height: 1, Data: []byte{0xff}},
		ir.IfBlock{Expr: "is_copy", Blocks: []ir.Block{ir.TextBlock{
			Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "COPY", Align: ir.AlignLeft}}}},
			Alignment: ir.AlignLeft,
			Font:      ir.FontNormal,
		}}},
		ir.EachBlock{Key: "lines", Blocks: []ir.Block{ir.TextBlock{
			Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "Line", Align: ir.AlignLeft}}}},
			Alignment: ir.AlignLeft,
			Font:      ir.FontNormal,
		}}},
		ir.DrawerBlock{},
		ir.CutBlock{Partial: true},
	}
	// Codepage не задан — ожидается CP437 по умолчанию (ESC t 0).
	got, err := Render(blocks, RenderOptions{CPL: 32})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte{0x1b, '@', 0x1b, 't', 0}) {
		t.Fatalf("expected CP437 init (ESC t 0): % x", got[:5])
	}
	if !bytes.Contains(got, []byte("Hello")) {
		t.Fatal("rendered output does not contain ASCII text")
	}
	for _, marker := range [][]byte{
		{0x1d, '(', 'k'},
		{0x1d, 'k', 67, 13},
		{0x1d, 'v', '0'},
		{0x1b, 'p', 0, 25, 250},
		{0x1d, 'V', 1},
	} {
		if !bytes.Contains(got, marker) {
			t.Fatalf("missing marker % x", marker)
		}
	}
}

// TestRenderBasicsCP866 проверяет CP866 рендер при явном указании Codepage:"cp866".
func TestRenderBasicsCP866(t *testing.T) {
	blocks := []ir.Block{
		ir.TextBlock{
			Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "Привет", Align: ir.AlignCenter}}}},
			Alignment: ir.AlignCenter,
			Font:      ir.FontDouble,
			Bold:      true,
		},
		ir.CutBlock{Partial: true},
	}
	got, err := Render(blocks, RenderOptions{CPL: 32, Codepage: "cp866"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte{0x1b, '@', 0x1b, 't', 17}) {
		t.Fatalf("expected CP866 init (ESC t 17): % x", got[:5])
	}
	hello, _ := EncodeCP866("Привет")
	if !bytes.Contains(got, hello) {
		t.Fatal("rendered output does not contain CP866 text")
	}
}

// TestRenderCP437RejectsCyrillic проверяет, что кириллица не кодируется в CP437.
func TestRenderCP437RejectsCyrillic(t *testing.T) {
	blocks := []ir.Block{
		ir.TextBlock{
			Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "Привет", Align: ir.AlignLeft}}}},
			Alignment: ir.AlignLeft,
			Font:      ir.FontNormal,
		},
	}
	_, err := Render(blocks, RenderOptions{CPL: 32}) // CP437 default
	if err == nil {
		t.Fatal("expected error for Cyrillic text with CP437 codepage")
	}
}

func TestRenderCPL32And48(t *testing.T) {
	for _, cpl := range []int{32, 48} {
		got, err := Render([]ir.Block{ir.RuleBlock{}}, RenderOptions{CPL: cpl})
		if err != nil {
			t.Fatal(err)
		}
		if count := bytes.Count(got, []byte("-")); count != cpl {
			t.Fatalf("cpl %d: got %d rule chars", cpl, count)
		}
	}
}

// TestRenderParserFixture использует CP866, так как basic.rl содержит кириллицу.
func TestRenderParserFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("fixtures", "basic.rl"))
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := parser.Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Render(blocks, RenderOptions{CPL: 48, Codepage: "cp866"}); err != nil {
		t.Fatal(err)
	}
}

func TestRenderRejectsRasterOnlyTextPrimitive(t *testing.T) {
	_, err := Render([]ir.Block{ir.TextBlock{
		Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "text"}}}},
		Alignment: ir.AlignLeft,
		Font:      ir.FontNormal,
	}}, RenderOptions{RasterOnly: true})
	if err == nil {
		t.Fatal("expected raster-only text error")
	}
}
