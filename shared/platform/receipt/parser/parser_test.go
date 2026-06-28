package parser

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mh-pos-platform/receipt/ir"
)

func TestParseLevel1Constructs(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		blocks []ir.Block
	}{
		{
			name:  "text",
			input: "hello",
			blocks: []ir.Block{ir.TextBlock{
				Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "hello", Align: ir.AlignLeft}}}},
				Alignment: ir.AlignLeft,
				Font:      ir.FontNormal,
			}},
		},
		{
			name:  "columns width alignment",
			input: "{w:auto,6,8}{a:left,center,right}Name\tQty\tTotal",
			blocks: []ir.Block{ir.TextBlock{
				Lines: []ir.TextLine{{Columns: []ir.Column{
					{Text: "Name", Align: ir.AlignLeft},
					{Text: "Qty", Width: 6, Align: ir.AlignCenter},
					{Text: "Total", Width: 8, Align: ir.AlignRight},
				}}},
				Alignment: ir.AlignLeft,
				Font:      ir.FontNormal,
			}},
		},
		{
			name:  "font directive",
			input: "{a:center}{f:double}Ticket",
			blocks: []ir.Block{ir.TextBlock{
				Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "Ticket", Align: ir.AlignCenter}}}},
				Alignment: ir.AlignCenter,
				Font:      ir.FontDouble,
			}},
		},
		{
			name:  "bold directive",
			input: "{b}Bold",
			blocks: []ir.Block{ir.TextBlock{
				Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "Bold", Align: ir.AlignLeft}}}},
				Alignment: ir.AlignLeft,
				Font:      ir.FontNormal,
				Bold:      true,
			}},
		},
		{
			name:  "bold marker",
			input: "**Bold**",
			blocks: []ir.Block{ir.TextBlock{
				Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "Bold", Align: ir.AlignLeft}}}},
				Alignment: ir.AlignLeft,
				Font:      ir.FontNormal,
				Bold:      true,
			}},
		},
		{name: "rule", input: "---", blocks: []ir.Block{ir.RuleBlock{}}},
		{name: "empty line", input: "", blocks: []ir.Block{ir.SpaceBlock{Lines: 1}}},
		{name: "space directive", input: "{s:3}", blocks: []ir.Block{ir.SpaceBlock{Lines: 3}}},
		{name: "qr", input: "{qr:{{.qr_payload}}}", blocks: []ir.Block{ir.QRBlock{Payload: "{{.qr_payload}}", Model: 2}}},
		{name: "qr size", input: "{qr:size=4:{{.qr_payload}}}", blocks: []ir.Block{ir.QRBlock{Payload: "{{.qr_payload}}", Size: 4, Model: 2}}},
		{name: "barcode", input: "{barcode:ean13:4601234567890}", blocks: []ir.Block{ir.BarcodeBlock{Type: "ean13", Data: "4601234567890", HRI: true}}},
		{name: "image", input: "{image:aGVsbG8=}", blocks: []ir.Block{ir.ImageBlock{Data: []byte("hello")}}},
		{name: "cut", input: "{cut}", blocks: []ir.Block{ir.CutBlock{}}},
		{name: "partial cut", input: "{cut:partial}", blocks: []ir.Block{ir.CutBlock{Partial: true}}},
		{name: "drawer", input: "{drawer}", blocks: []ir.Block{ir.DrawerBlock{}}},
		{
			name:  "inline if",
			input: "{if:is_copy}{a:center}COPY{/if}",
			blocks: []ir.Block{ir.IfBlock{Expr: "is_copy", Blocks: []ir.Block{ir.TextBlock{
				Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "COPY", Align: ir.AlignCenter}}}},
				Alignment: ir.AlignCenter,
				Font:      ir.FontNormal,
			}}}},
		},
		{
			name: "nested each",
			input: `{each:lines}
{w:auto,6}{a:left,right}{{.name}}	{{.quantity}}
{/each}`,
			blocks: []ir.Block{ir.EachBlock{Key: "lines", Blocks: []ir.Block{ir.TextBlock{
				Lines: []ir.TextLine{{Columns: []ir.Column{
					{Text: "{{.name}}", Align: ir.AlignLeft},
					{Text: "{{.quantity}}", Width: 6, Align: ir.AlignRight},
				}}},
				Alignment: ir.AlignLeft,
				Font:      ir.FontNormal,
			}}}},
		},
		{
			name: "if with nested each",
			input: `{if:modifiers}{each:modifiers}
{a:left}+ {{.name}}
{/each}{/if}`,
			blocks: []ir.Block{ir.IfBlock{Expr: "modifiers", Blocks: []ir.Block{ir.EachBlock{Key: "modifiers", Blocks: []ir.Block{ir.TextBlock{
				Lines:     []ir.TextLine{{Columns: []ir.Column{{Text: "+ {{.name}}", Align: ir.AlignLeft}}}},
				Alignment: ir.AlignLeft,
				Font:      ir.FontNormal,
			}}}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.blocks) {
				t.Fatalf("blocks mismatch\nwant: %#v\n got: %#v", tt.blocks, got)
			}
		})
	}
}

func TestParseFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("fixtures", "precheck_level1.rl"))
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 10 {
		t.Fatalf("unexpected block count: %d", len(blocks))
	}
	if _, ok := blocks[6].(ir.EachBlock); !ok {
		t.Fatalf("expected lines each block at index 6, got %T", blocks[6])
	}
	if _, ok := blocks[8].(ir.IfBlock); !ok {
		t.Fatalf("expected copy if block at index 8, got %T", blocks[8])
	}
}

func TestParseInvalidInputReturnsErrorWithoutPanic(t *testing.T) {
	inputs := []string{
		"{a:sideways}bad",
		"{w:0}bad",
		"{f:huge}bad",
		"{qr:}",
		"{barcode:unknown:123}",
		"{if:copy}",
		"{/if}",
		"{if:a}{/each}",
		"**bad",
		"{s:nope}",
		"{image:}",
		"{qr:size=9:bad}",
		"{unknown}",
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("panic: %v", recovered)
				}
			}()
			if _, err := Parse(input); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
