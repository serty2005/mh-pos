package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mh-pos-platform/receipt/escpos"
	"mh-pos-platform/receipt/ir"
	"mh-pos-platform/receipt/layout"
)

func loadTemplate(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("templates", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestRenderDefaultPrecheckFixture(t *testing.T) {
	var snapshot PrecheckSnapshot
	loadSnapshot(t, "precheck_snapshot.json", &snapshot)

	blocks, err := Render(loadTemplate(t, "default_precheck.rl"), ProjectPrecheck(snapshot))
	if err != nil {
		t.Fatal(err)
	}

	assertNoControlBlocks(t, blocks)
	assertTextBlock(t, blocks[1], "Exhibition TechWorld 2026", ir.AlignCenter)
	assertTextBlock(t, blocks[4], "RECEIPT", ir.AlignCenter)
	assertColumns(t, blocks[9], []ir.Column{
		{Text: "Standard Ticket", Align: ir.AlignLeft},
		{Text: "1", Width: 6, Align: ir.AlignRight},
		{Text: "600,00 RUB", Width: 12, Align: ir.AlignRight},
	})
	assertTextBlock(t, blocks[10], "+ VIP Zone: 100,00 RUB", ir.AlignLeft)
	assertColumns(t, blocks[11], []ir.Column{
		{Text: "Latte Coffee", Align: ir.AlignLeft},
		{Text: "2", Width: 6, Align: ir.AlignRight},
		{Text: "700,00 RUB", Width: 12, Align: ir.AlignRight},
	})
	assertColumns(t, blocks[14], []ir.Column{
		{Text: "Discount:", Align: ir.AlignLeft},
		{Text: "-50,00 RUB", Width: 16, Align: ir.AlignRight},
	})
	assertColumns(t, blocks[15], []ir.Column{
		{Text: "VAT 20%:", Align: ir.AlignLeft},
		{Text: "208,33 RUB", Width: 16, Align: ir.AlignRight},
	})
	assertCutHasFeed(t, blocks)
	assertRenderableAtCPL(t, blocks, 32, 48)
}

func TestRenderDefaultTicketFixture(t *testing.T) {
	var snapshot TicketSnapshot
	loadSnapshot(t, "ticket_snapshot.json", &snapshot)

	blocks, err := Render(loadTemplate(t, "default_ticket.rl"), ProjectTicket(snapshot))
	if err != nil {
		t.Fatal(err)
	}

	assertNoControlBlocks(t, blocks)
	assertTextBlock(t, blocks[0], "Exhibition TechWorld 2026", ir.AlignCenter)
	assertTextBlock(t, blocks[3], "TICKET", ir.AlignCenter)
	assertTextBlockFont(t, blocks[3], ir.FontDouble)
	assertTextBlock(t, blocks[6], "Standard Ticket", ir.AlignCenter)
	assertTextBlockFont(t, blocks[6], ir.FontDouble)
	if got, ok := blocks[9].(ir.QRBlock); !ok {
		t.Fatalf("expected ticket QR block at index 9, got %T", blocks[9])
	} else if got.Payload != "MHT1:019044ab-0000-7000-0000-000000000001" || got.Size != 6 || got.Model != 2 {
		t.Fatalf("unexpected QR block: %#v", got)
	}
	assertTextBlock(t, blocks[12], "Amount: 500,00 RUB", ir.AlignCenter)
	assertCutHasFeed(t, blocks)
	assertRenderableAtCPL(t, blocks, 32, 48)
}

func assertTextBlockFont(t *testing.T, block ir.Block, font ir.Font) {
	t.Helper()
	text, ok := block.(ir.TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", block)
	}
	if text.Font != font {
		t.Fatalf("font mismatch: want %q, got %q", font, text.Font)
	}
}

func TestRenderControlFlowAndMoneyEdgeCases(t *testing.T) {
	ctx := PrecheckPrintContext{
		RestaurantName: "Demo",
		CurrencyCode:   "RUB",
		Lines: []PrecheckLine{{
			Name:       "No modifiers",
			Quantity:   1,
			TotalMinor: 100,
			Modifiers:  []PrecheckModifier{},
		}},
	}
	template := `{a:center}{{.RestaurantName}}
{if:discount_total_minor}discount hidden{/if}
{each:lines}
{{.name}} {{.total_minor | money}}
{if:modifiers}modifier hidden{/if}
{/each}`

	blocks, err := Render(template, ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected only visible root and line blocks, got %d: %#v", len(blocks), blocks)
	}
	assertTextBlock(t, blocks[0], "Demo", ir.AlignCenter)
	assertTextBlock(t, blocks[1], "No modifiers 1,00 RUB", ir.AlignLeft)
}

func TestRenderRejectsUnsupportedPipe(t *testing.T) {
	_, err := Render("{{.total_minor | unknown}}", PrecheckPrintContext{TotalMinor: 100})
	if err == nil {
		t.Fatal("expected unsupported pipe error")
	}
}

func assertNoControlBlocks(t *testing.T, blocks []ir.Block) {
	t.Helper()
	for i, block := range blocks {
		switch block.(type) {
		case ir.IfBlock, ir.EachBlock:
			t.Fatalf("control block leaked at index %d: %T", i, block)
		}
	}
}

func assertTextBlock(t *testing.T, block ir.Block, text string, align ir.Alignment) {
	t.Helper()
	got, ok := block.(ir.TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", block)
	}
	if len(got.Lines) != 1 || len(got.Lines[0].Columns) != 1 {
		t.Fatalf("expected single-column text block, got %#v", got)
	}
	if got.Lines[0].Columns[0].Text != text || got.Lines[0].Columns[0].Align != align {
		t.Fatalf("text block mismatch: %#v", got.Lines[0].Columns[0])
	}
}

func assertColumns(t *testing.T, block ir.Block, want []ir.Column) {
	t.Helper()
	got, ok := block.(ir.TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", block)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("expected one line, got %#v", got.Lines)
	}
	if !reflect.DeepEqual(got.Lines[0].Columns, want) {
		t.Fatalf("columns mismatch\nwant: %#v\n got: %#v", want, got.Lines[0].Columns)
	}
}

func assertCutHasFeed(t *testing.T, blocks []ir.Block) {
	t.Helper()
	if len(blocks) < 2 {
		t.Fatalf("expected feed and cut blocks, got %#v", blocks)
	}
	if _, ok := blocks[len(blocks)-1].(ir.CutBlock); !ok {
		t.Fatalf("expected final CutBlock, got %T", blocks[len(blocks)-1])
	}
	feed, ok := blocks[len(blocks)-2].(ir.SpaceBlock)
	if !ok || feed.Lines < 3 {
		t.Fatalf("expected at least 3 feed lines before cut, got %#v", blocks[len(blocks)-2])
	}
}

func assertRenderableAtCPL(t *testing.T, blocks []ir.Block, cpls ...int) {
	t.Helper()
	for _, cpl := range cpls {
		for _, block := range blocks {
			text, ok := block.(ir.TextBlock)
			if !ok {
				continue
			}
			for _, line := range text.Lines {
				textCPL := layout.TextCPL(cpl, text.Font)
				rendered := layout.RenderColumns(line.Columns, textCPL)
				if len([]rune(rendered)) > textCPL {
					t.Fatalf("line exceeds CPL %d for font %q: %q", textCPL, text.Font, rendered)
				}
			}
		}
		if _, err := escpos.Render(blocks, escpos.RenderOptions{CPL: cpl}); err != nil {
			t.Fatalf("ESC/POS render failed at CPL %d: %v", cpl, err)
		}
	}
}
