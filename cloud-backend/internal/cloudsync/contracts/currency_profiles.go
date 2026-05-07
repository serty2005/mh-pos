package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CurrencyProfile defines canonical Cloud currency template used for pilot provisioning.
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

// ValidateMasterDataPayload verifies stream-specific payload constraints.
func ValidateMasterDataPayload(streamName string, payload json.RawMessage) error {
	if strings.TrimSpace(streamName) != MasterDataStreamCurrencies {
		return nil
	}
	var body currencyProfilesPayload
	if err := json.Unmarshal(payload, &body); err != nil {
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
		if item.MinorUnit < 0 || item.MinorUnit > 3 {
			return fmt.Errorf("%w: minor_unit must be between 0 and 3", ErrInvalidEnvelope)
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
