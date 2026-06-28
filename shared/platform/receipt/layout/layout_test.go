package layout

import (
	"testing"

	"mh-pos-platform/receipt/ir"
)

func TestRenderColumnsCPL32And48(t *testing.T) {
	columns := []ir.Column{
		{Text: "Name", Align: ir.AlignLeft},
		{Text: "Qty", Width: 6, Align: ir.AlignRight},
		{Text: "Total", Width: 8, Align: ir.AlignRight},
	}
	tests := []struct {
		cpl  int
		want string
	}{
		{32, "Name                 Qty   Total"},
		{48, "Name                                 Qty   Total"},
	}
	for _, tt := range tests {
		if got := RenderColumns(columns, tt.cpl); got != tt.want {
			t.Fatalf("cpl %d\nwant %q\n got %q", tt.cpl, tt.want, got)
		}
	}
}

func TestRenderColumnsFitsUTF8(t *testing.T) {
	got := RenderColumns([]ir.Column{{Text: "Привет", Width: 3}}, 32)
	if got != "При" {
		t.Fatalf("unexpected fit: %q", got)
	}
}

func TestTextCPLDoubleFont(t *testing.T) {
	if got := TextCPL(48, ir.FontDouble); got != 24 {
		t.Fatalf("double font CPL mismatch: got %d", got)
	}
	if got := TextCPL(48, ir.FontNormal); got != 48 {
		t.Fatalf("normal font CPL mismatch: got %d", got)
	}
}
