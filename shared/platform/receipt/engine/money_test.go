package engine

import "testing"

func TestFormatMoneyMinor(t *testing.T) {
	tests := []struct {
		name     string
		minor    int64
		currency string
		want     string
	}{
		{name: "целые рубли", minor: 50000, currency: "RUB", want: "500,00 RUB"},
		{name: "копейки", minor: 12345, currency: "RUB", want: "123,45 RUB"},
		{name: "ведущий ноль в дробной части", minor: 50005, currency: "RUB", want: "500,05 RUB"},
		{name: "разряды тысяч", minor: 123456789, currency: "RUB", want: "1 234 567,89 RUB"},
		{name: "ровно тысяча", minor: 100000, currency: "RUB", want: "1 000,00 RUB"},
		{name: "ноль", minor: 0, currency: "RUB", want: "0,00 RUB"},
		{name: "меньше рубля", minor: 7, currency: "RUB", want: "0,07 RUB"},
		{name: "отрицательная сумма", minor: -50000, currency: "RUB", want: "-500,00 RUB"},
		{name: "неизвестная валюта — код как суффикс", minor: 50000, currency: "USD", want: "500,00 USD"},
		{name: "code lower-case нормализуется", minor: 50000, currency: "rub", want: "500,00 RUB"},
		{name: "пустой код — без суффикса", minor: 50000, currency: "", want: "500,00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatMoneyMinor(tt.minor, tt.currency); got != tt.want {
				t.Fatalf("FormatMoneyMinor(%d, %q) = %q, want %q", tt.minor, tt.currency, got, tt.want)
			}
		})
	}
}
