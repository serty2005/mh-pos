package engine

import (
	"strconv"
	"strings"
)

// currencySymbols сопоставляет ISO currency code с печатаемым суффиксом.
// Значения выбраны печатно-безопасными для generic ESC/POS CP866 path.
// Для неизвестных кодов FormatMoneyMinor использует сам код как суффикс.
var currencySymbols = map[string]string{
	"RUB": "RUB",
}

// FormatMoneyMinor форматирует денежную сумму из minor units (копеек) в
// человекочитаемую строку вида "500,00 RUB": десятичный разделитель — запятая,
// разряды тысяч разделяются пробелом, далее печатно-безопасный суффикс валюты.
//
// Используется шаблонным движком (POS-73) на этапе рендера; print context при
// этом хранит исходные minor units, поэтому проекция остаётся детерминированной
// и не зависит от локали форматирования.
//
// minor трактуется как сумма в сотых долях основной единицы (2 знака после
// запятой). Отрицательные суммы получают ведущий минус. Если currencyCode не
// известен, в качестве суффикса используется сам код; для пустого кода суффикс
// не добавляется.
func FormatMoneyMinor(minor int64, currencyCode string) string {
	negative := minor < 0
	abs := minor
	if negative {
		abs = -abs
	}

	units := abs / 100
	frac := abs % 100

	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	b.WriteString(groupThousands(units))
	b.WriteByte(',')
	// дробная часть всегда два знака с ведущим нулём
	if frac < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.FormatInt(frac, 10))

	if suffix := currencySuffix(currencyCode); suffix != "" {
		b.WriteByte(' ')
		b.WriteString(suffix)
	}
	return b.String()
}

// currencySuffix возвращает символ валюты, либо сам код, либо пустую строку.
func currencySuffix(currencyCode string) string {
	code := strings.TrimSpace(currencyCode)
	if code == "" {
		return ""
	}
	if symbol, ok := currencySymbols[strings.ToUpper(code)]; ok {
		return symbol
	}
	return code
}

// groupThousands форматирует неотрицательное целое с пробелом между разрядами
// тысяч: 1234567 -> "1 234 567".
func groupThousands(n int64) string {
	digits := strconv.FormatInt(n, 10)
	count := len(digits)
	if count <= 3 {
		return digits
	}

	var b strings.Builder
	// первая группа — остаток от деления длины на 3
	lead := count % 3
	if lead == 0 {
		lead = 3
	}
	b.WriteString(digits[:lead])
	for i := lead; i < count; i += 3 {
		b.WriteByte(' ')
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}
