package contracts_test

import (
	"encoding/json"
	"testing"

	"cloud-backend/internal/cloudsync/contracts"
)

func TestValidateMasterDataPayloadCurrencies(t *testing.T) {
	payload := json.RawMessage(`{
		"currencies":[
			{
				"currency_code":643,
				"currency_alpha_code":"RUB",
				"minor_unit":2,
				"currency_iso_name":"Russian Ruble",
				"currency_symbol":"₽",
				"curr_basic_name":"р",
				"curr_add_name":"коп.",
				"show_add":true,
				"show_currency_basic_name":true
			},
			{
				"currency_code":414,
				"currency_alpha_code":"KWD",
				"minor_unit":3,
				"currency_iso_name":"Kuwaiti Dinar",
				"currency_symbol":"KD",
				"curr_basic_name":"KD",
				"curr_add_name":"fils",
				"show_add":true,
				"show_currency_basic_name":true
			},
			{
				"currency_code":990,
				"currency_alpha_code":"CLF",
				"minor_unit":4,
				"currency_iso_name":"Unidad de Fomento",
				"currency_symbol":"CLF",
				"curr_basic_name":"clf",
				"curr_add_name":"minor",
				"show_add":true,
				"show_currency_basic_name":true
			}
		]
	}`)
	if err := contracts.ValidateMasterDataPayload(contracts.MasterDataStreamCurrencies, payload); err != nil {
		t.Fatalf("expected valid currencies payload, got %v", err)
	}
}

func TestValidateMasterDataPayloadCurrenciesRejectsDuplicateCode(t *testing.T) {
	payload := json.RawMessage(`{
		"currencies":[
			{
				"currency_code":643,
				"currency_alpha_code":"RUB",
				"minor_unit":2,
				"currency_iso_name":"Russian Ruble",
				"currency_symbol":"₽",
				"curr_basic_name":"р",
				"curr_add_name":"коп.",
				"show_add":true,
				"show_currency_basic_name":true
			},
			{
				"currency_code":643,
				"currency_alpha_code":"USD",
				"minor_unit":2,
				"currency_iso_name":"US Dollar",
				"currency_symbol":"$",
				"curr_basic_name":"$",
				"curr_add_name":"¢",
				"show_add":true,
				"show_currency_basic_name":true
			}
		]
	}`)
	if err := contracts.ValidateMasterDataPayload(contracts.MasterDataStreamCurrencies, payload); err == nil {
		t.Fatal("expected duplicate currency_code to be rejected")
	}
}

func TestCanonicalActiveCurrencyProfilesContainsSEA(t *testing.T) {
	profiles := contracts.CanonicalActiveCurrencyProfiles()
	if len(profiles) < 100 {
		t.Fatalf("expected broad ISO catalog, got only %d profiles", len(profiles))
	}
	required := map[string]int{
		"IDR": 2,
		"THB": 2,
		"VND": 0,
		"MYR": 2,
		"SGD": 2,
		"PHP": 2,
	}
	found := make(map[string]int, len(profiles))
	for _, profile := range profiles {
		found[profile.CurrencyAlphaCode] = profile.MinorUnit
	}
	for code, expectedMinorUnit := range required {
		minorUnit, ok := found[code]
		if !ok {
			t.Fatalf("expected %s in canonical active currency catalog", code)
		}
		if minorUnit != expectedMinorUnit {
			t.Fatalf("unexpected minor_unit for %s: got %d want %d", code, minorUnit, expectedMinorUnit)
		}
	}
}
