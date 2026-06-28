package escpos

import "fmt"

// EncodeCP437 кодирует ASCII-текст в CP437 (PC437, USA/Standard Europe).
// Поддерживает только ASCII-range 0x20-0x7E и управляющие символы \n, \r, \t.
// Типографские суррогаты (смарт-кавычки, тире, ₽) нормализуются так же, как в EncodeCP866.
// Для символов вне диапазона возвращается ошибка.
func EncodeCP437(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if replacement, ok := cp866Replacement(r); ok {
			out = append(out, replacement...)
			continue
		}
		b, ok := encodeRuneCP437(r)
		if !ok {
			return nil, fmt.Errorf("cp437: unsupported rune %q", r)
		}
		out = append(out, b)
	}
	return out, nil
}

// CanEncodeCP437 возвращает false для символов вне CP437-диапазона.
func CanEncodeCP437(s string) bool {
	for _, r := range s {
		if _, ok := cp866Replacement(r); ok {
			continue
		}
		if _, ok := encodeRuneCP437(r); !ok {
			return false
		}
	}
	return true
}

func encodeRuneCP437(r rune) (byte, bool) {
	switch {
	case r == '\n' || r == '\r' || r == '\t':
		return byte(r), true
	case r >= 0x20 && r <= 0x7e:
		return byte(r), true
	default:
		return 0, false
	}
}
