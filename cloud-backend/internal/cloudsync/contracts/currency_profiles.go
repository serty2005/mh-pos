package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	iso4217 "github.com/JohannesJHN/iso4217"
)

// CurrencyProfile задает canonical Cloud currency template для pilot provisioning.
type CurrencyProfile struct {
	CurrencyCode          int    `json:"currency_code"`
	CurrencyAlphaCode     string `json:"currency_alpha_code"`
	MinorUnit             int    `json:"minor_unit"`
	CurrencyIsoName       string `json:"currency_iso_name"`
	CurrencySymbol        string `json:"currency_symbol"`
	CurrBasicName         string `json:"curr_basic_name"`
	CurrAddName           string `json:"curr_add_name"`
	ShowAdd               bool   `json:"show_add"`
	ShowCurrencyBasicName bool   `json:"show_currency_basic_name"`
}

type currencyProfilesPayload struct {
	Currencies []CurrencyProfile `json:"currencies"`
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

var canonicalActiveCurrencyProfiles = buildCanonicalActiveCurrencyProfiles()

// CanonicalActiveCurrencyProfiles возвращает deterministic snapshot активного ISO 4217 cloud catalog.
func CanonicalActiveCurrencyProfiles() []CurrencyProfile {
	return slices.Clone(canonicalActiveCurrencyProfiles)
}

// ValidateMasterDataPayload проверяет stream-specific payload constraints.
func ValidateMasterDataPayload(streamName string, payload json.RawMessage) error {
	switch strings.TrimSpace(streamName) {
	case MasterDataStreamCurrencies:
		return validateCurrenciesPayload(payload)
	case MasterDataStreamRestaurants:
		return validateMasterDataObjectPayload(streamName, payload, []string{"node_device_id", "restaurant_id", "sync_mode", "checkpoint_token", "cloud_version", "cloud_updated_at", "restaurants"}, []string{"restaurants"})
	case MasterDataStreamDevices:
		return validateMasterDataObjectPayload(streamName, payload, []string{"node_device_id", "restaurant_id", "sync_mode", "checkpoint_token", "cloud_version", "cloud_updated_at", "devices"}, []string{"devices"})
	case MasterDataStreamStaff:
		return validateMasterDataObjectPayload(streamName, payload, []string{"node_device_id", "restaurant_id", "sync_mode", "checkpoint_token", "cloud_version", "cloud_updated_at", "roles", "employees"}, []string{"roles", "employees"})
	case MasterDataStreamFloor:
		return validateMasterDataObjectPayload(streamName, payload, []string{"node_device_id", "restaurant_id", "sync_mode", "checkpoint_token", "cloud_version", "cloud_updated_at", "halls", "tables"}, []string{"halls", "tables"})
	case MasterDataStreamCatalog:
		return validateMasterDataObjectPayload(streamName, payload, []string{"node_device_id", "restaurant_id", "sync_mode", "checkpoint_token", "cloud_version", "cloud_updated_at", "catalog_items", "folders", "folder_parameters", "tags", "item_tags", "modifier_groups", "modifier_options", "modifier_bindings", "menu_item_modifier_groups"}, []string{"catalog_items", "folders", "folder_parameters", "tags", "item_tags", "modifier_groups", "modifier_options", "modifier_bindings", "menu_item_modifier_groups"})
	case MasterDataStreamMenu:
		return validateMasterDataObjectPayload(streamName, payload, []string{"node_device_id", "restaurant_id", "sync_mode", "checkpoint_token", "cloud_version", "cloud_updated_at", "menu_items"}, []string{"menu_items"})
	case MasterDataStreamPricing:
		return validateMasterDataObjectPayload(streamName, payload, []string{"node_device_id", "restaurant_id", "sync_mode", "checkpoint_token", "cloud_version", "cloud_updated_at", "tax_profiles", "tax_rules", "service_charge_rules", "pricing_policies"}, []string{"tax_profiles", "tax_rules", "service_charge_rules", "pricing_policies"})
	default:
		return nil
	}
}

func validateCurrenciesPayload(payload json.RawMessage) error {
	var body currencyProfilesPayload
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return fmt.Errorf("%w: currencies payload must be valid JSON object", ErrInvalidEnvelope)
	}
	if len(body.Currencies) == 0 {
		return fmt.Errorf("%w: currencies payload must contain at least one item", ErrInvalidEnvelope)
	}
	seenNumeric := make(map[int]struct{}, len(body.Currencies))
	seenAlpha := make(map[string]struct{}, len(body.Currencies))
	for _, item := range body.Currencies {
		if item.CurrencyCode <= 0 {
			return fmt.Errorf("%w: currency_code must be positive", ErrInvalidEnvelope)
		}
		alpha := strings.ToUpper(strings.TrimSpace(item.CurrencyAlphaCode))
		if len(alpha) != 3 {
			return fmt.Errorf("%w: currency_alpha_code must be ISO 4217 alpha-3", ErrInvalidEnvelope)
		}
		if item.MinorUnit < 0 || item.MinorUnit > 4 {
			return fmt.Errorf("%w: minor_unit must be between 0 and 4", ErrInvalidEnvelope)
		}
		if strings.TrimSpace(item.CurrencyIsoName) == "" ||
			strings.TrimSpace(item.CurrencySymbol) == "" ||
			strings.TrimSpace(item.CurrBasicName) == "" ||
			strings.TrimSpace(item.CurrAddName) == "" {
			return fmt.Errorf("%w: currency names and symbols are required", ErrInvalidEnvelope)
		}
		if _, ok := seenNumeric[item.CurrencyCode]; ok {
			return fmt.Errorf("%w: duplicate currency_code %d", ErrInvalidEnvelope, item.CurrencyCode)
		}
		if _, ok := seenAlpha[alpha]; ok {
			return fmt.Errorf("%w: duplicate currency_alpha_code %q", ErrInvalidEnvelope, alpha)
		}
		seenNumeric[item.CurrencyCode] = struct{}{}
		seenAlpha[alpha] = struct{}{}
	}
	return nil
}

func validateMasterDataObjectPayload(streamName string, payload json.RawMessage, allowedKeys, dataKeys []string) error {
	var body map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return fmt.Errorf("%w: %s payload must be valid JSON object", ErrInvalidEnvelope, streamName)
	}
	if len(body) == 0 {
		return fmt.Errorf("%w: %s payload must not be empty", ErrInvalidEnvelope, streamName)
	}
	allowed := make(map[string]struct{}, len(allowedKeys))
	for _, key := range allowedKeys {
		allowed[key] = struct{}{}
	}
	for key := range body {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("%w: unsupported %s payload field %q", ErrInvalidEnvelope, streamName, key)
		}
	}
	hasData := false
	for _, key := range dataKeys {
		if raw, ok := body[key]; ok && string(raw) != "null" {
			if len(raw) == 0 {
				continue
			}
			var items []json.RawMessage
			if err := json.Unmarshal(raw, &items); err != nil {
				return fmt.Errorf("%w: %s.%s must be an array", ErrInvalidEnvelope, streamName, key)
			}
			hasData = true
		}
	}
	if hasData {
		return nil
	}
	return fmt.Errorf("%w: %s payload must include supported stream data", ErrInvalidEnvelope, streamName)
}

func buildCanonicalActiveCurrencyProfiles() []CurrencyProfile {
	active := iso4217.AllActive()
	profilesByAlpha := make(map[string]CurrencyProfile, len(active))
	for _, currency := range active {
		alpha := strings.ToUpper(strings.TrimSpace(currency.Alpha3))
		if len(alpha) != 3 || currency.Numeric <= 0 || currency.MinorUnits < 0 || currency.MinorUnits > 4 {
			continue
		}
		profile := CurrencyProfile{
			CurrencyCode:          currency.Numeric,
			CurrencyAlphaCode:     alpha,
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
		profilesByAlpha[alpha] = profile
	}

	keys := slices.Collect(maps.Keys(profilesByAlpha))
	slices.Sort(keys)
	profiles := make([]CurrencyProfile, 0, len(keys))
	for _, alpha := range keys {
		profiles = append(profiles, profilesByAlpha[alpha])
	}
	return profiles
}

func init() {
	if len(canonicalActiveCurrencyProfiles) == 0 {
		panic("active ISO 4217 currency catalog is empty")
	}
	for _, profile := range canonicalActiveCurrencyProfiles {
		if profile.CurrencyCode <= 0 {
			panic(fmt.Sprintf("invalid currency code for %s", profile.CurrencyAlphaCode))
		}
		if profile.MinorUnit < 0 || profile.MinorUnit > 4 {
			panic(fmt.Sprintf("invalid minor unit for %s: %s", profile.CurrencyAlphaCode, strconv.Itoa(profile.MinorUnit)))
		}
	}
}
