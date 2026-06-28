package ir

import "testing"

func TestBlocksImplementBlock(t *testing.T) {
	blocks := []Block{
		TextBlock{},
		RuleBlock{},
		SpaceBlock{},
		QRBlock{},
		BarcodeBlock{},
		ImageBlock{},
		CutBlock{},
		DrawerBlock{},
		IfBlock{},
		EachBlock{},
	}
	if len(blocks) != 10 {
		t.Fatalf("unexpected block count: %d", len(blocks))
	}
}
