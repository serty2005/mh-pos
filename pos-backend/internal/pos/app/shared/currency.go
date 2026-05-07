package shared

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	iso4217 "github.com/JohannesJHN/iso4217"
)

// CurrencyProfile описывает canonical metadata валюты для runtime-валидации и UI-форматирования.
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

type currencyDisplayOverride struct {
	symbol        string
	basicName     string
	addName       string
	showAdd       bool
	showBasicName bool
}

var currencyDisplayOverrides = map[string]currencyDisplayOverride{
	"RUB": {symbol: "₽", basicName: "р", addName: "коп.", showAdd: true, showBasicName: true},
	"USD": {symbol: "$", basicName: "$", addName: "¢", showAdd: true, showBasicName: true},
	"EUR": {symbol: "€", basicName: "€", addName: "c", showAdd: true, showBasicName: true},
	"KZT": {symbol: "₸", basicName: "₸", addName: "тиын", showAdd: true, showBasicName: true},
	"BYN": {symbol: "Br", basicName: "Br", addName: "коп.", showAdd: true, showBasicName: true},
	"UAH": {symbol: "₴", basicName: "грн", addName: "коп.", showAdd: true, showBasicName: true},
	"BHD": {symbol: "BD", basicName: "BD", addName: "fils", showAdd: true, showBasicName: true},
	"JOD": {symbol: "JD", basicName: "JD", addName: "qirsh", showAdd: true, showBasicName: true},
	"KWD": {symbol: "KD", basicName: "KD", addName: "fils", showAdd: true, showBasicName: true},
	"OMR": {symbol: "OMR", basicName: "OMR", addName: "baisa", showAdd: true, showBasicName: true},
	"TND": {symbol: "DT", basicName: "DT", addName: "millime", showAdd: true, showBasicName: true},
	"VND": {symbol: "₫", basicName: "₫", addName: "xu", showAdd: false, showBasicName: true},
	"THB": {symbol: "฿", basicName: "฿", addName: "satang", showAdd: true, showBasicName: true},
	"IDR": {symbol: "Rp", basicName: "Rp", addName: "sen", showAdd: true, showBasicName: true},
}

var currencyProfilesByAlpha = buildCurrencyProfilesByAlpha()
var sortedCurrencyProfiles = buildSortedCurrencyProfiles(currencyProfilesByAlpha)

// CurrencyProfileByCode возвращает canonical профиль валюты по ISO 4217 alpha code.
func CurrencyProfileByCode(code string) (CurrencyProfile, bool) {
	profile, ok := currencyProfilesByAlpha[strings.ToUpper(strings.TrimSpace(code))]
	return profile, ok
}

// CurrencyProfiles возвращает deterministic snapshot active ISO 4217 каталога, поддержанного runtime.
func CurrencyProfiles() []CurrencyProfile {
	return slices.Clone(sortedCurrencyProfiles)
}

// ValidateCurrencyCode нормализует и валидирует ISO 4217 alpha code из active каталога.
func ValidateCurrencyCode(code string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	profile, ok := CurrencyProfileByCode(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported currency code %q", normalized)
	}
	return profile.AlphaCode, nil
}

// CurrencyMinorUnit возвращает minor unit precision для поддержанного ISO 4217 alpha code.
func CurrencyMinorUnit(code string) (int, error) {
	profile, ok := CurrencyProfileByCode(code)
	if !ok {
		return 0, fmt.Errorf("unsupported currency code %q", strings.ToUpper(strings.TrimSpace(code)))
	}
	return profile.MinorUnit, nil
}

func buildCurrencyProfilesByAlpha() map[string]CurrencyProfile {
	active := iso4217.AllActive()
	profiles := make(map[string]CurrencyProfile, len(active))
	for _, currency := range active {
		alpha := strings.ToUpper(strings.TrimSpace(currency.Alpha3))
		if len(alpha) != 3 || currency.Numeric <= 0 || currency.MinorUnits < 0 || currency.MinorUnits > 4 {
			continue
		}
		profile := CurrencyProfile{
			NumericCode:           currency.Numeric,
			AlphaCode:             alpha,
			MinorUnit:             currency.MinorUnits,
			CurrencyIsoName:       strings.TrimSpace(currency.Name),
			CurrencySymbol:        alpha,
			CurrBasicName:         strings.ToLower(alpha),
			CurrAddName:           "minor",
			ShowAdd:               currency.MinorUnits > 0,
			ShowCurrencyBasicName: true,
		}
		if override, ok := currencyDisplayOverrides[alpha]; ok {
			profile.CurrencySymbol = override.symbol
			profile.CurrBasicName = override.basicName
			profile.CurrAddName = override.addName
			profile.ShowAdd = override.showAdd
			profile.ShowCurrencyBasicName = override.showBasicName
		}
		profiles[alpha] = profile
	}
	return profiles
}

func buildSortedCurrencyProfiles(source map[string]CurrencyProfile) []CurrencyProfile {
	keys := slices.Collect(maps.Keys(source))
	slices.Sort(keys)
	out := make([]CurrencyProfile, 0, len(keys))
	for _, key := range keys {
		out = append(out, source[key])
	}
	return out
}

func init() {
	if len(currencyProfilesByAlpha) == 0 {
		panic("active ISO 4217 currency catalog is empty")
	}
	for alpha, profile := range currencyProfilesByAlpha {
		if len(alpha) != 3 {
			panic(fmt.Sprintf("invalid currency alpha code %q", alpha))
		}
		if profile.NumericCode <= 0 {
			panic(fmt.Sprintf("invalid currency numeric code for %s", alpha))
		}
		if profile.MinorUnit < 0 || profile.MinorUnit > 4 {
			panic(fmt.Sprintf("invalid minor unit for %s: %s", alpha, strconv.Itoa(profile.MinorUnit)))
		}
	}
}
