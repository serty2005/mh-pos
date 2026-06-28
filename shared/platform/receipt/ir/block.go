// Package ir описывает width-independent модель печатного документа.
package ir

// Alignment задает выравнивание текста в строке или колонке.
type Alignment string

const (
	AlignLeft   Alignment = "left"
	AlignCenter Alignment = "center"
	AlignRight  Alignment = "right"
)

// Font задает базовый размер текста для блока.
type Font string

const (
	FontNormal  Font = "normal"
	FontDouble  Font = "double"
	FontSmaller Font = "smaller"
)

// Block является одним элементом документа; рендерер применяет целевой CPL позже.
type Block interface {
	blockMarker()
}

// TextBlock хранит одну или несколько текстовых строк с общим стилем.
type TextBlock struct {
	Lines     []TextLine
	Alignment Alignment
	Font      Font
	Bold      bool
}

func (TextBlock) blockMarker() {}

// TextLine хранит колонки одной логической строки.
type TextLine struct {
	Columns []Column
}

// Column хранит текст, width hint и выравнивание отдельной колонки.
type Column struct {
	Text  string
	Width int
	Align Alignment
}

// RuleBlock означает горизонтальную черту на всю ширину ленты.
type RuleBlock struct{}

func (RuleBlock) blockMarker() {}

// SpaceBlock означает N пустых строк.
type SpaceBlock struct {
	Lines int
}

func (SpaceBlock) blockMarker() {}

// QRBlock хранит payload QR-кода без привязки к конкретным ESC/POS-командам.
type QRBlock struct {
	Payload string
	Size    int
	Model   int
}

func (QRBlock) blockMarker() {}

// BarcodeBlock хранит barcode payload и тип, поддерживаемый Level 1.
type BarcodeBlock struct {
	Type string
	Data string
	HRI  bool
}

func (BarcodeBlock) blockMarker() {}

// ImageBlock хранит растровые байты изображения; размер может быть заполнен позже.
type ImageBlock struct {
	Data   []byte
	Width  int
	Height int
}

func (ImageBlock) blockMarker() {}

// CutBlock означает полный или частичный рез бумаги.
type CutBlock struct {
	Partial bool
}

func (CutBlock) blockMarker() {}

// DrawerBlock означает импульс на кассовый ящик.
type DrawerBlock struct{}

func (DrawerBlock) blockMarker() {}

// IfBlock сохраняет Level 1 условие до стадии template engine.
type IfBlock struct {
	Expr   string
	Blocks []Block
}

func (IfBlock) blockMarker() {}

// EachBlock сохраняет Level 1 итерацию до стадии template engine.
type EachBlock struct {
	Key    string
	Blocks []Block
}

func (EachBlock) blockMarker() {}
