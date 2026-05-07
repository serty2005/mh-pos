package shared

import (
	"fmt"
	"strings"
)

// CurrencyProfile defines canonical pilot currency metadata used by validation and UI-facing formatting policy.
type CurrencyProfile struct {
	NumericCode           int
	AlphaCode             string
	MinorUnit             int
	CurrencyIsoName       string
	CurrencySymbol        string
	CurrBasicName         string
	CurrAddName           string
	ShowAdd               bool
	ShowCurrencyBasicName bool
}

var currencyProfilesByAlpha = map[string]CurrencyProfile{
	"RUB": {
		NumericCode:           643,
		AlphaCode:             "RUB",
		MinorUnit:             2,
		CurrencyIsoName:       "Russian Ruble",
		CurrencySymbol:        "₽",
		CurrBasicName:         "р",
		CurrAddName:           "коп.",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"USD": {
		NumericCode:           840,
		AlphaCode:             "USD",
		MinorUnit:             2,
		CurrencyIsoName:       "US Dollar",
		CurrencySymbol:        "$",
		CurrBasicName:         "$",
		CurrAddName:           "¢",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"EUR": {
		NumericCode:           978,
		AlphaCode:             "EUR",
		MinorUnit:             2,
		CurrencyIsoName:       "Euro",
		CurrencySymbol:        "€",
		CurrBasicName:         "€",
		CurrAddName:           "c",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"KZT": {
		NumericCode:           398,
		AlphaCode:             "KZT",
		MinorUnit:             2,
		CurrencyIsoName:       "Kazakhstani Tenge",
		CurrencySymbol:        "₸",
		CurrBasicName:         "₸",
		CurrAddName:           "тиын",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"BYN": {
		NumericCode:           933,
		AlphaCode:             "BYN",
		MinorUnit:             2,
		CurrencyIsoName:       "Belarusian Ruble",
		CurrencySymbol:        "Br",
		CurrBasicName:         "Br",
		CurrAddName:           "коп.",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"UAH": {
		NumericCode:           980,
		AlphaCode:             "UAH",
		MinorUnit:             2,
		CurrencyIsoName:       "Hryvnia",
		CurrencySymbol:        "₴",
		CurrBasicName:         "грн",
		CurrAddName:           "коп.",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"BHD": {
		NumericCode:           48,
		AlphaCode:             "BHD",
		MinorUnit:             3,
		CurrencyIsoName:       "Bahraini Dinar",
		CurrencySymbol:        "BD",
		CurrBasicName:         "BD",
		CurrAddName:           "fils",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"JOD": {
		NumericCode:           400,
		AlphaCode:             "JOD",
		MinorUnit:             3,
		CurrencyIsoName:       "Jordanian Dinar",
		CurrencySymbol:        "JD",
		CurrBasicName:         "JD",
		CurrAddName:           "qirsh",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"KWD": {
		NumericCode:           414,
		AlphaCode:             "KWD",
		MinorUnit:             3,
		CurrencyIsoName:       "Kuwaiti Dinar",
		CurrencySymbol:        "KD",
		CurrBasicName:         "KD",
		CurrAddName:           "fils",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"OMR": {
		NumericCode:           512,
		AlphaCode:             "OMR",
		MinorUnit:             3,
		CurrencyIsoName:       "Omani Rial",
		CurrencySymbol:        "OMR",
		CurrBasicName:         "OMR",
		CurrAddName:           "baisa",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
	"TND": {
		NumericCode:           788,
		AlphaCode:             "TND",
		MinorUnit:             3,
		CurrencyIsoName:       "Tunisian Dinar",
		CurrencySymbol:        "DT",
		CurrBasicName:         "DT",
		CurrAddName:           "millime",
		ShowAdd:               true,
		ShowCurrencyBasicName: true,
	},
}

// CurrencyProfileByCode returns canonical pilot currency profile by ISO 4217 alpha code.
func CurrencyProfileByCode(code string) (CurrencyProfile, bool) {
	profile, ok := currencyProfilesByAlpha[strings.ToUpper(strings.TrimSpace(code))]
	return profile, ok
}

// ValidateCurrencyCode normalizes and validates ISO 4217 alpha code supported by the pilot profile catalog.
func ValidateCurrencyCode(code string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	profile, ok := CurrencyProfileByCode(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported currency code %q", normalized)
	}
	return profile.AlphaCode, nil
}

// CurrencyMinorUnit returns minor unit precision for supported ISO 4217 alpha code.
func CurrencyMinorUnit(code string) (int, error) {
	profile, ok := CurrencyProfileByCode(code)
	if !ok {
		return 0, fmt.Errorf("unsupported currency code %q", strings.ToUpper(strings.TrimSpace(code)))
	}
	return profile.MinorUnit, nil
}
