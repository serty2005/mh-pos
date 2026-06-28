// Package escpos рендерит receipt IR в ESC/POS-команды и пишет их в raw-принтер.
package escpos

import (
	"fmt"
	"strings"
)

// EncodeCP866 кодирует ASCII и русскую кириллицу в PC866.
func EncodeCP866(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if replacement, ok := cp866Replacement(r); ok {
			out = append(out, replacement...)
			continue
		}
		b, ok := encodeRuneCP866(r)
		if !ok {
			return nil, fmt.Errorf("cp866: unsupported rune %q", r)
		}
		out = append(out, b)
	}
	return out, nil
}

// DecodeCP866 декодирует базовый PC866-набор, нужный для русских чеков.
func DecodeCP866(raw []byte) string {
	var b strings.Builder
	for _, c := range raw {
		b.WriteRune(decodeByteCP866(c))
	}
	return b.String()
}

// CanEncodeCP866 возвращает false для символов, которым нужен raster fallback.
func CanEncodeCP866(s string) bool {
	for _, r := range s {
		if _, ok := cp866Replacement(r); ok {
			continue
		}
		if _, ok := encodeRuneCP866(r); !ok {
			return false
		}
	}
	return true
}

func cp866Replacement(r rune) ([]byte, bool) {
	switch r {
	case '₽':
		return []byte("RUB"), true
	case '«', '»', '“', '”', '„':
		return []byte{'"'}, true
	case '‘', '’':
		return []byte{'\''}, true
	case '—', '–', '−':
		return []byte{'-'}, true
	case '…':
		return []byte("..."), true
	case '×':
		return []byte{'x'}, true
	case '\u00a0':
		return []byte{' '}, true
	default:
		return nil, false
	}
}

func encodeRuneCP866(r rune) (byte, bool) {
	switch {
	case r == '\n' || r == '\r' || r == '\t':
		return byte(r), true
	case r >= 0x20 && r <= 0x7e:
		return byte(r), true
	case r >= 'А' && r <= 'Я':
		return byte(0x80 + r - 'А'), true
	case r >= 'а' && r <= 'п':
		return byte(0xa0 + r - 'а'), true
	case r >= 'р' && r <= 'я':
		return byte(0xe0 + r - 'р'), true
	case r == 'Ё':
		return 0xf0, true
	case r == 'ё':
		return 0xf1, true
	case r == '№':
		return 0xfc, true
	default:
		return 0, false
	}
}

func decodeByteCP866(c byte) rune {
	switch {
	case c == '\n' || c == '\r' || c == '\t':
		return rune(c)
	case c >= 0x20 && c <= 0x7e:
		return rune(c)
	case c >= 0x80 && c <= 0x9f:
		return 'А' + rune(c-0x80)
	case c >= 0xa0 && c <= 0xaf:
		return 'а' + rune(c-0xa0)
	case c >= 0xe0 && c <= 0xef:
		return 'р' + rune(c-0xe0)
	case c == 0xf0:
		return 'Ё'
	case c == 0xf1:
		return 'ё'
	case c == 0xfc:
		return '№'
	default:
		return '\uFFFD'
	}
}
